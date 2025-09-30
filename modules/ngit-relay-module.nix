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

      # single-instance convenience
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

      # DataDir and LogDir (top-level defaults can include ${name})
      DataDir = mkOption {
        type = types.str;
        default = "/var/lib/ngit-relay";
      };
      LogDir = mkOption {
        type = types.str;
        default = "/var/log/ngit-relay";
      };

      # multi-instance API
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
            # per-instance overrides (null to inherit)
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
        # leave DataDir/LogDir null so they fall back to top-level defaults which may use ${name}
        DataDir = null;
        LogDir = null;
      };
    };

    sanitizeName = name: builtins.replaceStrings [ " " "/" ] [ "-" "-" ] name;

    mkEnv = env:
      lib.foldl' (m: k: let v = env.${k}; in m // { "${k}" = toString v; }) { }
      (builtins.attrNames env);

    # helper: expand a template default that may contain ${name}
    expandWithName = str: name:
      let containsTemplate = builtins.match ".*\\$\\{name\\}.*" str != null;
      in if containsTemplate then
        lib.replaceStrings [ "${name}" ] [ name ] str
      else
        str;

    makeContainer = name: inst:
      let
        unit = "ngit-relay-" + name;
        containerName = unit;
        imageRef = if config.services.ngitRelay.imageFromFlake != null then
          config.services.ngitRelay.imageFromFlake
        else
          throw
          "ngitRelay: set services.ngitRelay.imageFromFlake to the flake-built image path (inputs.<flake>.packages.<system>.image)";
        mergedEnv = lib.recursiveUpdate defaults (inst.environment or { });
        envMap = mkEnv mergedEnv;

        # resolve DataDir and LogDir (instance override -> top-level -> defaults)
        topDataDir = config.services.ngitRelay.DataDir;
        topLogDir = config.services.ngitRelay.LogDir;

        dataDirRaw = if inst.DataDir != null then inst.DataDir else topDataDir;
        logDirRaw = if inst.LogDir != null then inst.LogDir else topLogDir;

        dataDir = expandWithName dataDirRaw name;
        logDir = expandWithName logDirRaw name;

        dockerVolumes = [
          "${dataDir}/repos:/srv/ngit-relay/repos:rw"
          "${dataDir}/blossom:/srv/ngit-relay/blossom:rw"
          "${dataDir}/relay-db:/srv/ngit-relay/relay-db:rw"
          "${logDir}:/var/log/ngit-relay:rw"
        ];

      in {
        # Create host-side dirs for declared volumes
        system.activationScripts."ngit-relay-mkdirs-${name}" = ''
          ${pkgs.mkdir}/bin/mkdir -p ${
            lib.concatStringsSep " "
            (map (p: builtins.elemAt (lib.splitString ":" p) 0) dockerVolumes)
          }
          ${pkgs.chown}/bin/chown -R root:root ${
            lib.concatStringsSep " "
            (map (p: builtins.elemAt (lib.splitString ":" p) 0) dockerVolumes)
          } || true
        '';

        # Declare the container via services.docker-containers
        services.docker-containers = lib.genAttrs [ unit ] (name: {
          image = imageRef;
          containerName = containerName;
          restartPolicy = inst.restart or "unless-stopped";
          ports = inst.ports;
          volumes = dockerVolumes;
          environment = envMap;
        });
      };

    containers = builtins.mapAttrsToList (name: inst:
      makeContainer
      (sanitizeName (if inst.name != null then inst.name else name)) inst)
      instances;
    mergedServices =
      lib.foldl' (acc: x: acc // x.services.docker-containers) { } containers;
    activationScripts =
      lib.foldl' (acc: x: acc // x.system.activationScripts) { } containers;

  in {
    services.docker.enable = true;

    # merge generated docker-containers entries
    services.docker-containers = mergedServices;

    # merge activation scripts to create dirs
    system.activationScripts = activationScripts;
  });
}
