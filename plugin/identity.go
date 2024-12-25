package plugin

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	"filippo.io/age"
	"filippo.io/age/agessh"
	page "filippo.io/age/plugin"
	"golang.org/x/crypto/ssh"
)

type Identity struct {
	Version    uint8
	PubKey     ssh.PublicKey
	privateKey []byte
}

func (i *Identity) Serialize() []any {
	return []interface{}{
		&i.Version,
	}
}

func (i *Identity) Recipient() *Recipient {
	return NewRecipient(i.PubKey)
}

func (i *Identity) Unwrap(stanzas []*age.Stanza) (fileKey []byte, err error) {
	ageIdentity, err := agessh.ParseIdentity(i.privateKey)
	if err != nil {
		return nil, err
	}
	switch i := ageIdentity.(type) {
	case *agessh.RSAIdentity:
		return i.Unwrap(stanzas)
	case *agessh.Ed25519Identity:
		return i.Unwrap(stanzas)
	default:
		return nil, fmt.Errorf("unsupported key type: %T", i)
	}
}

func NewIdentity(privateKey []byte) (*Identity, error) {
	// use agessh to check the identity can be parsed
	_, err := agessh.ParseIdentity(privateKey)
	if err != nil {
		return nil, err
	}

	// TODO: use agesshIdentity.SshKey instead when it hopefully gets exposed
	sshKey, err := ssh.ParseRawPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.NewSignerFromKey(sshKey)
	if err != nil {
		return nil, err
	}

	identity := &Identity{
		Version:    1,
		PubKey:     signer.PublicKey(),
		privateKey: privateKey,
	}

	return identity, nil
}

func DecodeIdentity(s string) (*Identity, error) {
	var key Identity
	name, b, err := page.ParseIdentity(s)
	if err != nil {
		return nil, err
	}
	if name != PluginName {
		return nil, fmt.Errorf("invalid hrp")
	}
	r := bytes.NewBuffer(b)
	for _, f := range key.Serialize() {
		if err := binary.Read(r, binary.BigEndian, f); err != nil {
			return nil, err
		}
	}

	publicKey, err := ssh.ParsePublicKey(r.Bytes())
	if err != nil {
		return nil, err
	}

	key.PubKey = publicKey

	privateKey, err := ReadKeyFromPubKeyOp(publicKey)
	if err != nil {
		return nil, err
	}

	key.privateKey = privateKey

	return &key, nil
}

func ParseIdentity(f io.Reader) (*Identity, error) {
	// Same parser as age
	const privateKeySizeLimit = 1 << 24 // 16 MiB
	scanner := bufio.NewScanner(io.LimitReader(f, privateKeySizeLimit))
	var n int
	for scanner.Scan() {
		n++
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		identity, err := DecodeIdentity(line)
		if err != nil {
			return nil, fmt.Errorf("error at line %d: %v", n, err)
		}
		return identity, nil
	}
	return nil, fmt.Errorf("no identities found")
}

func EncodeIdentity(i *Identity) string {
	var b bytes.Buffer
	for _, v := range i.Serialize() {
		binary.Write(&b, binary.BigEndian, v)
	}

	binary.Write(&b, binary.BigEndian, i.PubKey.Marshal())

	return page.EncodeIdentity(PluginName, b.Bytes())
}

var (
	marshalTemplate = `
# Created: %s
`
)

func WriteMarshalHeader(w io.Writer) {
	s := fmt.Sprintf(marshalTemplate, time.Now())
	s = strings.TrimSpace(s)
	fmt.Fprintf(w, "%s\n", s)
}

func (i *Identity) Marshal(w io.Writer) error {
	WriteMarshalHeader(w)
	fmt.Fprintf(w, "# Recipient: %s\n", i.Recipient())
	fmt.Fprintf(w, "\n%s\n", EncodeIdentity(i))
	return nil
}
