package provider

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/files
var files embed.FS

func Test_loadFileFunction_encodeContentAsDataBlob(t *testing.T) {
	tested := false
	require.NoError(t, fs.WalkDir(files, "testdata/files", func(p string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() || strings.HasSuffix(p, ".golden.txt") {
			return nil
		}
		t.Run(filepath.Base(p), func(t *testing.T) {
			fileContent, err := files.ReadFile(p)
			require.NoError(t, err)
			encoded := encodeContentAsDataBlob(fileContent)
			expected, err := files.ReadFile(p + ".golden.txt")
			require.NoError(t, err)
			assert.Equal(t, string(expected), encoded)
		})
		tested = true
		return nil
	}))
	require.True(t, tested, "expected at least one file to be tested")
}
