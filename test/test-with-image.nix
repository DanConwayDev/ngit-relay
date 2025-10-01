{ lib, pkgs, config, ... }:

{
  imports = [ ../modules/ngit-relay-module.nix ];

  # Minimal system configuration for testing
  system.stateVersion = "24.05";
  fileSystems."/" = {
    device = "/dev/disk/by-label/nixos";
    fsType = "ext4";
  };
  boot.loader.grub.device = "/dev/sda";

  services.ngitRelay = {
    enable = true;
    imageFromFlake = /fake/path/to/image;
    environment = {
      NGIT_DOMAIN = "test.example.com";
      NGIT_OWNER_NPUB = "npub1test123";
    };
  };
}
