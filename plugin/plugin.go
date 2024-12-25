package plugin

import "fmt"

const (
	PluginName = "1p"
)

func CreateIdentityFromPath(privateKeyPath string) (*Identity, error) {
	privateKey, err := ReadKeyFromPathOp(privateKeyPath)
	if err != nil {
		return nil, err
	}

	return NewIdentity(privateKey)
}

func GetAllIdentities() (identities []Identity, err error) {
	privateKeyForOpRef, err := ReadAllKeysOp()
	if err != nil {
		return nil, err
	}

	for _, privateKey := range privateKeyForOpRef {
		i, err := NewIdentity(privateKey)
		if err != nil {
			return nil, err
		}
		identities = append(identities, *i)
	}
	return
}

func MarshalAllRecipients() (out string, err error) {
	privateKeysForOpRef, err := ReadAllKeysOp()
	if err != nil {
		return "", err
	}
	for opRef, privateKey := range privateKeysForOpRef {
		identity, err := NewIdentity(privateKey)
		if err != nil {
			return "", err
		}

		out += fmt.Sprintf("%s: %s\n", opRef, identity.Recipient())
	}
	return
}
