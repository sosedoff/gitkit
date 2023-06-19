package gitkit_test

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

func TestKeyLookupFunctionIsNeeded(t *testing.T) {
	s := newSSH(t, t.TempDir())

	err := s.Listen(":0") // random port
	assert.Equal(t, errors.New("public key lookup func is not provided"), err)
}

func TestListener(t *testing.T) {
	testDir := t.TempDir()

	s := newSSH(t, testDir)
	s.PublicKeyLookupFunc = func(content string) (*gitkit.PublicKey, error) {
		return &gitkit.PublicKey{Id: "1234"}, nil
	}

	sshdConfig, err := setup(t, testDir)
	assert.NoError(t, err)

	s.SetSSHConfig(sshdConfig)

	listener, err := net.Listen("tcp4", ":0")
	assert.NoError(t, err)

	s.SetListener(listener)
	t.Logf("address: %s", listener.Addr())

	go func() {
		defer s.Stop()
		err := s.Serve()
		assert.NoError(t, err)
	}()

	// assert the keys are created
	assert.FileExists(t, filepath.Join(testDir, "keys/gitkit.rsa"))
	assert.FileExists(t, filepath.Join(testDir, "keys/gitkit.rsa.pub"))
}

func newSSH(t *testing.T, baseDir string) *gitkit.SSH {
	t.Helper()

	return gitkit.NewSSH(gitkit.Config{
		Auth:       true,
		AutoCreate: true,
		KeyDir:     filepath.Join(baseDir, "keys"),
		Dir:        filepath.Join(baseDir, "repos"),
	})
}

// custom setup function to replicate what gitkit does to setup the ssh server,
// but doesn't do when you supply a custom listener
func setup(t *testing.T, dir string) (*ssh.ServerConfig, error) {
	t.Helper()

	config := &ssh.ServerConfig{
		ServerVersion: fmt.Sprintf("SSH-2.0-gitkit %s", "testing"),
	}

	config.NoClientAuth = true

	// create server key
	k := gitkit.NewKey(filepath.Join(dir, "keys"))

	err := k.CreateRSA()
	assert.NoError(t, err)

	private, err := k.GetRSA()
	assert.NoError(t, err)

	config.AddHostKey(private)
	return config, nil
}
