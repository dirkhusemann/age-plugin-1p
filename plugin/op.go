package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

func ReadKeyFromPathOp(path string) (key []byte, err error) {
	// Log.Printf("reading path from 1Password: %s", path)
	cmd := exec.Command("op", "read", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("could not read key from 1Password at: %v: %v", path, err)
	}
	return output, nil
}

func ListSSHFingerprintsOp() (output []byte, err error) {
	cmd := exec.Command("op", "item", "list", "--categories", "SSH Key", "--format=json")
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("could not get list of SSH keys from 1Password: %v", err)
	}
	return
}

func UnmarshalItemList(output []byte) (items []map[string]interface{}, err error) {
	err = json.Unmarshal(output, &items)
	if err != nil {
		return nil, fmt.Errorf("could not decode list of SSH keys from 1Password: %v", err)
	}
	return
}

func ReadKeyFromPubKeyOp(pubKey ssh.PublicKey) (privateKey []byte, err error) {
	fingerprint := ssh.FingerprintSHA256(pubKey)
	Log.Printf("fingerprint=%s", fingerprint)

	output, err := ListSSHFingerprintsOp()
	if err != nil {
		return nil, err
	}

	items, err := UnmarshalItemList(output)
	if err != nil {
		return nil, err
	}

	var privateKeyPath string

	for _, item := range items {
		additional_information := item["additional_information"].(string)

		if additional_information == fingerprint {
			vault := item["vault"].(map[string]interface{})
			privateKeyPath = fmt.Sprintf("op://%s/%s/private key", vault["id"], item["id"])
			break
		}
	}

	if privateKeyPath == "" {
		return nil, fmt.Errorf("private key not found in 1Password for public key: %s", ssh.MarshalAuthorizedKey(pubKey))
	}

	return ReadKeyFromPathOp(privateKeyPath)
}

func ReadAllKeysOp() (privateKeyFromOpRef map[string][]byte, err error) {
	opItemList := exec.Command("op", "item", "list", "--categories", "SSH Key", "--format=json")
	opItemGet := exec.Command("op", "item", "get", "-", "--fields", "private_key", "--format=json")

	outPipe, err := opItemList.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys from 1Password: %v", err)
	}
	defer outPipe.Close()

	opItemList.Start()
	opItemGet.Stdin = outPipe
	output, err := opItemGet.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH keys from 1Password: %v", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	privateKeyFromOpRef = make(map[string][]byte)

	for {
		var item map[string]interface{}
		err := decoder.Decode(&item)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to decode SSH keys from 1Password: %v", err)
		}

		value := []byte(item["value"].(string))
		reference := item["reference"].(string)
		privateKeyFromOpRef[reference] = value
	}
	return
}
