{ lib, pkgs, config, ... }:

with lib;

let
  defaults = {
    NGIT_DOMAIN = "localhost";
    NGIT_RELAY_NAME = "local test ngit-relay instance";
    NGIT_RELAY_DESCRIPTION =
      "instance of ngit-relay, a nostr-permissioned git server with relay and blossom server";
    NGIT_OWNER_NPUB = "";
    NGIT_BLOSSOM_MAX_FILE_SIZE_MB = "100";
    NGIT_BLOSSOM_MAX_CAPACITY_GB = "50";
    NGIT_LOG_DIR = "/var/log/ngit-relay";
    NGIT_LOG_LEVEL = "DEBUG";
    NGIT_LOG_MAX_SIZE_MB = "20";
    NGIT_LOG_MAX_BACKUPS = "10";
    NGIT_LOG_MAX_AGE_DAYS = "30";
    NGIT_INTERNAL_RELAY_PORT_FOR_SSL_PROXY = "8081";
    NGINX_ENTRYPOINTS_WORKER_CONNECTIONS = "2048";
  };

  cfg = config.services.ngitRelay;
in {
  options = {
    services.ngitRelay = {
      enable = mkOption {
        type = types.bool;
        default = false;
      };

      name = mkOption {
        type = types.str;
        default = "ngit-relay";
      };
      ports = mkOption {
        type = types.listOf types.str;
        default = [ "127.0.0.1:8081:8081" ];
      };
      environment = mkOption {
        type = types.attrsOf types.str;
        default = { };
      };

      DataDir = mkOption {
        type = types.str;
        default = "/var/lib/ngit-relay";
      };
      LogDir = mkOption {
        type = types.str;
        default = "/var/log/ngit-relay";
      };

      instances = mkOption {
        type = types.attrsOf (types.submodule {
          options = {
            name = mkOption {
              type = types.nullOr types.str;
              default = null;
            };
            ports = mkOption {
              type = types.listOf types.str;
              default = [ "127.0.0.1:8081:8081" ];
            };
            environment = mkOption {
              type = types.attrsOf types.str;
              default = { };
            };
            restart = mkOption {
              type = types.str;
              default = "unless-stopped";
            };
            DataDir = mkOption {
              type = types.nullOr types.str;
              default = null;
            };
            LogDir = mkOption {
              type = types.nullOr types.str;
              default = null;
            };
          };
        });
        default = { };
      };

      imageFromFlake = mkOption {
        type = types.nullOr types.path;
        default = null;
      };
    };
  };

  config = mkIf config.services.ngitRelay.enable (let
    useInstances =
      builtins.length (builtins.attrNames config.services.ngitRelay.instances)
      > 0;
    instances = if useInstances then
      config.services.ngitRelay.instances
    else {
      default = {
        name = config.services.ngitRelay.name;
        ports = config.services.ngitRelay.ports;
        environment = config.services.ngitRelay.environment;
        restart = "unless-stopped";
        DataDir = null;
        LogDir = null;
      };
    };

    sanitizeName = name: builtins.replaceStrings [ " " "/" ] [ "-" "-" ] name;

    mkEnv = env:
      lib.foldl' (m: k: let v = env.${k}; in m // { "${k}" = toString v; }) { }
      (builtins.attrNames env);

    expandWithName = str: name: str;

    # parse "host:hostPort:containerPort" or "host:containerPort:containerPort" entries into oci-containers port format
    parsePort = p:
      let
        parts = lib.splitString ":" p;
        host = builtins.elemAt parts 0;
        hostPort = builtins.elemAt parts 1;
        containerPort = builtins.elemAt parts 2;
      in {
        hostAddress = host;
        hostPort = builtins.tryEval hostPort // {
          success = true;
          value = hostPort;
        }; # keep as string
        containerPort = containerPort;
      };

    makeContainer = name: inst:
      let
        unit = "ngit-relay-" + name;
        imageTarball = if config.services.ngitRelay.imageFromFlake != null then
          toString config.services.ngitRelay.imageFromFlake
        else
          throw
          "ngitRelay: set services.ngitRelay.imageFromFlake to the flake-built image path (inputs.<flake>.packages.<system>.image)";
        # Use a predictable image name that will be created by the activation script
        imageRef = "ngit-relay-image:latest";
        mergedEnv = lib.recursiveUpdate defaults (inst.environment or { });
        envMap = mkEnv mergedEnv;

        topDataDir = config.services.ngitRelay.DataDir;
        topLogDir = config.services.ngitRelay.LogDir;

        dataDirRaw = if inst.DataDir != null then inst.DataDir else topDataDir;
        logDirRaw = if inst.LogDir != null then inst.LogDir else topLogDir;

        dataDir = expandWithName dataDirRaw name;
        logDir = expandWithName logDirRaw name;

        binds = [
          "${dataDir}/repos:/srv/ngit-relay/repos:rw"
          "${dataDir}/blossom:/srv/ngit-relay/blossom:rw"
          "${dataDir}/relay-db:/srv/ngit-relay/relay-db:rw"
          "${logDir}:/var/log/ngit-relay:rw"
        ];

        # convert simple "host:hostPort:containerPort" strings into oci-containers port maps
        portsParsed = map (p:
          let parts = lib.splitString ":" p;
          in {
            hostAddress = builtins.elemAt parts 0;
            hostPort = builtins.elemAt parts 1;
            containerPort = builtins.elemAt parts 2;
          }) inst.ports;
      in {
        name = unit;
        image = imageRef;
        imageTarball = imageTarball;
        binds = binds;
        env = envMap;
        ports = portsParsed;
        dataDir = dataDir;
        logDir = logDir;
      };

    containersList = lib.mapAttrsToList (name: inst:
      makeContainer
      (sanitizeName (if inst.name != null then inst.name else name)) inst)
      instances;

    dockerContainers = lib.listToAttrs (map (c: {
      name = c.name;
      value = {
        image = c.image;
        volumes = c.binds;
        environment = c.env;
        ports =
          map (p: "${p.hostAddress}:${p.hostPort}:${p.containerPort}") c.ports;
      };
    }) containersList);

    # Create systemd service overrides to add dependencies and handle container lifecycle
    containerServiceOverrides =
      if config.services.ngitRelay.imageFromFlake != null then
        lib.listToAttrs (map (c: {
          name = "docker-${c.name}";
          value = {
            after = [ "ngit-relay-image-loader.service" ];
            requires = [ "ngit-relay-image-loader.service" ];
            serviceConfig = {
              # Override oci-containers scripts to check container existence first
              ExecStartPre = pkgs.writeShellScript "check-and-stop-container" ''
                # Check if container exists before trying to stop it
                if ${pkgs.docker}/bin/docker ps -a --format "{{.Names}}" | grep -q "^docker-${c.name}$"; then
                  echo "Stopping existing container docker-${c.name}"
                  ${pkgs.docker}/bin/docker stop docker-${c.name}
                  ${pkgs.docker}/bin/docker rm -f docker-${c.name}
                else
                  echo "Container docker-${c.name} does not exist, nothing to stop"
                fi
              '';
              ExecStopPost =
                pkgs.writeShellScript "check-and-remove-container" ''
                  # Check if container exists before trying to remove it
                  if ${pkgs.docker}/bin/docker ps -a --format "{{.Names}}" | grep -q "^docker-${c.name}$"; then
                    echo "Removing container docker-${c.name}"
                    ${pkgs.docker}/bin/docker rm -f docker-${c.name}
                  else
                    echo "Container docker-${c.name} does not exist, nothing to remove"
                  fi
                '';
            };
          };
        }) containersList)
      else
        { };

    # Create activation scripts for both directory creation and Docker image loading
    mkdirActivationScripts = lib.listToAttrs (map (container: {
      name = "ngit-relay-mkdirs-${container.name}";
      value = ''
        ${pkgs.coreutils}/bin/mkdir -p ${
          lib.concatStringsSep " "
          (map (p: builtins.elemAt (lib.splitString ":" p) 0) container.binds)
        }
        ${pkgs.coreutils}/bin/chown -R root:root ${
          lib.concatStringsSep " "
          (map (p: builtins.elemAt (lib.splitString ":" p) 0) container.binds)
        } || true
      '';
    }) containersList);

    # Create a systemd service to load the Docker image
    imageLoaderService =
      if config.services.ngitRelay.imageFromFlake != null then {
        "ngit-relay-image-loader" = {
          description = "Load ngit-relay Docker image from tarball";
          wantedBy = [ "multi-user.target" ];
          before = map (c: "docker-${c.name}.service") containersList;
          serviceConfig = {
            Type = "oneshot";
            RemainAfterExit = true;
            ExecStart = pkgs.writeShellScript "load-ngit-relay-image" ''
              set -euo pipefail

              # Load the Docker image from tarball and tag it with a predictable name
              if [ -f "${
                toString config.services.ngitRelay.imageFromFlake
              }" ]; then
                echo "Loading ngit-relay Docker image from ${
                  toString config.services.ngitRelay.imageFromFlake
                }"
                ${pkgs.docker}/bin/docker load < "${
                  toString config.services.ngitRelay.imageFromFlake
                }"
                
                # Get the loaded image ID and tag it with our predictable name
                IMAGE_ID=$(${pkgs.docker}/bin/docker images --format "{{.ID}}" --filter "reference=ngit-relay-image:latest" | head -n1)
                if [ -z "$IMAGE_ID" ]; then
                  # If the image wasn't tagged as ngit-relay-image:latest, find the most recent image
                  IMAGE_ID=$(${pkgs.docker}/bin/docker images --format "{{.ID}}" | head -n1)
                  if [ -n "$IMAGE_ID" ]; then
                    echo "Tagging image $IMAGE_ID as ngit-relay-image:latest"
                    ${pkgs.docker}/bin/docker tag "$IMAGE_ID" ngit-relay-image:latest
                  else
                    echo "Error: No Docker image found after loading tarball"
                    exit 1
                  fi
                fi
                echo "ngit-relay Docker image loaded and tagged as ngit-relay-image:latest"
              else
                echo "Error: Docker image tarball not found at ${
                  toString config.services.ngitRelay.imageFromFlake
                }"
                exit 1
              fi
            '';
          };
          after = [ "docker.service" ];
          requires = [ "docker.service" ];
        };
      } else
        { };

    activationScripts = mkdirActivationScripts;

  in {
    virtualisation.oci-containers.backend = "docker";
    virtualisation.oci-containers.containers = dockerContainers;
    systemd.services = imageLoaderService // containerServiceOverrides;
    system.activationScripts = activationScripts;
  });
}
