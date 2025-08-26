package types

// Authorizer defines the interface for authorization.
// It checks if a user is allowed to perform a specific request.
type Authorizer interface {
	Authorize(userID string, request string) bool
}
