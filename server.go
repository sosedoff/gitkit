package gitkit

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

// CombinedServer contains the configuration for both the SSH and HTTP servers
// which can be run concurrently if both are configured.
type CombinedServer struct {
	config     config
	HTTPserver *Server
	SSHServer  *SSH
}

// New creates a new server from the provided configuration.  The server
// starts from a default configuration and modifies the config using the
// options you provide.  The server can then be started with server.Run()
// See the examples directory for common configuration tasks like running
// both the HTTP and SSH servers concurrently, changing ports, TLS config,
// and more.
func New(options ...func(*config)) CombinedServer {

	defaultNoAuthKeyLookup := func(content string) (*PublicKey, error) {
		return &PublicKey{Id: "12345"}, nil
	}

	// start with sensible defaults and then apply user custom options
	// to override defaults
	config := config{
		// Git configuration
		GitPath:      "git",
		Dir:          "./",
		AutoCreate:   true,
		AutoHooks:    false,
		Hooks:        nil,
		UseNamespace: false,

		// HTTP server configuration
		EnableHTTP:   true,
		HTTPAuth:     false,
		TLSKey:       "",
		TLSCert:      "",
		HTTPAuthFunc: nil,
		HTTPPort:     8080,

		// SSH server configuration
		EnableSSH:     true,
		KeyDir:        ".keys",
		SSHKeyName:    "gitkit",
		GitUser:       "git",
		SSHAuth:       true,
		SSHAuthFunc:   nil,
		SSHPubKeyFunc: defaultNoAuthKeyLookup,
		SSHPort:       2222,
	}

	for _, option := range options {
		option(&config)
	}

	server := CombinedServer{
		config: config,
	}

	if config.EnableHTTP {
		server.HTTPserver = NewHTTP(config)
		err := server.HTTPserver.Setup()
		if err != nil {
			log.Fatal(err)
		}
	}
	if config.EnableSSH {
		server.SSHServer = NewSSH(config)
	}
	return server
}

// Run inspects the provided configuration and will run the HTTP and SSH server
// concurrently if both are configured
func (s *CombinedServer) Run() {

	if (s.config.EnableHTTP) && (s.config.EnableSSH) {
		muxH := http.NewServeMux()
		muxH.HandleFunc("/", s.HTTPserver.ServeHTTP)

		log.Printf("Git smart HTTP server running on port %d\n", s.config.HTTPPort)
		log.Printf("Git SSH server running on port %d\n", s.config.SSHPort)
		go func() {
			port := strings.Join([]string{"127.0.0.1", strconv.Itoa(s.config.HTTPPort)}, ":")
			if s.config.TLSKey == "" || s.config.TLSCert == "" {
				log.Fatal(http.ListenAndServe(port, muxH))
			} else {
				log.Fatal(http.ListenAndServeTLS(port, s.config.TLSCert, s.config.TLSKey, muxH))
			}
		}()
		log.Fatal(s.SSHServer.ListenAndServe(strings.Join([]string{"127.0.0.1", strconv.Itoa(s.config.SSHPort)}, ":")))
	}
	if s.config.EnableHTTP {
		log.Printf("Git smart HTTP server running on port %d\n", s.config.HTTPPort)
		muxH := http.NewServeMux()
		muxH.HandleFunc("/", s.HTTPserver.ServeHTTP)
		port := strings.Join([]string{"127.0.0.1", strconv.Itoa(s.config.HTTPPort)}, ":")
		if s.config.TLSKey == "" || s.config.TLSCert == "" {
			log.Fatal(http.ListenAndServe(port, muxH))
		} else {
			log.Fatal(http.ListenAndServeTLS(port, s.config.TLSCert, s.config.TLSKey, muxH))
		}
	}
	if s.config.EnableSSH {
		log.Printf("Git SSH server running on port %d\n", s.config.SSHPort)
		log.Fatal(s.SSHServer.ListenAndServe(strings.Join([]string{"127.0.0.1", strconv.Itoa(s.config.SSHPort)}, ":")))

	}
	return
}
