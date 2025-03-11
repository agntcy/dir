package types

// API defines a unified interface to Dir services.
type API interface {
	// Store returns an implementation of the StoreAPI
	Store() StoreAPI

	// Routing returns an implementation of the RoutingAPI
	Routing() RoutingAPI
}
