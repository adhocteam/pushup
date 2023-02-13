{
    description = "Pushup is a new web framework for Go";

    inputs.nixpkgs.url = "nixpkgs/nixos-unstable";
    inputs.flake-utils.url = "github:numtide/flake-utils";

    outputs = { self, nixpkgs, flake-utils }:
        flake-utils.lib.eachSystem [ "x86_64-linux" "aarch64-darwin" "aarch64-linux" ]
            (system:
                let pkgs = nixpkgs.legacyPackages.${system}; in
                {
                    devShell = import ./shell.nix { inherit pkgs; };
                }
            );
}
