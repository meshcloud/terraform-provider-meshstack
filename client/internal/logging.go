package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
)

var Log Logger = noopLogger{}

// Logger only supports Debug and Info log levels.
type Logger interface {
	Info(ctx context.Context, msg string, args ...any)
	Debug(ctx context.Context, msg string, args ...any)
}

type noopLogger struct{}

func (n noopLogger) Info(context.Context, string, ...any) {
	// do nothing
}

func (n noopLogger) Debug(context.Context, string, ...any) {
	// do nothing
}

type loggedHeaders http.Header

var _ fmt.Stringer = loggedHeaders(nil)

func (l loggedHeaders) String() string {
	var lines []string
	for _, k := range slices.Sorted(maps.Keys(l)) {
		for _, v := range l[k] {
			// Avoid printing that longish JWT Bearer token (which is also a secret)
			if k == "Authorization" {
				v = "[REDACTED]"
			}
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return strings.Join(lines, "\n")
}

type loggedBody struct {
	io.Reader
}

var _ fmt.Stringer = loggedBody{}

func (l loggedBody) String() string {
	if buffer, ok := l.Reader.(*bytes.Buffer); ok {
		return bytesToPrettyJson(buffer.Bytes())
	} else if buffer == nil {
		return "<empty>"
	}
	return fmt.Sprintf("<unknown> %v", l.Reader)
}

func bytesToPrettyJson(data []byte) string {
	if len(data) == 0 {
		return "<empty>"
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err == nil {
		if indented, err := json.MarshalIndent(decoded, "", "  "); err == nil {
			return string(indented)
		}
	}
	// should never happen as we should only transfer JSON in request/responses
	return fmt.Sprintf("<string,len=%d> %s", len(data), string(data))
}
