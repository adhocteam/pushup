{
  description = "Pushup is a new web framework for Go";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.11";

  outputs = { self, nixpkgs }:
    let
      name = "pushup";
      forEachSystem = fn:
        nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-darwin" "aarch64-linux" ]
        (system:
          fn {
            pkgs = nixpkgs.legacyPackages.${system};
            inherit system;
          });
      fs = nixpkgs.lib.fileset;
    in {
      packages = forEachSystem ({ pkgs, system }:
        let
          pname = name;
          version = "0.0.1"; # TODO(paulsmith): source from version.go
          sourceFiles = fs.difference
            # include anything that matches these files inc. recursive directories
            (fs.unions [
              (fs.fileFilter (file: file.hasExt "go") ./.)
              ./_runtime
              ./banner.txt
              ./go.mod
              ./scaffold
              ./testdata
              ./vendor
            ])
            # exclude these files
            (fs.unions [ ./example ./tools ]);
          src = fs.toSource {
            root = ./.;
            fileset = sourceFiles;
          };
          vendorHash = null;
          subPackages = ".";
          CGO_ENABLED = 0;
          meta = with nixpkgs.lib; {
            description = "Pushup web framework for Go";
            homepage = "https://pushup.adhoc.dev/";
            license = licenses.mit;
          };

        in {
          default = pkgs.buildGoModule rec {
            inherit pname version src vendorHash subPackages CGO_ENABLED meta;
          };

          withGo = pkgs.buildGoModule rec {
            inherit pname version src vendorHash subPackages CGO_ENABLED meta;
            nativeBuildInputs = with pkgs; [ makeWrapper ];
            allowGoReference = true;
            postInstall = ''
              wrapProgram $out/bin/${pname} --prefix PATH : ${
                pkgs.lib.makeBinPath (with pkgs; [ go ])
              }
            '';
          };
        });

      devShells = forEachSystem ({ pkgs, ... }: {
        default =
          pkgs.mkShell { buildInputs = with pkgs; [ go_1_20 gotools gopls ]; };
      });
    };
}
