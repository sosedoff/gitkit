package gitkit

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type Credential struct {
	Username string
	Password string
}

func parseAuth(data string) (Credential, error) {
	cred := Credential{}

	if !strings.HasPrefix(data, "Basic ") {
		return cred, fmt.Errorf("not a basic authentication")
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.Replace(data, "Basic ", "", 1))
	if err != nil {
		return cred, err
	}

	chunks := strings.Split(string(decoded), ":")
	cred.Username = chunks[0]
	cred.Password = chunks[1]

	return cred, nil
}
