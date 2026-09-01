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
            # go 1.27 (pinned, in lock-step with go.mod — and with the meshStack CLI,
            # whose client package this provider consumes; that module needs 1.27)
            go_1_27
            
            # goimports, godoc, etc.
            gotools

            # No golangci-lint here: it is a tool directive in go.mod, so `task lint` builds it
            # with the pinned Go rather than taking whatever nixpkgs built it with.

            # https://www.shellcheck.net — lints .github/scripts (task lint:shell)
            shellcheck

            # https://github.com/hashicorp/terraform-plugin-docs
            terraform-plugin-docs

            # https://github.com/hashicorp/terraform
            terraform

            # https://taskfile.dev
            go-task
          ];

          shellHook = ''
            # Explicitly set GOROOT to Nix-installed Go
            export GOROOT="${pkgs.go_1_27}/share/go"
            
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
