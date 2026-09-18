package logging

import (
	"context"
	"log/slog"
	"maps"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SlogHandler forwards log/slog records to tflog, so that records the meshStack CLI's pkg/
// packages write through the slog default logger land in the terraform log stream.
//
// It carries more than debugging output, because pkg/ reports to a front end in exactly two ways —
// an error return, or an slog record. A warning it raises therefore reaches a practitioner as a
// TF_LOG=WARN log line and never as a warning diagnostic in plan output.
type SlogHandler struct {
	MessagePrefix string
	// fields holds what WithAttrs collected, qualified by the groups that were open at the time:
	// log/slog keeps an attribute added before WithGroup out of that group.
	fields map[string]any
	groups []string
}

var _ slog.Handler = SlogHandler{}

// Enabled passes everything through, because tflog owns the level: TF_LOG decides what terraform
// keeps, and a filter here would hide records the practitioner asked for. The cost is that every
// record is handled, including the ones TF_LOG drops, so nothing here may render an attribute.
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
	// log/slog's handler contract makes an empty name a no-op.
	if name == "" {
		return h
	}
	next := h
	next.groups = append(append([]string{}, h.groups...), name)
	return next
}

// put flattens a group path into a dotted key, because tflog's fields are a flat map.
func (h SlogHandler) put(fields map[string]any, attr slog.Attr) {
	prefix := ""
	for _, group := range h.groups {
		prefix += group + "."
	}
	putAttr(fields, prefix, attr)
}

func putAttr(fields map[string]any, prefix string, attr slog.Attr) {
	value := attr.Value.Resolve()
	if value.Kind() != slog.KindGroup {
		fields[prefix+attr.Key] = value.Any()
		return
	}
	// log/slog's handler contract inlines a group whose key is empty.
	if attr.Key != "" {
		prefix += attr.Key + "."
	}
	for _, member := range value.Group() {
		putAttr(fields, prefix, member)
	}
}
