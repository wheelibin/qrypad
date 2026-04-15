package password

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keyringService = "qrypad"

var ErrPasswordNotSaved = errors.New("password not saved")

func GetPassword(connectionName string) (string, error) {
	password, err := keyring.Get(keyringService, connectionName)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrPasswordNotSaved
		}
		return "", fmt.Errorf("failed to read password from keyring: %w", err)
	}
	return password, nil
}

func SetPassword(connectionName string, pass string) error {
	err := keyring.Set(keyringService, connectionName, pass)
	if err != nil {
		return err
	}
	return nil
}
