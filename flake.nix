{
  description = "Pushup is a web framework for Go";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

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
          version = "0.3.0"; # TODO(paulsmith): source from version.go
          sourceFiles = fs.difference
            # include anything that matches these files inc. recursive directories
            (fs.unions [
              (fs.fileFilter (file: file.hasExt "go") ./.)
              ./banner.txt
              ./go.mod
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
          # https://github.com/NixOS/nixpkgs/blob/master/pkgs/build-support/go/module.nix
          default = pkgs.buildGoModule.override {
            go = pkgs.go_1_23;
          } rec {
            inherit pname version src vendorHash subPackages CGO_ENABLED meta;
          };
        });

      devShells = forEachSystem ({ pkgs, ... }: {
        default =
          pkgs.mkShell { buildInputs = with pkgs; [ go_1_23 gotools gopls ]; };
      });
    };
}
