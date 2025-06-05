{
  description = "A set of convenience programs for NixOS";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    systems.url = "github:nix-systems/default-linux";

    dix = {
      url = "github:bloxx12/dix";
      inputs = {
        nixpkgs.follows = "nixpkgs";
        systems.follows = "systems";
      };
    };
  };

  outputs = {flake-parts, systems, self, ...}@inputs: flake-parts.lib.mkFlake {inherit inputs;} {
    systems = import systems;

    perSystem = {pkgs, inputs', ...}: {
      devShells.default = with pkgs; mkShellNoCC {
        packages = [
          go
          gopls

          nix-output-monitor
          inputs'.dix.packages.default
        ];
      };

      packages = let
        package = pkgs.callPackage ./package.nix {
          commit = self.shortRev or "dirty";
          dix = inputs'.dix.packages.default;
        };
      in {
        default = package;
        nq = package;
      };
    };
  };
}
