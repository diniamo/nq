{buildGoModule, commit, lib, makeBinaryWrapper, nix-output-monitor, dix}:
buildGoModule {
  pname = "nq";
  version = "0-unstable-${commit}";

  src = lib.cleanSource ./.;
  
  vendorHash = "sha256-M3w80FoM5ak5YtuW5PaB4t47unTC/OXS99Sk3/C7dtg=";
  nativeBuildInputs = [makeBinaryWrapper];

  subPackages = [
    "cmd/rebuild"
    "cmd/rollback"
    "cmd/clean"
  ];

  postFixup = ''
    wrapProgram $out/bin/rebuild \
      --prefix PATH : ${lib.makeBinPath [nix-output-monitor dix]}
    wrapProgram $out/bin/rollback \
      --prefix PATH : ${dix}/bin
  '';

  meta = {
    description = "A set of convenience programs for NixOS";
    homepage = "https://github.com/diniamo/nq";
    license = lib.licenses.eupl12;
    platforms = lib.platforms.linux;
    maintainers = [lib.maintainers.diniamo];
    mainProgram = "rebuild";
  };
}
