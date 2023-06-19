package gitkit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

var (
	ErrAlreadyStarted = errors.New("server has already been started")
	ErrNoListener     = errors.New("cannot call Serve() before Listen()")
)

type PublicKey struct {
	Id          string
	Name        string
	Fingerprint string
	Content     string
}

type SSH struct {
	listener net.Listener

	sshconfig             *ssh.ServerConfig
	config                *Config
	PublicKeyLookupFunc   func(string) (*PublicKey, error)
	ReposForKeyLookupFunc func(*PublicKey) ([]string, error)
}

func NewSSH(config Config) *SSH {
	s := &SSH{config: &config}

	// Use PATH if full path is not specified
	if s.config.GitPath == "" {
		s.config.GitPath = "git"
	}
	return s
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func cleanCommand(cmd string) string {
	i := strings.Index(cmd, "git")
	if i == -1 {
		return cmd
	}
	return cmd[i:]
}

func execCommandBytes(cmdname string, args ...string) ([]byte, []byte, error) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cmd := exec.Command(cmdname, args...)
	cmd.Stdout = bufOut
	cmd.Stderr = bufErr

	err := cmd.Run()
	return bufOut.Bytes(), bufErr.Bytes(), err
}

func execCommand(cmdname string, args ...string) (string, string, error) {
	bufOut, bufErr, err := execCommandBytes(cmdname, args...)
	return string(bufOut), string(bufErr), err
}

func (s *SSH) handleConnection(exts map[string]string, chans <-chan ssh.NewChannel) {
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		ch, reqs, err := newChan.Accept()
		if err != nil {
			log.Printf("error accepting channel: %v", err)
			continue
		}

		go func(in <-chan *ssh.Request) {
			defer ch.Close()

			for req := range in {
				payload := cleanCommand(string(req.Payload))

				switch req.Type {
				case "env":
					log.Printf("ssh: incoming env request: %s\n", payload)

					args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
					if len(args) != 2 {
						log.Printf("env: invalid env arguments: '%#v'", args)
						continue
					}

					args[0] = strings.TrimLeft(args[0], "\x04")
					if len(args[0]) == 0 {
						log.Printf("env: invalid key from payload: %s", payload)
						continue
					}

					_, _, err := execCommandBytes("env", args[0]+"="+args[1])
					if err != nil {
						log.Printf("env: %v", err)
						return
					}
				case "exec":
					log.Printf("ssh: incoming exec request: %s\n", payload)

					cmdName := strings.TrimLeft(payload, "'()")
					log.Printf("ssh: payload '%v'", cmdName)

					if strings.HasPrefix(cmdName, "\x00") {
						cmdName = strings.Replace(cmdName, "\x00", "", -1)[1:]
					}

					gitcmd, err := ParseGitCommand(cmdName)
					if err != nil {
						log.Println("ssh: error parsing command:", err)
						ch.Write([]byte("Invalid command.\r\n"))
						return
					}

					if !repoExists(filepath.Join(s.config.Dir, gitcmd.Repo)) && s.config.AutoCreate == true {
						err := initRepo(gitcmd.Repo, s.config)
						if err != nil {
							logError("repo-init", err)
							return
						}
					}

					cmd := exec.Command(gitcmd.Command, gitcmd.Repo)
					cmd.Dir = s.config.Dir

					envVariables := os.Environ()
					// append data via ssh.Permissions.Extensions
					for k, v := range exts {
						log.Println("k=" + k + ", v=" + v)
						envVariables = append(envVariables, "GITKIT_"+strings.ToUpper(k)+"="+v)
					}
					cmd.Env = envVariables

					// cmd.Env = append(os.Environ(), "SSH_ORIGINAL_COMMAND="+cmdName)

					stdout, err := cmd.StdoutPipe()
					if err != nil {
						log.Printf("ssh: cant open stdout pipe: %v", err)
						return
					}

					stderr, err := cmd.StderrPipe()
					if err != nil {
						log.Printf("ssh: cant open stderr pipe: %v", err)
						return
					}

					input, err := cmd.StdinPipe()
					if err != nil {
						log.Printf("ssh: cant open stdin pipe: %v", err)
						return
					}

					if err = cmd.Start(); err != nil {
						log.Printf("ssh: start error: %v", err)
						return
					}

					req.Reply(true, nil)
					go io.Copy(input, ch)
					io.Copy(ch, stdout)
					io.Copy(ch.Stderr(), stderr)

					if err = cmd.Wait(); err != nil {
						log.Printf("ssh: command failed: %v", err)
						return
					}

					ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				default:
					ch.Write([]byte("Unsupported request type.\r\n"))
					log.Println("ssh: unsupported req type:", req.Type)
					return
				}
			}
		}(reqs)
	}
}

