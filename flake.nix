{
  description =
    "ngit-relay flake: builds image and provides NixOS service module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        dockerTools = pkgs.dockerTools;
        image = dockerTools.buildImage {
          name = "ngit-relay-image";
          tag = "latest";
          created = "now";
          copyToRoot = pkgs.buildEnv {
            name = "ngit-relay-root";
            paths = [ dockerTools.binSh dockerTools.caCertificates ];
          };
          config = {
            Cmd = [ "/bin/sh" ];
            WorkingDir = "/app";
          };
        };

        nixosModule = import ./modules/ngit-relay-module.nix;
      in {
        packages = {
          image = image;
          default = image;
        };

        defaultPackage = image;

        devShells.default = pkgs.mkShell { buildInputs = [ pkgs.go ]; };
      }) // {
        # Export module at top level, outside of eachDefaultSystem
        nixosModules.ngit-relay = import ./modules/ngit-relay-module.nix;
      };
}
