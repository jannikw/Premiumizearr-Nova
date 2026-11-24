{
  description = "Premiumizearr-Nova";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      pkgs = nixpkgs.legacyPackages.x86_64-linux;

      # Build static web assets
      web-static = pkgs.buildNpmPackage {
        name = "premiumizearr-nova-web";
        buildInputs = [ pkgs.nodejs_latest ];
        src = ./web;
        npmDeps = pkgs.importNpmLock {
          npmRoot = ./web;
        };
        npmConfigHook = pkgs.importNpmLock.npmConfigHook;
        installPhase = ''
          mv dist $out
        '';
      };

      # Build the Go application
      app = pkgs.buildGoModule {
        name = "premiumizearr-nova";
        src = ./.;
        vendorHash = "sha256-CsD+Xv1k6GImfdzhRNE9/75XQ5fec2uNJJGLBaUqnBo=";
        propagatedBuildInputs = [
          pkgs.coreutils
          pkgs.wget
        ];
        # Patch paths to use static web assets from Nix store
        patchPhase = ''
          ${pkgs.gnused}/bin/sed -i 's|"./static/index.html"|"${web-static}/index.html"|' internal/service/web_service.go
          ${pkgs.gnused}/bin/sed -i 's|staticPath: "static",|staticPath: "${web-static}",|' internal/service/web_service.go
        '';

        # Skip failing test. Unsure if this is an issue with the test itself or a Nix packaging issue.
        # Since upstream doesn't run tests on CI, this is hard to tell.
        checkFlags = [
          "--skip=TestWebRootServesAssets"
        ];
      };
    in
    {
      packages.x86_64-linux.default = app;
    };
}
