package provider

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type encodeFileFunction struct{}

func NewEncodeFileFunction() function.Function {
	return encodeFileFunction{}
}

func (e encodeFileFunction) Metadata(ctx context.Context, request function.MetadataRequest, response *function.MetadataResponse) {
	response.Name = "encode_file"
}

func (e encodeFileFunction) Definition(ctx context.Context, request function.DefinitionRequest, response *function.DefinitionResponse) {
	response.Definition = function.Definition{
		Parameters: []function.Parameter{function.StringParameter{
			Name:                "content",
			MarkdownDescription: "The file content to encode, e.g. loaded via built-in `file()` function.",
		}},
		Return:  function.StringReturn{},
		Summary: "Encode file content as a base64 data blob",
		MarkdownDescription: "Encodes already-loaded file content as a base64 data blob. " +
			"Due to Terraform's limitation of representing all values as strings, this function only works with text content. " +
			"MIME type detection may be added in a future version. " +
			"This can be used e.g. for Building Block Definition inputs with type " + client.MeshBuildingBlockIOTypeFile.Markdown() +
			", where the MIME type is ignored by the runner implementation. " +
			"Use `provider::meshstack::load_file` instead to read and encode a file from disk in a single step.",
	}
}

func (e encodeFileFunction) Run(ctx context.Context, request function.RunRequest, response *function.RunResponse) {
	var content string
	if err := request.Arguments.GetArgument(ctx, 0, &content); err != nil {
		response.Error = err
		return
	}
	encoded := encodeContentAsDataBlob([]byte(content))
	if err := response.Result.Set(ctx, encoded); err != nil {
		response.Error = err
		return
	}
}

func encodeContentAsDataBlob(content []byte) string {
	return fmt.Sprintf("data:application/octet-stream;base64,%s", base64.StdEncoding.EncodeToString(content))
}
