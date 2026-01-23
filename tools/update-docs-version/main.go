package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

const (
	docsFilePath = "docs/index.md"
	placeholder  = "__MIN_MESHSTACK_VERSION__"
)

func main() {
	// Read version directly from the client package
	version := client.MinMeshStackVersion

	fmt.Printf("Using MinMeshStackVersion: %s\n", version)

	// Update the generated docs file
	if err := updateDocs(version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to update docs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated %s\n", docsFilePath)
}

// updateDocs reads the generated docs file, replaces the placeholder with the version, and writes it back.
func updateDocs(version string) error {
	content, err := os.ReadFile(docsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", docsFilePath, err)
	}

	updated := strings.ReplaceAll(string(content), placeholder, version)

	if err := os.WriteFile(docsFilePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", docsFilePath, err)
	}

	return nil
}
