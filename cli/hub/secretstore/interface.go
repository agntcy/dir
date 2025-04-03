package secretstore

type SecretStore interface {
	GetHubSecret(string) (*HubSecret, error)
	SaveHubSecret(string, *HubSecret) error
	RemoveHubSecret(string) error
}
