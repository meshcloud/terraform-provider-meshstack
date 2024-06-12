{
  description = "meshStack Terraform Provider";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
  };


  outputs = { self, nixpkgs }:
    let
      goVersion = 22; # Change this to update the whole stack
      
      # use an overlay to pin default go version
      overlays = [ (final: prev: { go = prev."go_1_${toString goVersion}"; }) ]; 

      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit overlays system; };
      });
    in
    {
      devShells = forEachSupportedSystem ({ pkgs }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            # go 1.22 (specified by overlay)
            go_1_22
            
            # goimports, godoc, etc.
            gotools

            # https://github.com/golangci/golangci-lint
            golangci-lint
          ];
        };
      });
    };
}
