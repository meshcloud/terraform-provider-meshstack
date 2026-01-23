package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/version"
)

const (
	docsFilePath = "docs/index.md"
	placeholder  = "__MIN_MESHSTACK_VERSION__"
)

func main() {
	// Use version directly from the client package
	// Update the generated docs file
	if err := updateDocs(client.MinMeshStackVersion); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: failed to update docs: %v\n", err)
		os.Exit(1)
	}
}

// updateDocs reads the generated docs file, replaces the placeholder with the version, and writes it back.
func updateDocs(version version.Version) error {
	fmt.Printf("Using MinMeshStackVersion: %s\n", version)

	content, err := os.ReadFile(docsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", docsFilePath, err)
	}

	updated := strings.ReplaceAll(string(content), placeholder, version.String())

	if err := os.WriteFile(docsFilePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", docsFilePath, err)
	}
	fmt.Printf("Successfully updated %s\n", docsFilePath)
	return nil
}
