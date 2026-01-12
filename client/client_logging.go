package client

import "github.com/meshcloud/terraform-provider-meshstack/client/internal"

// Logger exposes logging for client operations within this package (including internal).
type Logger = internal.Logger

// SetLogger allows setting the client logger. By default, no logging happens.
func SetLogger(logger Logger) {
	internal.Log = logger
}
