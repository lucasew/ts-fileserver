{
  description = "A simple fileserver that can be exposed to the Internet using Tailscale Funnel";

  inputs = {
    nixpkgs.url = "nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }@self: 
  flake-utils.lib.eachDefaultSystem (system: let
    pkgs = import nixpkgs { inherit system; };
  in {
    packages = {
      default = pkgs.python3Packages.callPackage ./package.nix { inherit self; };
    };
    devShells.default = pkgs.mkShell {
      buildInputs = with pkgs; [
        gopls
        go
      ];
    };
  });
}
