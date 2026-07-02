package auth

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 128
	bcryptCost     = 12
)

var ErrInvalidPassword = errors.New("invalid password")
var ErrNoPassword = errors.New("no password set")
var ErrWrongPassword = errors.New("wrong password")

func hashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	sum, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(sum), nil
}

func verifyPassword(hash, password string) error {
	if password == "" {
		return ErrWrongPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrWrongPassword
	}
	return nil
}

func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordLen || n > maxPasswordLen {
		return ErrInvalidPassword
	}
	return nil
}