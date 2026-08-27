package logging

import (
	"context"
	"log/slog"
	"maps"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SlogHandler forwards log/slog records to tflog, so that records from the meshStack CLI's
// pkg/ packages land in the terraform log stream.
//
// It is not optional. Everything under pkg/ logs through the slog default logger and takes no
// logger parameter, which is the right shape for a library with one process and one
// destination — but the destination differs per front end. Without this bridge those records
// reach slog's built-in handler and land on stderr in a format terraform does not expect, or
// vanish.
//
// It exists for debugging output. Anything a practitioner must see travels as a diagnostic
// through auth.Input.Warn, not as a log record.
type SlogHandler struct {
	// MessagePrefix says which process the record came from, so that the meshStack CLI's
	// records are distinguishable from the provider's own in one terraform log.
	MessagePrefix string
	// fields holds the attributes WithAttrs collected, already qualified by whatever groups
	// were open when they arrived. Qualifying on the way in is what keeps an attribute added
	// before WithGroup out of that group, which is what log/slog specifies.
	fields map[string]any
	groups []string
}

var _ slog.Handler = SlogHandler{}

// Enabled passes everything through, because tflog owns the level: TF_LOG decides what
// terraform keeps, and a level filter here would hide records the practitioner asked for.
func (h SlogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h SlogHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := make(map[string]any, len(h.fields)+record.NumAttrs())
	maps.Copy(fields, h.fields)
	record.Attrs(func(attr slog.Attr) bool {
		h.put(fields, attr)
		return true
	})

	message := h.MessagePrefix + record.Message
	switch {
	case record.Level >= slog.LevelError:
		tflog.Error(ctx, message, fields)
	case record.Level >= slog.LevelWarn:
		tflog.Warn(ctx, message, fields)
	case record.Level >= slog.LevelInfo:
		tflog.Info(ctx, message, fields)
	default:
		tflog.Debug(ctx, message, fields)
	}
	return nil
}

func (h SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h
	next.fields = make(map[string]any, len(h.fields)+len(attrs))
	maps.Copy(next.fields, h.fields)
	for _, attr := range attrs {
		h.put(next.fields, attr)
	}
	return next
}

func (h SlogHandler) WithGroup(name string) slog.Handler {
	next := h
	next.groups = append(append([]string{}, h.groups...), name)
	return next
}

// put flattens a group path into a dotted key, because tflog's fields are a flat map.
func (h SlogHandler) put(fields map[string]any, attr slog.Attr) {
	key := attr.Key
	for i := len(h.groups) - 1; i >= 0; i-- {
		key = h.groups[i] + "." + key
	}
	fields[key] = attr.Value.Resolve().Any()
}
