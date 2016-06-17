package gitkit

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

type service struct {
	method  string
	suffix  string
	handler func(string, http.ResponseWriter, *Request)
	rpc     string
}

type Server struct {
	config   Config
	services []service
	AuthFunc func(Credential, *Request) (bool, error)
}

type Request struct {
	*http.Request
	RepoName string
	RepoPath string
}

func New(cfg Config) *Server {
	s := Server{config: cfg}
	s.services = []service{
		service{"GET", "/info/refs", s.getInfoRefs, ""},
		service{"POST", "/git-upload-pack", s.postRPC, "git-upload-pack"},
		service{"POST", "/git-receive-pack", s.postRPC, "git-receive-pack"},
	}

	// Use PATH if full path is not specified
	if s.config.GitPath == "" {
		s.config.GitPath = "git"
	}

	return &s
}

func (s *Server) findService(r *http.Request) *service {
	for _, svc := range s.services {
		if r.Method == svc.method && strings.HasSuffix(r.URL.Path, svc.suffix) {
			return &svc
		}
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logInfo("request", r.Method+" "+r.Host+r.URL.String())

	svc := s.findService(r)
	if svc == nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	repoName := strings.Split(r.URL.Path, "/")[1]
	repoPath := path.Join(s.config.Dir, repoName)

	req := &Request{
		Request:  r,
		RepoName: repoName,
		RepoPath: repoPath,
	}

	if s.config.Auth {
		if s.AuthFunc == nil {
			logError("auth", fmt.Errorf("no auth backend provided"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header()["WWW-Authenticate"] = []string{`Basic realm=""`}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		cred, err := parseAuth(authHeader)
		if err != nil {
			logError("auth", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		allow, err := s.AuthFunc(cred, req)
		if !allow || err != nil {
			if err != nil {
				logError("auth", err)
			}

			logError("auth", fmt.Errorf("rejected user %s", cred.Username))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	if !repoExists(req.RepoPath) && s.config.AutoCreate == true {
		err := initRepo(req.RepoName, &s.config)
		if err != nil {
			logError("repo-init", err)
		}
	}

	if !repoExists(req.RepoPath) {
		logError("repo-init", fmt.Errorf("%s does not exist", req.RepoPath))
		http.NotFound(w, r)
		return
	}

	svc.handler(svc.rpc, w, req)
}

func (s *Server) getInfoRefs(_ string, w http.ResponseWriter, r *Request) {
	context := "get-info-refs"
	rpc := r.URL.Query().Get("service")

	if !(rpc == "git-upload-pack" || rpc == "git-receive-pack") {
		http.Error(w, "Not Found", 404)
		return
	}

	cmd, pipe := gitCommand(s.config.GitPath, subCommand(rpc), "--stateless-rpc", "--advertise-refs", r.RepoPath)
	if err := cmd.Start(); err != nil {
		fail500(w, context, err)
		return
	}
	defer cleanUpProcessGroup(cmd)

	w.Header().Add("Content-Type", fmt.Sprintf("application/x-%s-advertisement", rpc))
	w.Header().Add("Cache-Control", "no-cache")
	w.WriteHeader(200)

	if err := packLine(w, fmt.Sprintf("# service=%s\n", rpc)); err != nil {
		logError(context, err)
		return
	}

	if err := packFlush(w); err != nil {
		logError(context, err)
		return
	}

	if _, err := io.Copy(w, pipe); err != nil {
		logError(context, err)
		return
	}

	if err := cmd.Wait(); err != nil {
		logError(context, err)
		return
	}
}

func (s *Server) postRPC(rpc string, w http.ResponseWriter, r *Request) {
	context := "post-rpc"
	body := r.Body

	if r.Header.Get("Content-Encoding") == "gzip" {
		var err error
		body, err = gzip.NewReader(r.Body)
		if err != nil {
			fail500(w, context, err)
			return
		}
	}

	cmd, pipe := gitCommand(s.config.GitPath, subCommand(rpc), "--stateless-rpc", r.RepoPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fail500(w, context, err)
		return
	}
	defer stdin.Close()

	if err := cmd.Start(); err != nil {
		fail500(w, context, err)
		return
	}
	defer cleanUpProcessGroup(cmd)

	if _, err := io.Copy(stdin, body); err != nil {
		fail500(w, context, err)
		return
	}

	w.Header().Add("Content-Type", fmt.Sprintf("application/x-%s-result", rpc))
	w.Header().Add("Cache-Control", "no-cache")
	w.WriteHeader(200)

	if _, err := io.Copy(newWriteFlusher(w), pipe); err != nil {
		logError(context, err)
		return
	}
	if err := cmd.Wait(); err != nil {
		logError(context, err)
		return
	}
}

func (s *Server) Setup() error {
	return s.config.Setup()
}

func initRepo(name string, config *Config) error {
	p := path.Join(config.Dir, name)

	if err := exec.Command(config.GitPath, "init", "--bare", p).Run(); err != nil {
		return err
	}

	for hook, script := range config.Hooks {
		hookPath := filepath.Join(p, "hooks", hook)

		logInfo("repo-init", fmt.Sprintf("creating %s hook for %s", hook, name))
		if err := ioutil.WriteFile(hookPath, []byte(script), 0755); err != nil {
			logError("repo-init", err)
			return err
		}
	}

	return nil
}

func repoExists(p string) bool {
	_, err := os.Stat(path.Join(p, "objects"))
	return err == nil
}

func gitCommand(name string, args ...string) (*exec.Cmd, io.Reader) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = os.Environ()

	r, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	return cmd, r
}
