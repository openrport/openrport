package usercli

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

var ErrPasswordsDoNotMatch = errors.New("passwords do not match")

func promptForPassword(pr PasswordReader) (string, error) {
	fmt.Print("Enter password: ")
	password, err := pr.ReadPassword()
	if err != nil {
		return "", err
	}

	fmt.Print("\nConfirm password: ")
	confirm, err := pr.ReadPassword()
	if err != nil {
		return "", err
	}
	fmt.Print("\n")

	if password != confirm {
		return "", ErrPasswordsDoNotMatch
	}

	return password, nil
}

type PasswordReader interface {
	ReadPassword() (string, error)
}

func NewPasswordReader() PasswordReader {
	return &passwordReaderImpl{}
}

type passwordReaderImpl struct{}

func (passwordReaderImpl) ReadPassword() (string, error) {
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	return string(password), nil
}
