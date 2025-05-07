{
  description =
    "ngit-relay - a nostr-permissiond git server with embedded relay";

  inputs = {
    # Use the latest stable version of Nixpkgs
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }: {
    # Define the development shell
    devShells.x86_64-linux.default =
      nixpkgs.legacyPackages.x86_64-linux.mkShell {
        buildInputs = [ nixpkgs.legacyPackages.x86_64-linux.go ];
      };
  };
}
