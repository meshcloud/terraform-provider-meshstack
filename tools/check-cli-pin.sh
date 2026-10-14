#!/usr/bin/env bash
# Checks that go.mod pins the meshstack-cli code that a go.work builds this module against, as the
# acceptance runs do, by comparing the sources of the packages this module compiles. Without such a
# go.work, it checks that the pinned commit is on meshstack-cli's main.
set -euo pipefail

cli=github.com/meshcloud/meshstack-cli
cli_url=https://$cli

sources_template=$(
  cat <<'EOF'
{{- if and .Module (eq .Module.Path "github.com/meshcloud/meshstack-cli")}}
  {{- $m := .Module.Dir}}{{$d := .Dir}}
  {{- range .GoFiles}}{{$m}} {{$d}}/{{.}}{{"\n"}}{{end}}
  {{- range .EmbedFiles}}{{$m}} {{$d}}/{{.}}{{"\n"}}{{end}}
{{- end}}
EOF
)

sources() {
  go list -deps -test -f "$sources_template" ./... |
    while read -r module file; do
      printf '%s %s\n' "$(sha256sum <"$file" | cut -c1-64)" "${file#"$module"/}"
    done | sort -u
}

GOWORK=off go mod download "$cli"
version=$(GOWORK=off go list -m -f '{{.Version}}' "$cli")
if [[ $version =~ -([0-9a-f]{12})$ ]]; then
  revision=${BASH_REMATCH[1]}
else
  revision=$version
fi

workspace_dir=$(go list -m -f '{{.Dir}}' "$cli")
if [[ $workspace_dir != "$(GOWORK=off go list -m -f '{{.Dir}}' "$cli")" ]]; then
  differing=$(comm -3 <(GOWORK=off sources) <(sources) | awk '{ print $2 }' | sort -u)
  if [[ -n $differing ]]; then
    head=$(git -C "$workspace_dir" rev-parse HEAD)
    echo "::error::go.mod pins meshstack-cli $version, but the go.work builds $workspace_dir at $head, and these sources differ:"
    echo "$differing"
    echo "Push meshstack-cli, then repin: GOWORK=off go get $cli@$head && GOWORK=off go mod tidy"
    exit 1
  fi
  echo "go.mod pins meshstack-cli $version, which builds the same sources as $workspace_dir."
  exit 0
fi

clone=$(mktemp -d)
trap 'rm -rf "$clone"' EXIT
git clone --quiet --filter=blob:none --no-checkout "$cli_url" "$clone"
if ! git -C "$clone" merge-base --is-ancestor "$revision" origin/main 2>/dev/null; then
  echo "::error::go.mod pins meshstack-cli $version, which is not on its main branch."
  echo "Repin once the meshstack-cli change is on main: GOWORK=off go get $cli@main && GOWORK=off go mod tidy"
  exit 1
fi
echo "go.mod pins meshstack-cli $version, which is on its main branch."
