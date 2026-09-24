{
  description = "A terminal SQL client for Postgres, MySQL and SQLite";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "1.9.1";
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "qrypad";
          inherit version;

          src = ./.;

          vendorHash = "sha256-r3LUaVnYhgrpoVPWZInL/QiYiJ31vZJcvAGR+d7iCmY=";

          doCheck = false;

          ldflags = [
            "-s"
            "-w"
            "-X github.com/wheelibin/qrypad/internal/constants.Version=${version}"
          ];

          meta = with pkgs.lib; {
            description = "A terminal SQL client for Postgres, MySQL and SQLite";
            homepage = "https://github.com/wheelibin/qrypad";
            license = licenses.mit;
            mainProgram = "qrypad";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go gopls gotools ];
        };
      }
    );
}
