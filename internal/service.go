package internal

import "go.uber.org/fx"

// ServiceInfo contains information about the service.
type ServiceInfo struct {
	fx.Out

	Name    string `name:"service:name"`
	Version string `name:"service:version"`

	Development bool `name:"env:development"`
	Testing     bool `name:"env:testing"`
}
