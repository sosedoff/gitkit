package gitkit

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getCredential(t *testing.T) {
	req, _ := http.NewRequest("get", "http://localhost", nil)
	_, err := getCredential(req)
	assert.Error(t, err)
	assert.Equal(t, "authentication failed", err.Error())

	req, _ = http.NewRequest("get", "http://localhost", nil)
	req.SetBasicAuth("Alladin", "OpenSesame")
	cred, err := getCredential(req)

	assert.NoError(t, err)
	assert.Equal(t, "Alladin", cred.Username)
	assert.Equal(t, "OpenSesame", cred.Password)
}
