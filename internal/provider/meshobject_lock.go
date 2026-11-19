package provider

import "github.com/meshcloud/terraform-provider-meshstack/internal/util/xsync"

var meshObjectLocks xsync.KeyedMutex

func lockWorkspace(name string) (unlock func()) {
	return meshObjectLocks.Lock("meshWorkspace/" + name)
}

func lockProject(workspace, name string) (unlock func()) {
	return meshObjectLocks.Lock("meshProject/" + workspace + "." + name)
}
