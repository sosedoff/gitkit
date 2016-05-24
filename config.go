package gitkit

import (
	"io/ioutil"
	"os"
	"path"
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

func (c *Config) Setup() error {
	if _, err := os.Stat(c.Dir); err != nil {
		if err = os.Mkdir(c.Dir, 0755); err != nil {
			return err
		}
	}

	// Reconfigure all git hooks if they're configured, otherwise leave as-is.
	if len(c.Hooks) > 0 {
		items, err := ioutil.ReadDir(c.Dir)
		if err != nil {
			return err
		}

		for _, item := range items {
			if !item.IsDir() {
				continue
			}

			logInfo("hook-update", "updating repository hooks: "+item.Name())

			for hook, script := range c.Hooks {
				hookPath := path.Join(c.Dir, item.Name(), "hooks", hook)
				if err := ioutil.WriteFile(hookPath, []byte(script), 0755); err != nil {
					logError("hook-update", err)
					return err
				}
			}
		}
	}

	return nil
}
