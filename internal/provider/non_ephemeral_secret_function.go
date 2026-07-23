package provider

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nonEphemeralSecretFunction struct{}

func NewNonEphemeralSecretFunction() function.Function {
	return nonEphemeralSecretFunction{}
}

// The backend computes secret_hash, so it is not part of the return.
var nonEphemeralSecretReturnTypes = map[string]attr.Type{
	"secret_value":   types.StringType,
	"secret_version": types.StringType,
}

func (f nonEphemeralSecretFunction) Metadata(ctx context.Context, request function.MetadataRequest, response *function.MetadataResponse) {
	response.Name = "non_ephemeral_secret"
}

func (f nonEphemeralSecretFunction) Definition(ctx context.Context, request function.DefinitionRequest, response *function.DefinitionResponse) {
	response.Definition = function.Definition{
		Parameters: []function.Parameter{function.StringParameter{
			Name:                "secret_value",
			MarkdownDescription: "The secret value, stored in Terraform config or state rather than supplied via an `ephemeral` resource.",
		}},
		Return:  function.ObjectReturn{AttributeTypes: nonEphemeralSecretReturnTypes},
		Summary: "Build a secret block from a value stored in config or state, keyed for rotation by its hash",
		MarkdownDescription: "Builds the `sensitive` secret block used across the provider (for example " +
			"`meshstack_platform`, `meshstack_integration`, `meshstack_building_block`) from a secret value that " +
			"lives in Terraform config or state. It sets `secret_value` to the value and `secret_version` to " +
			"`sha256(secret_value)`. Change the value and the version changes with it, which sends the write only " +
			"`secret_value` to the backend again. Leave the value alone and there is no diff. This is the " +
			"documented `secret_version = nonsensitive(sha256(<secret_value>))` workaround in one call.\n\n" +
			"When the value is sensitive, for example a `sensitive` variable, wrap the argument in " +
			"`nonsensitive(...)`: `access_token = provider::meshstack::non_ephemeral_secret(nonsensitive(var.access_token))`. " +
			"Terraform and OpenTofu mark the whole result of a function call sensitive when any argument is, which " +
			"would hide `secret_version` as `(sensitive value)` and remove the visible rotation trigger this " +
			"function is for. Stripping the mark is safe here. `secret_value` is a write only attribute, so it " +
			"never reaches state and shows only as a placeholder in plans. `secret_version` is a `sha256` of the " +
			"value, so publish it only when revealing that hash is acceptable for your case, because a low entropy " +
			"value can be guessed from its hash.\n\n" +
			"Storing no secret at all is better. Where meshStack allows it, use keyless authentication such as " +
			"workload identity federation, or feed `secret_value` from an `ephemeral` resource. Reach for this " +
			"function only when the value already lives in config or state.",
	}
}

func (f nonEphemeralSecretFunction) Run(ctx context.Context, request function.RunRequest, response *function.RunResponse) {
	var value string
	if err := request.Arguments.GetArgument(ctx, 0, &value); err != nil {
		response.Error = err
		return
	}

	secret, diags := types.ObjectValue(nonEphemeralSecretReturnTypes, map[string]attr.Value{
		"secret_value":   types.StringValue(value),
		"secret_version": types.StringValue(fmt.Sprintf("%x", sha256.Sum256([]byte(value)))),
	})
	if response.Error = function.FuncErrorFromDiags(ctx, diags); response.Error != nil {
		return
	}

	response.Error = response.Result.Set(ctx, secret)
}
