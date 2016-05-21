package gitkit

type Config struct {
	Dir        string            // Directory that contains repositories
	GitPath    string            // Path to git binary
	AutoCreate bool              // Automatically create repostories
	Hooks      map[string][]byte // Scripts for hooks/* directory
	Auth       bool              // Require authentication
}
