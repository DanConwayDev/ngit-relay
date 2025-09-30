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
        image = pkgs.dockerTools.buildImage {
          name = "ngit-relay-image";
          tag = "latest";
          created = "now";
          copyToRoot = pkgs.buildEnv {
            name = "ngit-relay-root";
            paths = [ pkgs.dockerTools.binSh pkgs.dockerTools.caCertificates ];
          };
          config = {
            Cmd = [ "/bin/sh" ];
            WorkingDir = "/app";
          };
        };
      in {
        packages.${system} = {
          image = image;
          default = image;
        };

        defaultPackage = image;

        devShells.${system}.default =
          pkgs.mkShell { buildInputs = [ pkgs.go ]; };

      }) // {
        nixosModules = { ngit-relay = import ./modules/ngit-relay-module.nix; };
      };
}
