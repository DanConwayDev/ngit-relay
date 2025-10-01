{ lib, pkgs, config, ... }:

{
  imports = [ ../modules/ngit-relay-module.nix ];

  services.ngitRelay = {
    enable = true;
    # This should fail because imageFromFlake is not set
  };
}
