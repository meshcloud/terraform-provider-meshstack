package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/meshcloud/meshstack-cli/pkg/credential"
	"github.com/meshcloud/meshstack-cli/pkg/diags"
	"github.com/meshcloud/meshstack-cli/pkg/workspace"
)

// providerInput implements auth.Input over the provider block. It differs from the meshStack
// CLI's implementation in what it can do rather than in what it knows: it never prompts, so a
// terraform run can never block on a terminal that is not there, and it never opens a browser.
type providerInput struct {
	data MeshStackProviderModel
}

var _ auth.Input = (*providerInput)(nil)

func (i *providerInput) Explicit() auth.Values {
	values := auth.Values{
		Profile:   i.data.Profile.ValueString(),
		Endpoint:  i.data.Endpoint.ValueString(),
		Workspace: workspace.Name(i.data.Workspace.ValueString()),
		ApiKey:    i.data.ApiKey.ValueString(),
	}
	// The block names the method by which attributes it sets, because a secret never travels
	// in Values: pkg/auth learns that one is available from the demanded method and fetches
	// it through the accessors below only when it actually needs to mint.
	switch {
	case i.data.ApiToken.ValueString() != "":
		values.Method = credential.MethodManual
	case values.ApiKey != "" && i.data.ApiSecret.ValueString() != "":
		values.Method = credential.MethodApiKey
	}
	return values
}

// ApiKeySecret and ApiToken return the block attribute, and fall back to the environment.
// The fallback is not a second precedence rule: pkg/auth has already decided which source
// supplies the credential, and when that source was the environment this is where it reads
// the value from. Without it a CI job exporting MESHSTACK_API_SECRET would resolve to the
// API key method and then find no secret to mint with.
func (i *providerInput) ApiKeySecret(context.Context) (string, error) {
	if secret := i.data.ApiSecret.ValueString(); secret != "" {
		return secret, nil
	}
	secret, _ := auth.SecretFromEnv()
	return secret, nil
}

func (i *providerInput) ApiToken(context.Context) (string, error) {
	if token := i.data.ApiToken.ValueString(); token != "" {
		return token, nil
	}
	token, _ := auth.TokenFromEnv()
	return token, nil
}

// Browser is nil here, and pkg/auth then fails a dead login method by naming `meshstack
// login` rather than waiting for a browser nobody will see. The provider imports neither
// pkg/oidc/browser nor anything that does, so there is no browser flow in this binary to
// disable.
func (i *providerInput) Browser() auth.Browser { return nil }

// problemDiagnostic adapts a diags.Problem to a terraform diagnostic.
//
// Problem deliberately does not implement diag.Diagnostic itself: that interface's Severity()
// and Equal() mention terraform-plugin-framework types, so satisfying it would pull the
// framework into the meshStack CLI's dependency set — and from there into the public checksum
// database — against its two-dependency policy. This adapter is the whole cost of keeping it
// out.
func problemDiagnostic(p diags.Problem) diag.Diagnostic {
	return diag.NewErrorDiagnostic(p.Summary(), p.Detail())
}

// problemDiagnostics turns any error into diagnostics, using the Problem's structure when
// there is one so that the summary and the actionable paragraph land in the right places.
func problemDiagnostics(summary string, err error) diag.Diagnostics {
	var collected diag.Diagnostics
	if p, ok := errors.AsType[diags.Problem](err); ok {
		collected.Append(problemDiagnostic(p))
		return collected
	}
	collected.AddError(summary, err.Error())
	return collected
}
