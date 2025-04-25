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
        vendorHash = "sha256-1Ka6FxsUlqqD3rksXCO57KdJ2Ryzc78XBlRf/nSfDfA=";
        propagatedBuildInputs = [ pkgs.coreutils pkgs.wget ];
        # Patch paths to use static web assets from Nix store
        patchPhase = ''
          ${pkgs.gnused}/bin/sed -i 's|"./static/index.html"|"${web-static}/index.html"|' internal/service/web_service.go
          ${pkgs.gnused}/bin/sed -i 's|staticPath: "static",|staticPath: "${web-static}",|' internal/service/web_service.go
        '';
      };
    in
    {
      packages.x86_64-linux.default = app;
    };
}
