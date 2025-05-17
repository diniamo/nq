# Mostly taken from https://github.com/NixOS/nixpkgs/blob/master/pkgs/os-specific/linux/nixos-rebuild/nixos-rebuild.sh
let
  flake = builtins.getFlake "@flake@";
  configuration = flake.nixosConfigurations.@configuration@;
  
  motd = ''


    Hello and welcome to the NixOS configuration
        nixosConfigurations.@configuration@
        in @flake@

    The following is loaded into nix repl's scope:

        - @blue@config@reset@   All option values
        - @blue@options@reset@  Option data and metadata
        - @blue@pkgs@reset@     Nixpkgs package set
        - @blue@lib@reset@      Nixpkgs library functions
        - other module arguments

        - @blue@flake@reset@    Flake outputs, inputs and source info of $flake

    Use tab completion to browse around @blue@config@reset@.

    Use @bold@:r@reset@ to @bold@reload@reset@ everything after making a change in the flake.
        (assuming @flake@ is a mutable flake ref)

    See @bold@:?@reset@ for more repl commands.

    @attention@warning:@reset@ rebuild --repl does not currently enforce pure evaluation.
  '';
  
  scope =
    assert configuration._type or null == "configuration";
    assert configuration.class or "nixos" == "nixos";
    configuration._module.args //
    configuration._module.specialArgs // {
      inherit (configuration) config options;
      lib = configuration.lib or configuration.pkgs.lib;
      inherit flake;
    };
in builtins.seq scope builtins.trace motd scope