func (s *SSH) setup() error {
	if s.sshconfig != nil {
		return nil
	}
	config := &ssh.ServerConfig{
		ServerVersion: fmt.Sprintf("SSH-2.0-gitkit %s", Version),
	}

	if s.config.KeyDir == "" {
		return fmt.Errorf("key directory is not provided")
	}

	if !s.config.Auth {
		config.NoClientAuth = true
	} else {
		if s.PublicKeyLookupFunc == nil {
			return fmt.Errorf("public key lookup func is not provided")
		}

		if s.ReposForKeyLookupFunc == nil {
			log.Println("no repository callback, an authorized user may access any repositories")
		}

		config.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			pkey, err := s.PublicKeyLookupFunc(strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
			if err != nil {
				return nil, err
			}

			if pkey == nil {
				return nil, fmt.Errorf("auth handler did not return a key")
			}

			var repos []string

			if s.ReposForKeyLookupFunc != nil {
				repos, err = s.ReposForKeyLookupFunc(pkey)
				if err != nil {
					return nil, err
				}
			}

			return &ssh.Permissions{
				Extensions: map[string]string{
					"key":          pkey.Id,
					"fingerprint":  pkey.Fingerprint,
					"name":         pkey.Name,
					"repositories": strings.Join(repos, ","),
				},
			}, nil
		}
	}

	keyPath := s.config.KeyPath()

	k := NewKey(s.config.KeyDir)
	if !fileExists(keyPath) {
		if err := k.CreateRSA(); err != nil {
			return err
		}
	}

	private, err := k.GetRSA()
	if err != nil {
		return err
	}

	config.AddHostKey(private)
	s.sshconfig = config
	return nil
}

func (s *SSH) Listen(bind string) error {
	if s.listener != nil {
		return ErrAlreadyStarted
	}

	if err := s.setup(); err != nil {
		return err
	}

	if err := s.config.Setup(); err != nil {
		return err
	}

	var err error
	s.listener, err = net.Listen("tcp", bind)
	if err != nil {
		return err
	}

	return nil
}

func (s *SSH) Serve() error {
	if s.listener == nil {
		return ErrNoListener
	}

	for {
		// wait for connection or Stop()
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			log.Printf("ssh: handshaking for %s", conn.RemoteAddr())

			sConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshconfig)
			if err != nil {
				if err == io.EOF {
					log.Printf("ssh: handshaking was terminated: %v", err)
				} else {
					log.Printf("ssh: error on handshaking: %v", err)
				}
				return
			}

			log.Printf("ssh: connection from %s (%s)", sConn.RemoteAddr(), sConn.ClientVersion())

			if s.config.Auth && s.config.GitUser != "" && sConn.User() != s.config.GitUser {
				sConn.Close()
				return
			}

			var exts map[string]string
			if sConn.Permissions != nil {
				exts = sConn.Permissions.Extensions
			}

			go ssh.DiscardRequests(reqs)
			go s.handleConnection(exts, chans)
		}()
	}
}

func (s *SSH) ListenAndServe(bind string) error {
	if err := s.Listen(bind); err != nil {
		return err
	}
	return s.Serve()
}

// Stop stops the server if it has been started, otherwise it is a no-op.
func (s *SSH) Stop() error {
	if s.listener == nil {
		return nil
	}
	defer func() {
		s.listener = nil
	}()

	return s.listener.Close()
}

// Address returns the network address of the listener. This is in
// particular useful when binding to :0 to get a free port assigned by
// the OS.
func (s *SSH) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// SetSSHConfig can be used to set custom SSH Server settings.
func (s *SSH) SetSSHConfig(cfg *ssh.ServerConfig) {
	s.sshconfig = cfg
}

// SetListener can be used to set custom Listener.
func (s *SSH) SetListener(l net.Listener) {
	s.listener = l
}
