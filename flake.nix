{
  description =
    "ngit-relay flake: builds image and provides NixOS service module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { inherit system; };
      in {
        packages = {
          # Build the image from the repository Dockerfile (src/Dockerfile).
          image = pkgs.dockerTools.buildLayeredImage {
            name = "ngit-relay-image";
            # Path to Dockerfile context
            directory = ./.;
            dockerFile = ./src/Dockerfile;
            # optional: set extra build args if needed
            # buildArgs = { ...; };
          };
        };

        nixosModules.default =
          (import ./modules/ngit-relay-module.nix { inherit pkgs; });
      });
}
