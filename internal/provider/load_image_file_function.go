package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/gabriel-vasile/mimetype"
	"github.com/hashicorp/terraform-plugin-framework/function"
)

type loadImageFileFunction struct{}

func NewLoadImageFileFunction() function.Function {
	return loadImageFileFunction{}
}

func (l loadImageFileFunction) Metadata(ctx context.Context, request function.MetadataRequest, response *function.MetadataResponse) {
	response.Name = "load_image_file"
}

func (l loadImageFileFunction) Definition(ctx context.Context, request function.DefinitionRequest, response *function.DefinitionResponse) {
	response.Definition = function.Definition{
		Parameters: []function.Parameter{function.StringParameter{
			Name:                "filepath",
			MarkdownDescription: "The filename of the image",
		}},
		Return:              function.StringReturn{},
		Summary:             "Load an image file with meshStack API compliant encoding",
		MarkdownDescription: "Loads an image file and encodes it as a base64 string. Also detects the MIME type of the input file.",
	}
}

func (l loadImageFileFunction) Run(ctx context.Context, request function.RunRequest, response *function.RunResponse) {
	var filename string
	if err := request.Arguments.GetArgument(ctx, 0, &filename); err != nil {
		response.Error = err
		return
	}
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		response.Error = &function.FuncError{Text: fmt.Sprintf("Reading image file %s failed: %s", filename, err.Error())}
		return
	}
	if encoded, err := encodeImageAsDataBlob(fileContent); err != nil {
		response.Error = &function.FuncError{Text: fmt.Sprintf("Encoding image file %s (%d bytes) failed: %s", filename, len(fileContent), err.Error())}
		return
	} else if err := response.Result.Set(ctx, encoded); err != nil {
		response.Error = err
		return
	}
}

func encodeImageAsDataBlob(fileContent []byte) (string, error) {
	mimeType := mimetype.Detect(fileContent)
	if mimeType.Is("application/octet-stream") {
		return "", fmt.Errorf("cannot detect image type")
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(fileContent)), nil
}
