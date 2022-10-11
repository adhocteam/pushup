{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
    nativeBuildInputs = with pkgs; [
        go_1_18
        gopls
        gotools
        hugo
    ];
}
