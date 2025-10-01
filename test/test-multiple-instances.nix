{ lib, pkgs, config, ... }:

{
  imports = [ ../modules/ngit-relay-module.nix ];

  services.ngitRelay = {
    enable = true;
    imageFromFlake = /fake/path/to/image;

    instances = {
      main = {
        name = "main-relay";
        ports = [ "127.0.0.1:8081:8081" ];
        environment = {
          NGIT_DOMAIN = "main.example.com";
          NGIT_OWNER_NPUB = "npub1main123";
        };
      };

      test = {
        name = "test-relay";
        ports = [ "127.0.0.1:8082:8081" ];
        environment = {
          NGIT_DOMAIN = "test.example.com";
          NGIT_OWNER_NPUB = "npub1test456";
        };
        DataDir = "/var/lib/ngit-relay-test";
        LogDir = "/var/log/ngit-relay-test";
      };
    };

  };
}
