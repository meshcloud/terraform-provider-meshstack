package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type loadFileFunction struct{}

func NewLoadFileFunction() function.Function {
	return loadFileFunction{}
}

func (l loadFileFunction) Metadata(ctx context.Context, request function.MetadataRequest, response *function.MetadataResponse) {
	response.Name = "load_file"
}

func (l loadFileFunction) Definition(ctx context.Context, request function.DefinitionRequest, response *function.DefinitionResponse) {
	response.Definition = function.Definition{
		Parameters: []function.Parameter{function.StringParameter{
			Name:                "filepath",
			MarkdownDescription: "The path to the file to load.",
		}},
		Return:  function.StringReturn{},
		Summary: "Load a file and encode it as a base64 data blob",
		MarkdownDescription: "Reads a file from disk and encodes it as a base64 data blob. " +
			"MIME type detection may be added in a future version. " +
			"This can be used e.g. for Building Block Definition inputs with type " + client.MeshBuildingBlockIOTypeFile.Markdown() +
			", where the MIME type is ignored by the runner implementation. " +
			"Use `provider::meshstack::encode_file` instead if the file content is already loaded, e.g. via built-in `file()` function.",
	}
}

func (l loadFileFunction) Run(ctx context.Context, request function.RunRequest, response *function.RunResponse) {
	var filename string
	if err := request.Arguments.GetArgument(ctx, 0, &filename); err != nil {
		response.Error = err
		return
	}
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		response.Error = &function.FuncError{Text: fmt.Sprintf("Reading file %s failed: %s", filename, err.Error())}
		return
	}
	encoded := encodeContentAsDataBlob(fileContent)
	if err := response.Result.Set(ctx, encoded); err != nil {
		response.Error = err
		return
	}
}
