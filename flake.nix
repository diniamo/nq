{
  description = "A set of convenience programs for configuration switching on NixOS";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default-linux";
  };

  outputs = {nixpkgs, systems, self, ...}: let
    eachSystem = callback: nixpkgs.lib.genAttrs (import systems) (system: callback nixpkgs.legacyPackages.${system});
  in {
    devShells = eachSystem (pkgs: {
      default = with pkgs; mkShellNoCC {
        packages = [
          go
          gopls

          nix-output-monitor
          nvd
        ];
      };
    });

    packages = eachSystem (pkgs: let
      package = pkgs.callPackage ./package.nix {
        commit = self.shortRev or "dirty";
      };
    in {
      default = package;
      swich = package;
    });
  };
}
