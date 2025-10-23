package basic

// BasicAuthConfig configurações para Basic Authentication
type BasicAuthConfig struct {
	Realm string // Realm para o WWW-Authenticate header
}

// DefaultBasicAuthConfig retorna configuração padrão
func DefaultBasicAuthConfig() *BasicAuthConfig {
	return &BasicAuthConfig{
		Realm: "Restricted",
	}
}
