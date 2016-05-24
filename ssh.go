package gitkit

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

type PublicKey struct {
	Id          string
	Name        string
	Fingerprint string
	Content     string
}

type SSH struct {
	sshconfig           *ssh.ServerConfig
	config              *Config
	PublicKeyLookupFunc func(string) (*PublicKey, error)
}

func NewSSH(config Config) *SSH {
	return &SSH{config: &config}
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

func (s *SSH) handleConnection(keyID string, chans <-chan ssh.NewChannel) {
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
					args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
					if len(args) != 2 {
						log.Printf("env: invalid env arguments: '%#v'", args)
						continue
					}

					args[0] = strings.TrimLeft(args[0], "\x04")

					_, _, err := execCommandBytes("env", args[0]+"="+args[1])
					if err != nil {
						log.Printf("env: %v", err)
						return
					}
				case "exec":
					cmdName := strings.TrimLeft(payload, "'()")
					args := strings.Split(cmdName, " ")
					log.Printf("ssh: payload '%v'", cmdName)

					if strings.HasPrefix(cmdName, "\x00") {
						cmdName = strings.Replace(cmdName, "\x00", "", -1)[1:]
					}

					prefix := args[0]
					if !(prefix == "git-receive-pack" || prefix == "git-upload-pack") {
						ch.Write([]byte("Only git commmands are supported.\r\n"))
						return
					}

					cmd := exec.Command(prefix, strings.Replace(args[1], "'", "", 2))
					cmd.Dir = s.config.Dir
					cmd.Env = append(os.Environ(), "GITKIT_KEY="+keyID)
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
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Reject all incoming request if no key lookup is defined
			if s.PublicKeyLookupFunc == nil {
				log.Println("ssh: no public key lookup function is defined")
				return nil, fmt.Errorf("no key lookup func")
			}

			// Lookup public key with user-defined function
			pkey, err := s.PublicKeyLookupFunc(strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
			if err != nil {
				return nil, err
			}

			return &ssh.Permissions{Extensions: map[string]string{"key-id": pkey.Id}}, nil
		},
	}

	if s.config.KeyDir == "" {
		return fmt.Errorf("ssh: key directory is not provided")
	}

	keypath := filepath.Join(s.config.KeyDir, "gitkit.rsa")

	// Automatically create key pair for server use
	if !fileExists(keypath) {
		log.Println("ssh: creating a new server key")
		os.MkdirAll(filepath.Dir(keypath), os.ModePerm)
		_, stderr, err := execCommand("ssh-keygen", "-f", keypath, "-t", "rsa", "-N", "")
		if err != nil {
			return fmt.Errorf("Fail to generate private key: %v - %s", err, stderr)
		}
	} else {
		log.Println("ssh: server key exists, skipping creation")
	}

	privateBytes, err := ioutil.ReadFile(keypath)
	if err != nil {
		return err
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return err
	}

	config.AddHostKey(private)
	s.sshconfig = config
	return nil
}

func (s *SSH) ListenAndServe(bind string) error {
	if err := s.setup(); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("ssh: error accepting incoming connection: %v", err)
			continue
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
			go ssh.DiscardRequests(reqs)
			go s.handleConnection(sConn.Permissions.Extensions["key-id"], chans)
		}()
	}
}
