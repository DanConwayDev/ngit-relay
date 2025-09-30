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
        packages.${system} = {
          image = image;
          default = image;
        };

        defaultPackage = image;

        devShells.${system}.default =
          pkgs.mkShell { buildInputs = [ pkgs.go ]; };

        # Export module both under nixosModules and as a direct attribute
        nixosModules = { "ngit-relay" = nixosModule; };

        # Some consumers expect a single 'nixosModule' attr — provide that too.
        nixosModule = nixosModule;
      });
}
