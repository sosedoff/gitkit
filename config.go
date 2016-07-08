package gitkit

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

// PubKeyLookupFunc returns a PublicKey when provided the user attempting to log
// in via SSH.  The default function parses the `git` user's authorized_keys file.
type PubKeyLookupFunc func(string) (*PublicKey, error)

// HTTPAuthFunc should return true when presented an authorized user
type HTTPAuthFunc func(Credential, *Request) (bool, error)

// SSHAuthFunc should return true when presented with an authorized user.  The
// first argument is the keyID of the associated public key and the second
// is the GitCommand
type SSHAuthFunc func(string, *GitCommand) (bool, error)

// Option is a functional option type used to configure the server
type Option func(*config)

type config struct {
	// Git configuration
	GitPath      string            // Path to git binary
	Dir          string            // Directory that contains repositories
	AutoCreate   bool              // Automatically create repostories
	AutoHooks    bool              // Automatically setup git hooks
	Hooks        map[string][]byte // Scripts for hooks/* directory
	UseNamespace bool

	// HTTP server configuration
	EnableHTTP   bool // Enable HTTP service
	HTTPAuth     bool // Require authentication
	TLSKey       string
	TLSCert      string
	HTTPAuthFunc HTTPAuthFunc
	HTTPPort     int

	// SSH server configuration
	EnableSSH     bool   // Enable SSH service
	KeyDir        string // Directory for server ssh keys. Only used in SSH strategy.
	SSHKeyName    string
	GitUser       string // User for ssh connections
	SSHAuth       bool
	SSHAuthFunc   SSHAuthFunc
	SSHPubKeyFunc PubKeyLookupFunc
	SSHPort       int
}

// EnableAutoCreate will automatically create a new repository on the first push
// when set to true (default).
func EnableAutoCreate(flag bool) Option {
	return func(c *config) {
		c.AutoCreate = flag
	}
}

// SetGitPath sets the path to the git binary.  Defaults to `git` Assuming
// the git binary is in your $PATH
func SetGitPath(path string) Option {
	return func(c *config) {
		_, err := os.Stat(path)
		if err != nil || os.IsNotExist(err) {
			log.Fatal("Git binary does not exist at path provided.")
		}
	}
}

// UseNamespace allows Github-style namespaced repositories like user/repo.git.
// Namespaces are only allowed one level deep.  When set to false, repos are
// stored at the top level of the repository directory with no namespacing.
func UseNamespace(flag bool) Option {
	return func(c *config) {
		c.UseNamespace = flag
	}
}

// SetGitUser sets the allowed user when connecting via SSH.  Defaults to `git`.
func SetGitUser(user string) Option {
	return func(c *config) {
		c.GitUser = user
	}
}

// UseSSHAuthFunc will set a custom authorization function for SSH services.  The
// default configuration contains no authorization.  This function is called
// after the connection is accepted, but before any changes to the repository
// are allowed.  The SSHAuthFunc receives the KeyID associated with the logged
// in user and the GitCommand that will be run.  Setting this value will
// enable SSH services and enforce authorization for all requests.
func UseSSHAuthFunc(fun SSHAuthFunc) Option {
	return func(c *config) {
		c.SSHAuth = true
		c.EnableSSH = true
		c.SSHAuthFunc = fun
	}
}

// AddHooks will set hook scripts in all existing and new repositories. Pass
// a map[string][]byte with the name of the hook as the key and the content
// of the script as a []byte.
func AddHooks(hooks ...map[string][]byte) Option {
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
func AddHooksFromFiles(hooks ...map[string]string) Option {
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

// SetRepoPath sets the full path from which to serve the git repositories.
// Defaults to the current directory.
func SetRepoPath(repoPath string) Option {
	return func(c *config) {
		if _, err := os.Stat(repoPath); err != nil || os.IsNotExist(err) {
			log.Printf("Git repo path %s does not exist.  It will be created.\n", repoPath)
		}
		c.Dir = repoPath
	}
}

// EnableHTTP will enable smart-HTTP services with a default configuration on port
// 8080 when set to true (default).
func EnableHTTP(flag bool) Option {
	return func(c *config) {
		c.EnableHTTP = flag
	}
}

// EnableSSH will enable services over SSH with a default configuration on port
// 2222 when set to true (default).
func EnableSSH(flag bool) Option {
	return func(c *config) {
		c.EnableSSH = flag
	}
}

// SSHPort will set the port used to provide services over SSH.  Setting this value will
// automatically enable SSH services.
func SSHPort(port int) Option {
	return func(c *config) {
		c.EnableSSH = true
		c.SSHPort = port
	}
}

// HTTPPort will set the port used to provide HTTP services.  Settings this value
// will automatically enable HTTP services.
func HTTPPort(port int) Option {
	return func(c *config) {
		c.EnableHTTP = true
		c.HTTPPort = port
	}
}

// UseSSHPubKeyFunc will set a custom lookup function for a user's public key.
func UseSSHPubKeyFunc(fun PubKeyLookupFunc) Option {
	return func(c *config) {
		c.SSHPubKeyFunc = fun
		c.EnableSSH = true
		c.SSHAuth = true
	}
}

// UseHTTPAuthFunc will set a custom authorization function for HTTP services.  The
// default configuration contains no authorization.  Setting this value will
// enable HTTP services and enforce authorization for all requests.
func UseHTTPAuthFunc(fun HTTPAuthFunc) Option {
	return func(c *config) {
		c.HTTPAuth = true
		c.EnableHTTP = true
		c.HTTPAuthFunc = fun
	}
}

// UseSSHKey will set the SSH server key.  If not provided, the default
// configuration will generate a new key on starting the server.
func UseSSHKey(dir string, name string) Option {
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
func UseTLS(cert string, key string) Option {
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
