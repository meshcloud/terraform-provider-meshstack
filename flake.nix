{
  description = "meshStack Terraform Provider";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    nixpkgs-opentofu.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };


  outputs = { self, nixpkgs, nixpkgs-opentofu }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-darwin" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
        opentofu = nixpkgs-opentofu.legacyPackages.${system}.opentofu;
      });
    in
    {
      devShells = forEachSupportedSystem ({ pkgs, opentofu }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            # go 1.27 (pinned, in lock-step with go.mod and with meshstack-cli)
            go_1_27

            # No golangci-lint here: it is a tool directive in go.mod, built by `task lint`.

            opentofu

            # https://taskfile.dev
            go-task

            goreleaser
            zizmor
          ];

          shellHook = ''
            # Explicitly set GOROOT to Nix-installed Go
            export GOROOT="${pkgs.go_1_27}/share/go"
            
            # With auto, a newer go line in go.mod would download and run another Go than this one.
            export GOTOOLCHAIN=local

            # Taskfile.yml sets the same two for its test tasks; these cover a bare `go test`.
            export TF_ACC_TERRAFORM_PATH="${opentofu}/bin/tofu"
            export TF_ACC_PROVIDER_HOST=registry.opentofu.org
          '';
        };
      });
    };
}
