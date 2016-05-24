package gitkit

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/satori/go.uuid"
)

type Receiver struct {
	MasterOnly  bool
	TmpDir      string
	HandlerFunc func(*HookInfo, string) error
}

func ReadCommitMessage(sha string) (string, error) {
	buff, err := exec.Command("git", "show", "-s", "--format=%B", sha).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buff)), nil
}

func (r *Receiver) Handle(reader io.Reader) error {
	hook, err := ReadHookInput(reader)
	if err != nil {
		return err
	}

	if r.MasterOnly && hook.Ref != "refs/heads/master" {
		return fmt.Errorf("cant push to non-master branch")
	}

	tmpDir := path.Join(r.TmpDir, uuid.NewV4().String())
	if err := os.Mkdir(tmpDir, 0774); err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archiveCmd := fmt.Sprintf("git archive '%s' | tar -x -C '%s'", hook.NewRev, tmpDir)
	buff, err := exec.Command("bash", "-c", archiveCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cant archive repo: %s", buff)
	}

	if r.HandlerFunc != nil {
		return r.HandlerFunc(hook, tmpDir)
	}

	return nil
}
