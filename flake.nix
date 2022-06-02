{
    description = "Experiments and research for a web forms language";

    inputs.nixpkgs.url = "nixpkgs/nixos-unstable";
    inputs.flake-utils.url = "github:numtide/flake-utils";

    outputs = { self, nixpkgs, flake-utils }:
        flake-utils.lib.eachDefaultSystem
            (system:
                let pkgs = nixpkgs.legacyPackages.${system}; in
                {
                    devShell = import ./shell.nix { inherit pkgs; };
                }
            );
}
