{ lib, pkgs, config, ... }:

{
  imports = [ ../modules/ngit-relay-module.nix ];

  services.ngitRelay = {
    enable = true;
    imageFromFlake = /fake/path/to/image;
    environment = {
      NGIT_DOMAIN = "test.example.com";
      NGIT_OWNER_NPUB = "npub1test123";
    };
  };
}
