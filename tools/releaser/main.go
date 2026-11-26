package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/meshcloud/meshstack-cli/client"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, `Usage: go run ./tools/releaser "$(git describe --tags --abbrev=0 HEAD^)..HEAD"`)
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(revisionRange string) error {
	commits, err := listCommits(revisionRange)
	if err != nil {
		return err
	}
	notes := render(client.MinMeshStackVersion.String(), collectEntries(commits))
	if _, err := os.Stdout.WriteString(notes); err != nil {
		return err
	}
	previousTag, _, _ := strings.Cut(revisionRange, "..")
	if hint := nextVersionHint(previousTag, commits); hint != "" {
		annotate("notice", hint)
	}
	return nil
}

func warn(format string, args ...any) {
	annotate("warning", fmt.Sprintf(format, args...))
}

func annotate(level, message string) {
	switch {
	case os.Getenv("GITHUB_ACTIONS") == "true":
		message = "::" + level + "::" + strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(message)
	case level == "warning":
		message = "Warning: " + message
	}
	fmt.Fprintln(os.Stderr, message)
}
