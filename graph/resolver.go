package graph

import (
	"github.com/cccac/OpenWarn-Proxy/proxy"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{
	Proxy proxy.Proxy
}
