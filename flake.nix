{
  description = "Pushup is a new web framework for Go";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.11";

  outputs = { self, nixpkgs }:
    let
      version = "0.0.1"; # TODO(paulsmith): source from version.go
      forEachSystem = fn:
        nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-darwin" "aarch64-linux" ]
        (system:
          fn {
            pkgs = nixpkgs.legacyPackages.${system};
            inherit system;
          });
    in {
      packages = forEachSystem ({ pkgs, system }: {
        default = pkgs.buildGoModule rec {
          pname = "pushup";
          inherit version;
          src = ./.;
          vendorHash = null;
          subPackages = ".";
          CGO_ENABLED = 0;
          doCheck = false;
          nativeBuildInputs = with pkgs; [ makeWrapper ];
          allowGoReference = true;
          postInstall = ''
            wrapProgram $out/bin/${pname} --prefix PATH : ${
              pkgs.lib.makeBinPath (with pkgs; [ go ])
            }
          '';
          meta = with nixpkgs.lib; {
            description = "Pushup web framework for Go";
            homepage = "https://pushup.adhoc.dev/";
            license = licenses.mit;
          };
        };
      });

      devShells = forEachSystem ({ pkgs, ... }: {
        default =
          pkgs.mkShell { buildInputs = with pkgs; [ go_1_20 gotools gopls ]; };
      });
    };
}
