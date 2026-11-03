package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

func hclStep(t *testing.T, name string, step int, vars map[string]string) string {
	t.Helper()

	fileName := fmt.Sprintf("%s_step%d.tf", name, step)
	src, err := os.ReadFile(filepath.Join("testdata", "hcl", fileName))
	require.NoError(t, err)

	stepTemplate, err := template.New(fileName).
		Option("missingkey=error").
		Funcs(template.FuncMap{"example": includeExample}).
		Parse(string(src))
	require.NoError(t, err)

	var config strings.Builder
	require.NoError(t, stepTemplate.Execute(&config, vars))
	return config.String()
}

func includeExample(pathParts ...string) (string, error) {
	content, err := examples.ReadFile(strings.Join(pathParts, ""))
	return string(content), err
}
