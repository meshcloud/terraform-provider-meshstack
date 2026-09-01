package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/meshcloud/meshstack-cli/pkg/credential"
	"github.com/meshcloud/meshstack-cli/pkg/diags"
	"github.com/meshcloud/meshstack-cli/pkg/meshstack"
	"github.com/meshcloud/meshstack-cli/pkg/profile"
	"github.com/meshcloud/meshstack-cli/pkg/setting"
)

// blockSource is the provider block as a settings source: the top of every ranked list, and
// the whole of what this repository contributes to resolving a session. It never prompts and
// never opens a browser, because a terraform run must not block on a terminal that is not
// there.
type blockSource struct {
	data MeshStackProviderModel
}

var _ setting.Source = blockSource{}

func (b blockSource) Lookup(key string) (string, bool) {
	_, value := b.attribute(key)
	return value, value != ""
}

func (b blockSource) Describe(key string) string {
	name, _ := b.attribute(key)
	return "provider block " + name
}

// attribute answers both halves of setting.Source from one switch over the EnvKey, which is a
// setting's identity, so a value and the attribute named as its origin cannot drift apart.
func (b blockSource) attribute(key string) (name, value string) {
	switch key {
	case meshstack.Endpoint.EnvKey:
		return "endpoint", b.data.Endpoint.ValueString()
	case profile.Name.EnvKey:
		return "profile", b.data.Profile.ValueString()
	case meshstack.Workspace.EnvKey:
		return "workspace", b.data.Workspace.ValueString()
	case credential.ApiKeyId.EnvKey:
		return "apikey", b.data.ApiKey.ValueString()
	case credential.ApiSecret.EnvKey:
		return "apisecret", b.data.ApiSecret.ValueString()
	case credential.ApiToken.EnvKey:
		return "apitoken", b.data.ApiToken.ValueString()
	}
	return key, ""
}

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
