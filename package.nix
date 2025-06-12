{buildGoModule, commit, lib, makeBinaryWrapper, nix-output-monitor, dix}:
buildGoModule {
  pname = "nq";
  version = "0-unstable-${commit}";

  src = lib.cleanSource ./.;
  
  vendorHash = "sha256-xSs/7ROM9mUejRSsQdX3xOFbHssUbMR5++nagEPsmQU=";
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
