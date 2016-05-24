package gitkit

import (
	"path/filepath"
)

type Config struct {
	KeyDir     string            // Directory for server ssh keys. Only used in SSH strategy.
	Dir        string            // Directory that contains repositories
	GitPath    string            // Path to git binary
	GitUser    string            // User for ssh connections
	AutoCreate bool              // Automatically create repostories
	Hooks      map[string][]byte // Scripts for hooks/* directory
	Auth       bool              // Require authentication
}

func (c *Config) KeyPath() string {
	return filepath.Join(c.KeyDir, "gitkit.rsa")
}
