package logging

import (
	"context"
	"log/slog"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSlogHandlerFlattensGroupsAndAttrs pins the shape tflog needs: one flat map, with a group
// path folded into a dotted key. Nothing else in this bridge is worth a test — tflog writes to a
// context-bound sink that a unit test cannot read back.
func TestSlogHandlerFlattensGroupsAndAttrs(t *testing.T) {
	handler := SlogHandler{MessagePrefix: "meshstack: "}
	withAttrs := handler.WithAttrs([]slog.Attr{slog.String("profile", "dev")})
	nested, ok := withAttrs.WithGroup("token").(SlogHandler)
	require.True(t, ok)

	fields := map[string]any{}
	maps.Copy(fields, nested.fields)
	nested.put(fields, slog.Int("expiresIn", 300))

	// "profile" stays unqualified because it was added before the group opened, which is what
	// log/slog specifies; "expiresIn" arrives inside the group and is qualified.
	assert.Equal(t, map[string]any{"profile": "dev", "token.expiresIn": int64(300)}, fields)
}

// TestSlogHandlerLeavesRenderingToTheSink pins the other half of passing every level through: a
// record TF_LOG drops still reaches this handler, so an attribute must arrive at tflog unrendered
// and be formatted by the sink, or every terraform run pays for debug output nobody reads. The
// meshStack CLI logs its request and response bodies this way.
func TestSlogHandlerLeavesRenderingToTheSink(t *testing.T) {
	rendered := 0
	handler := SlogHandler{}

	fields := map[string]any{}
	handler.put(fields, slog.Any("body", countingBody{&rendered}))

	assert.Zero(t, rendered)
	assert.IsType(t, countingBody{}, fields["body"])
}

type countingBody struct{ rendered *int }

func (c countingBody) String() string {
	*c.rendered++
	return "rendered"
}

// TestSlogHandlerPassesEveryLevel pins that the level filter is tflog's, not this handler's:
// TF_LOG decides what terraform keeps, and filtering here would hide records a practitioner asked
// for.
func TestSlogHandlerPassesEveryLevel(t *testing.T) {
	handler := SlogHandler{}
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		assert.True(t, handler.Enabled(context.Background(), level))
	}
}
