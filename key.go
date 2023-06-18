package gitkit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

type Key struct {
	keyDir  string
	keyName string
}

func NewKey(keyDir string) *Key {
	return &Key{
		keyDir:  keyDir,
		keyName: "gitkit.rsa",
	}
}

func (k *Key) CreateRSA() error {
	if err := os.MkdirAll(k.keyDir, os.ModePerm); err != nil {
		return err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	keyPath := filepath.Join(k.keyDir, k.keyName)

	privateKeyFile, err := os.Create(keyPath)
	if err != nil {
		return err
	}

	if err := os.Chmod(keyPath, 0600); err != nil {
		return err
	}
	defer privateKeyFile.Close()
	if err != nil {
		return err
	}
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return err
	}

	pubKeyPath := keyPath + ".pub"
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	return os.WriteFile(pubKeyPath, ssh.MarshalAuthorizedKey(pub), 0644)
}

func (k *Key) GetRSA() (ssh.Signer, error) {
	keyPath := filepath.Join(k.keyDir, k.keyName)
	privateBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(privateBytes)
}
