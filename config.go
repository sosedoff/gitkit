package gitkit

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// PubKeyLookupFunc returns a PublicKey when provided the user attempting to log
// in via SSH.  The default function parses the `git` user's authorized_keys file.
type PubKeyLookupFunc func(string) (*PublicKey, error)

// AuthFunc should return true when presented an authorized user
type AuthFunc func(Credential, *Request) (bool, error)

type config struct {
	KeyDir        string            // Directory for server ssh keys. Only used in SSH strategy.
	Dir           string            // Directory that contains repositories
	GitPath       string            // Path to git binary
	GitUser       string            // User for ssh connections
	AutoCreate    bool              // Automatically create repostories
	AutoHooks     bool              // Automatically setup git hooks
	Hooks         map[string][]byte // Scripts for hooks/* directory
	Auth          bool              // Require authentication
	EnableHTTP    bool              // Enable HTTP service
	EnableSSH     bool              // Enable SSH service
	TLSKey        string
	TLSCert       string
	AuthFunc      AuthFunc
	SSHPubKeyFunc PubKeyLookupFunc
	SSHKeyName    string
	HTTPPort      int
	SSHPort       int
}

// AutoCreate will automatically create a new repository on the first push
// when set to true (default).
func AutoCreate(flag bool) {
	return func(c *config) {
		c.AutoCreate = flag
	}
}

// AddHooks will set hook scripts in all existing and new repositories. Pass
// a map[string][]byte with the name of the hook as the key and the content
// of the script as a []byte.
func AddHooks(hooks ...map[string][]byte) {
	return func(c *config) {
		c.AutoHooks = true
		if c.Hooks == nil {
			c.Hooks = make(map[string][]byte)
		}
		for _, hook := range hooks {
			for key, value := range hook {
				c.Hooks[key] = value
			}
		}
	}
}

// AddHooksFromFiles will set hook scripts in all existing and new
// repositories from a set of existing script files.  Pass a map[string]string
// containing the name of the hook as the key and the full path to the script
// as the value.
// ```
// hooks := map[string]string{
//	"pre-receive": "/path/to/pre-receive.sh",
// }
// server := gitkit.New(hooks, otherOptions...)
// ```
func AddHooksFromFiles(hooks ...map[string]string) {
	return func(c *config) {
		c.AutoHooks = true
		if c.Hooks == nil {
			c.Hooks = make(map[string][]byte)
		}
		for _, hook := range hooks {
			for key, file := range hook {
				script, err := ioutil.ReadFile(file)
				if err != nil {
					log.Fatal(err)
				}
				c.Hooks[key] = script
			}
		}
	}
}

// UseRepoPath sets the full path from which to serve the git repositories.
// Defaults to the current directory.
func UseRepoPath(repoPath string) {
	return func(c *config) {
		if _, err := os.Stat(repoPath); err != nil || os.IsNotExist(err) {
			log.Warn(fmt.Sprintf("Git repo path %s does not exist.  It will be created.\n", repoPath))
		}
		c.Dir = repoPath
	}
}

// UseHTTP will enable smart-HTTP services with a default configuration on port
// 8080 when set to true (default).
func UseHTTP(flag bool) {
	return func(c *config) {
		c.EnableHTTP = flag
	}
}

// UseSSH will enable services over SSH with a default configuration on port
// 2222 when set to true (default).
func UseSSH(flag bool) {
	return func(c *config) {
		c.EnableSSH = flag
	}
}

// SSHPort will set the port used to provide services over SSH.  Setting this value will
// automatically enable SSH services.
func SSHPort(port int) {
	return func(c *config) {
		c.EnableSSH = true
		c.SSHPort = port
	}
}

// HTTPPort will set the port used to provide HTTP services.  Settings this value
// will automatically enable HTTP services.
func HTTPPort(port int) {
	return func(c *config) {
		c.EnableHTTP = true
		c.HTTPPort = port
	}
}

// UseSSHPubKeyFunc will set a custom lookup function for a user's public key.
func UseSSHPubKeyFunc(fun PubKeyLookupFunc) {
	return func(c *config) {
		c.PubKeyFunc = fun
		c.EnableSSH = true
	}
}

// UseAuthFunc will set a custom authorization function for HTTP services.  The
// default configuration contains no authorization.  Setting this value will
// enable HTTP services and enforce authorization for all requests.
func UseAuthFunc(fun AuthFunc) {
	return func(c *config) {
		c.Auth = true
		c.EnableHTTP = true
		c.AuthFunc = fun
	}
}

// UseSSHKey will set the SSH server key.  If not provided, the default
// configuration will generate a new key on starting the server.
func UseSSHKey(dir string, name string) {
	_, err := os.Stat(filepath.Join(dir, name))
	if err != nil || !os.IsExist(err) {
		log.Fatal("SSH key does not exist at path provided.")
	}
	return func(c *config) {
		c.KeyDir = dir
		c.SSHKeyName = name
		c.EnableSSH = true
	}
}

// UseTLS will set the TLS key and cert paths to provide termination of smart-HTTP
// services.
func UseTLS(key string, cert string) {
	_, errKey := os.Stat(key)
	_, errCert := os.Stat(cert)
	if errKey != nil || !os.IsExist(errKey) {
		log.Fatal("TLS key does not exist at path provided.")
	}
	if errCert != nil || !os.IsExist(errCert) {
		log.Fatal("TLS cert does not exist at path provided.")
	}
	return func(c *config) {
		c.TLSCert = cert
		c.TLSKey = key
	}
}

func (c *config) KeyPath() string {
	return filepath.Join(c.KeyDir, c.SSHKeyName)
}

func (c *config) Setup() error {
	if _, err := os.Stat(c.Dir); err != nil {
		if err = os.Mkdir(c.Dir, 0755); err != nil {
			return err
		}
	}

	if c.AutoHooks == true {
		return c.setupHooks()
	}

	return nil
}

func (c *config) setupHooks() error {
	files, err := ioutil.ReadDir(c.Dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		hooksPath := path.Join(c.Dir, file.Name(), "hooks")

		// Cleanup all existing hooks
		hookFiles, err := ioutil.ReadDir(hooksPath)
		if err == nil {
			for _, h := range hookFiles {
				os.Remove(path.Join(hooksPath, h.Name()))
			}
		}

		// Setup new hooks
		for hook, script := range c.Hooks {
			if err := ioutil.WriteFile(path.Join(hooksPath, hook), []byte(script), 0755); err != nil {
				logError("hook-update", err)
				return err
			}
		}
	}

	return nil
}
