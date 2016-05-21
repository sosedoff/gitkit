package gitkit

type Config struct {
	Host       string            // Specifies host interface to listed on
	Port       int               // Server port
	Dir        string            // Directory that contains repositories
	GitPath    string            // Path to git binary
	AutoCreate bool              // Automatically create repostories
	Hooks      map[string][]byte // Scripts for hooks/* directory
}
