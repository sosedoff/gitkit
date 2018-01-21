package gitkit

import (
	"fmt"
	"net/http"
)

type Credential struct {
	Username string
	Password string
}

func getAuth(req *http.Request) (Credential, error) {
	cred := Credential{}

	user, pass, ok := req.BasicAuth()
	if !ok {
		return cred, fmt.Errorf("authentication failed")
	}

	cred.Username = user
	cred.Password = pass

	return cred, nil
}
