package logging

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type TerraformClientLogger struct {
	MessagePrefix string
}

var _ client.Logger = TerraformClientLogger{}

func (l TerraformClientLogger) Info(ctx context.Context, msg string, args ...any) {
	tflog.Info(ctx, l.MessagePrefix+msg, convertArgsForLogging(args))
}

func (l TerraformClientLogger) Debug(ctx context.Context, msg string, args ...any) {
	tflog.Debug(ctx, l.MessagePrefix+msg, convertArgsForLogging(args))
}

func convertArgsForLogging(args []any) (result map[string]any) {
	if len(args)%2 == 1 {
		// should actually not happen (args should always be multiple of 2)
		args = append(args, "<missing value>")
	}
	mapOfArgs := make(map[string][]any)
	for i := 0; i < len(args)/2; i++ {
		var key string
		var ok bool
		if key, ok = args[2*i].(string); !ok {
			if keyStringer, ok := args[2*i].(fmt.Stringer); !ok {
				// should actually not happen, log result output anyway
				key = fmt.Sprintf("'%v'(%T) <non-string key at i=%d>", args[2*i], args[2*i], 2*i)
			} else {
				key = keyStringer.String()
			}
		}
		mapOfArgs[key] = append(mapOfArgs[key], args[2*i+1])
	}
	result = make(map[string]any)
	for k, vs := range mapOfArgs {
		if len(vs) == 1 {
			result[k] = vs[0]
		} else {
			for i := range vs {
				result[fmt.Sprintf("%s <duplicate=%d>", k, i)] = vs[i]
			}
		}
	}
	return
}
