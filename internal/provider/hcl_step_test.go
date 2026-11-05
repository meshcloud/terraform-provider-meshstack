package provider

import (
	"embed"
	"fmt"
	"path"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

//go:embed testdata/hcl
var hclSteps embed.FS

type hclStepVars map[string]string

func hclStep(t *testing.T, dir string, step int, vars hclStepVars) string {
	t.Helper()

	filePath := path.Join("testdata", "hcl", dir, fmt.Sprintf("step%d.tf", step))
	src, err := hclSteps.ReadFile(filePath)
	require.NoError(t, err)

	stepTemplate, err := template.New(filePath).
		Delims("[[", "]]").
		Option("missingkey=error").
		Funcs(template.FuncMap{"example": includeExample, "replace": replaceLiteral}).
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

func replaceLiteral(literal, replacement, config string) (string, error) {
	if !strings.Contains(config, literal) {
		return "", fmt.Errorf("%q does not occur in the config it should replace", literal)
	}
	return strings.ReplaceAll(config, literal, replacement), nil
}
