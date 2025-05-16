{buildGoModule, commit, lib, makeBinaryWrapper, nix-output-monitor, nvd}:
buildGoModule {
  pname = "swich";
  version = "0-unstable-${commit}";

  src = lib.cleanSource ./.;
  
  vendorHash = "sha256-M3w80FoM5ak5YtuW5PaB4t47unTC/OXS99Sk3/C7dtg=";
  nativeBuildInputs = [makeBinaryWrapper];

  subPackages = [
    "cmd/rebuild"
    "cmd/rollback"
  ];

  postFixup = ''
    wrapProgram $out/bin/rebuild \
      --prefix PATH : ${lib.makeBinPath [nix-output-monitor nvd]}
    wrapProgram $out/bin/rollback \
      --prefix PATH : ${nvd}/bin
  '';

  meta = {
    description = "A set of convenience programs for configuration switching on NixOS";
    homepage = "https://github.com/diniamo/swich";
    license = lib.licenses.eupl12;
    platforms = lib.platforms.linux;
    maintainers = [lib.maintainers.diniamo];
    mainProgram = "rebuild";
  };
}
