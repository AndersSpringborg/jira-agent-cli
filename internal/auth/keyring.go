package auth

import (
	"github.com/zalando/go-keyring"
)

const serviceName = "jira-cli"

func SetToken(profile string, token string) error {
	return keyring.Set(serviceName, profile, token)
}

func GetToken(profile string) (string, error) {
	secret, err := keyring.Get(serviceName, profile)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", err
	}
	return secret, nil
}

func DeleteToken(profile string) (bool, error) {
	err := keyring.Delete(serviceName, profile)
	if err != nil {
		if err == keyring.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
