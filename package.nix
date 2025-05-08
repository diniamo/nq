{buildGoModule, commit, lib, makeBinaryWrapper, nix-output-monitor, nvd}:
buildGoModule {
  pname = "rebuild";
  version = "0-unstable-${commit}";

  src = lib.cleanSource ./.;
  vendorHash = "sha256-M3w80FoM5ak5YtuW5PaB4t47unTC/OXS99Sk3/C7dtg=";

  nativeBuildInputs = [makeBinaryWrapper];

  postFixup = ''
    wrapProgram $out/bin/rebuild \
      --prefix PATH : ${lib.makeBinPath [nix-output-monitor nvd]}
  '';

  meta = {
    description = "A convenience program for rebuilding on NixOS";
    homepage = "https://github.com/diniamo/rebuild";
    license = lib.licenses.eupl12;
    platforms = lib.platforms.linux;
    maintainers = [lib.maintainers.diniamo];
    mainProgram = "rebuild";
  };
}
