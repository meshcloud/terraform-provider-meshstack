package provider

import (
	"embed"
	"fmt"
	"path"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/images
var images embed.FS

func Test_encodeImageAsDataBlob(t *testing.T) {
	g := goldie.New(t, goldie.WithNameSuffix(".golden.txt"), goldie.WithFixtureDir("testdata/images"))
	for _, ext := range []string{"gif", "jpg", "png", "svg", "webp"} {
		t.Run(ext, func(t *testing.T) {
			filename := fmt.Sprintf("image.%s", ext)
			fileContent, err := images.ReadFile(path.Join("testdata/images", filename))
			require.NoError(t, err)
			encoded, err := encodeImageAsDataBlob(fileContent)
			require.NoError(t, err)
			g.Assert(t, filename, []byte(encoded))
		})
	}
}
