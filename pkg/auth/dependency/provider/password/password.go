package password

const providerPassword string = "password"

type Provider interface {
	IsLoginIDValid(loginID map[string]string) bool
	CreatePrincipalsByLoginID(authInfoID string, password string, loginID map[string]string) error
	CreatePrincipal(principal Principal) error
	GetPrincipalByLoginIDWithRealm(loginIDKey string, loginID string, realm string, principal *Principal) (err error)
	GetPrincipalsByUserID(userID string) ([]*Principal, error)
	GetPrincipalsByEmail(email string) ([]*Principal, error)
	UpdatePrincipal(principal Principal) error
}
