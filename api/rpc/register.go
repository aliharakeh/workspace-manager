package rpc

// Registerer collects the methods a server serves. The daemon passes one to
// the handler registry.
type Registerer interface {
	Handle(method string, fn Handler)
}
