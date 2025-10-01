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
        imageRef = if config.services.ngitRelay.imageFromFlake != null then
          config.services.ngitRelay.imageFromFlake
        else
          throw
          "ngitRelay: set services.ngitRelay.imageFromFlake to the flake-built image path (inputs.<flake>.packages.<system>.image)";
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
        restartPolicy = inst.restart or "unless-stopped";
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
        binds = c.binds;
        env = c.env;
        restartPolicy = c.restartPolicy;
        ports = c.ports;
      };
    }) containersList);

    activationScripts = lib.listToAttrs (map (container: {
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

  in {
    virtualisation.oci-containers.enable = true;
    virtualisation.oci-containers.containers = dockerContainers;
    system.activationScripts = activationScripts;
  });
}
