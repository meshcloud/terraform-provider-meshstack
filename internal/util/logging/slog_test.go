package logging

import (
	"log/slog"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tflog writes to a context-bound sink a unit test cannot read back, so these tests cover what the
// handler decides before it hands a record over, and nothing beyond that.
func TestSlogHandlerQualifiesOnlyTheAttrsAddedAfterWithGroup(t *testing.T) {
	handler := SlogHandler{MessagePrefix: "meshstack: "}
	withAttrs := handler.WithAttrs([]slog.Attr{slog.String("profile", "dev")})
	nested, ok := withAttrs.WithGroup("token").(SlogHandler)
	require.True(t, ok)

	fields := map[string]any{}
	maps.Copy(fields, nested.fields)
	nested.put(fields, slog.Int("expiresIn", 300))

	assert.Equal(t, map[string]any{"profile": "dev", "token.expiresIn": int64(300)}, fields)
}

func TestSlogHandlerFlattensAGroupValuedAttribute(t *testing.T) {
	handler, ok := SlogHandler{}.WithGroup("token").(SlogHandler)
	require.True(t, ok)

	fields := map[string]any{}
	handler.put(fields, slog.Group("request",
		slog.String("method", "GET"),
		slog.Group("header", slog.String("accept", "application/json")),
		slog.Group("", slog.Int("attempt", 2)),
	))

	assert.Equal(t, map[string]any{
		"token.request.method":        "GET",
		"token.request.header.accept": "application/json",
		"token.request.attempt":       int64(2),
	}, fields)
}

func TestSlogHandlerOpensNoGroupForAnEmptyName(t *testing.T) {
	handler, ok := SlogHandler{}.WithGroup("").(SlogHandler)
	require.True(t, ok)

	fields := map[string]any{}
	handler.put(fields, slog.String("profile", "dev"))

	assert.Equal(t, map[string]any{"profile": "dev"}, fields)
}

// A record TF_LOG drops still reaches this handler, so an attribute has to arrive at tflog
// unrendered — the meshStack CLI logs its request and response bodies as attributes.
func TestSlogHandlerDoesNotRenderAnAttribute(t *testing.T) {
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
