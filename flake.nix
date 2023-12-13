{
  description = "Pushup is a new web framework for Go";

  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      forEachSystem = fn:
        nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-darwin" "aarch64-linux" ]
        (system: fn nixpkgs.legacyPackages.${system});
    in {
      devShells = forEachSystem (pkgs: {
        default =
          pkgs.mkShell { buildInputs = with pkgs; [ go_1_20 gotools gopls ]; };
      });
    };
}
