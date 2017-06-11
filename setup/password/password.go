package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func getPassword() (string, error) {
	if len(os.Args) < 2 {
		return "", errors.New(`use "password <password> to generate a new password"`)
	}

	var password = os.Args[1]

	var hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func main() {
	var hash, err = getPassword()

	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}

	fmt.Println(hash)
}
