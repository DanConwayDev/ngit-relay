{ lib, pkgs, config, ... }:

{
  imports = [ ../modules/ngit-relay-module.nix ];

  services.ngitRelay = {
    enable = true;
    imageFromFlake = /fake/path/to/image;
    DataDir = "/custom/data/ngit";
    LogDir = "/custom/logs/ngit";
    ports = [ "0.0.0.0:8080:8081" "127.0.0.1:8443:8443" ];
    environment = {
      NGIT_DOMAIN = "custom.example.com";
      NGIT_RELAY_NAME = "Custom Test Relay";
      NGIT_OWNER_NPUB = "npub1custom789";
      NGIT_LOG_LEVEL = "INFO";
    };
  };
}
