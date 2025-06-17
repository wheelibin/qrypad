package password

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const keyringService = "qrypad"

var ErrPasswordNotSaved = errors.New("password not saved")

func GetPassword(dbAlias string) (string, error) {
	password, err := keyring.Get(keyringService, dbAlias)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrPasswordNotSaved
		}
	}
	return password, nil
}

func SetPassword(dbAlias string, pass string) error {
	err := keyring.Set(keyringService, dbAlias, pass)
	if err != nil {
		return err
	}
	return nil
}
