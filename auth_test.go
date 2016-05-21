package gitkit

import (
	"strings"
	"testing"
)

func Test_parseAuth(t *testing.T) {
	_, err := parseAuth("foobar")
	if err == nil {
		t.Error("Expected error, got no error")
	}
	if err.Error() != "not a basic authentication" {
		t.Errorf("Expected error message, got: %s", err.Error())
	}

	_, err = parseAuth("Basic qwe123")
	if err == nil {
		t.Error("Expected error, got no error")
	}
	if !strings.Contains(err.Error(), "illegal base64 data") {
		t.Errorf("Expected base64 decode error, got: %s", err.Error())
	}

	cred, err := parseAuth("Basic QWxhZGRpbjpPcGVuU2VzYW1l")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cred.Username != "Aladdin" {
		t.Fatalf("Got username: %s", cred.Username)
	}
	if cred.Password != "OpenSesame" {
		t.Fatalf("Got username: %s", cred.Password)
	}
}
