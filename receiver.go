package gitkit

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/gofrs/uuid"
	"golang.org/x/exp/slices"
)

const ZeroSHA = "0000000000000000000000000000000000000000"

type Receiver struct {
	Debug       bool
	MasterOnly  bool
	AllowedRefs []string
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

func IsForcePush(hook *HookInfo) (bool, error) {
	// New branch or tag OR deleted branch or tag
	if hook.OldRev == ZeroSHA || hook.NewRev == ZeroSHA {
		return false, nil
	}

	out, err := exec.Command("git", "merge-base", hook.OldRev, hook.NewRev).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git merge base failed: %s", out)
	}

	base := strings.TrimSpace(string(out))

	// Non fast-forwarded, meaning force
	return base != hook.OldRev, nil
}

func (r *Receiver) CheckAllowedBranch(hook *HookInfo) error {
	if r.MasterOnly { // for BC
		r.AllowedRefs = append(r.AllowedRefs, "refs/heads/master")
	}

	if len(r.AllowedRefs) == 0 {
		return nil
	}

	if !slices.Contains(r.AllowedRefs, hook.Ref) {
		return fmt.Errorf("cannot push branch, allowed branches: %s", strings.Join(r.AllowedRefs, ", "))
	}

	return nil
}

func (r *Receiver) Handle(reader io.Reader) error {
	hook, err := ReadHookInput(reader)
	if err != nil {
		return err
	}

	if err = r.CheckAllowedBranch(hook); err != nil {
		return err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("error generating new uuid: %v", err)
	}

	tmpDir := path.Join(r.TmpDir, id.String())
	if err := os.MkdirAll(tmpDir, 0774); err != nil {
		return err
	}

	// Cleanup temp directory unless we're in debug mode
	if !r.Debug {
		defer os.RemoveAll(tmpDir)
	}

	archiveCmd := fmt.Sprintf("git archive '%s' | tar -x -C '%s'", hook.NewRev, tmpDir)
	buff, err := exec.Command("bash", "-c", archiveCmd).CombinedOutput()
	if err != nil {
		if len(buff) > 0 && strings.Contains(string(buff), "Damaged tar archive") {
			return fmt.Errorf("Error: repository might be empty!")
		}
		return fmt.Errorf("cant archive repo: %s", buff)
	}

	if r.HandlerFunc != nil {
		return r.HandlerFunc(hook, tmpDir)
	}

	return nil
}
