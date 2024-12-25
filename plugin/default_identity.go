package plugin

import (
	"filippo.io/age"
	page "filippo.io/age/plugin"
)

type DefaultIdentity struct {
	identities []Identity
}

func NewDefaultIdentity() (*DefaultIdentity, error) {
	d := new(DefaultIdentity)
	identities, err := GetAllIdentities()
	if err != nil {
		return nil, err
	}
	d.identities = identities
	return d, nil
}

func (d *DefaultIdentity) Unwrap(stanzas []*age.Stanza) (fileKey []byte, err error) {
	for _, identity := range d.identities {
		fileKey, err := identity.Unwrap(stanzas)
		if err == age.ErrIncorrectIdentity {
			continue
		} else if err == nil {
			return fileKey, nil
		}

		return nil, err
	}
	return nil, age.ErrIncorrectIdentity
}

func EncodeDefaultIdentity() string {
	return page.EncodeIdentity(PluginName, nil)
}
