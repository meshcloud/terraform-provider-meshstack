package provider

import (
	"embed"
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/images
var images embed.FS

func Test_encodeImageAsDataBlob(t *testing.T) {
	for _, ext := range []string{"gif", "jpg", "png", "svg", "webp"} {
		t.Run(ext, func(t *testing.T) {
			filename := fmt.Sprintf("image.%s", ext)
			fileContent, err := images.ReadFile(path.Join("testdata/images", filename))
			require.NoError(t, err)
			encoded, err := encodeImageAsDataBlob(fileContent)
			require.NoError(t, err)
			expected, err := images.ReadFile(path.Join("testdata/images", filename+".golden.txt"))
			require.NoError(t, err)
			assert.Equal(t, string(expected), encoded)
		})
	}
}
