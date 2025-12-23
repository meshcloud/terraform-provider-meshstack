{
  description = "meshStack Terraform Provider";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
  };


  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { 
          inherit system; 
          config.allowUnfreePredicate = pkg: builtins.elem (nixpkgs.lib.getName pkg) [
            "terraform"
          ];
        };
      });
    in
    {
      devShells = forEachSupportedSystem ({ pkgs }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            # go 1.25 (pinned)
            go_1_25
            
            # goimports, godoc, etc.
            gotools

            # https://github.com/golangci/golangci-lint
            golangci-lint

            # https://github.com/hashicorp/terraform-plugin-docs
            terraform-plugin-docs

            # https://github.com/hashicorp/terraform
            terraform
          ];

          shellHook = ''
            # Explicitly set GOROOT to Nix-installed Go
            export GOROOT="${pkgs.go_1_25}/share/go"
            
            # Isolate Go environment from system
            export GOPATH="$PWD/.nix-go"
            export GOCACHE="$PWD/.nix-go/cache"
            export GOMODCACHE="$PWD/.nix-go/mod"
            export GOBIN="$PWD/.nix-go/bin"
            export PATH="$GOBIN:$PATH"
            
            mkdir -p "$GOPATH" "$GOCACHE" "$GOMODCACHE" "$GOBIN"
          '';
        };
      });
    };
}
