{
  description = "notes-api — demo backend for Russel";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in {
      packages.${system}.default = pkgs.buildGoModule {
        pname = "notes-api";
        version = "0.1.0";
        src = ./.;
        vendorHash = null;
      };

      devShells.${system}.default = pkgs.mkShell {
        buildInputs = [ pkgs.go ];
      };
    };
}
