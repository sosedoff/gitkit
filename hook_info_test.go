package gitkit

import (
	"strings"
	"testing"
)

func Test_ReadHookInput(t *testing.T) {
	input := "e285100b636ac67fa28d85685072158edaa01685 a3d33576d686e7dc1d90ec4b1a6e94e760a893b2 refs/heads/master\n"
	info, err := ReadHookInput(strings.NewReader(input))

	if err != nil {
		t.Errorf("Unexpected error", err)
	}

	if info.OldRev != "e285100b636ac67fa28d85685072158edaa01685" {
		t.Errorf("Expected oldrev to be %s, got %s", "e285100b636ac67fa28d85685072158edaa01685", info.OldRev)
	}

	if info.NewRev != "a3d33576d686e7dc1d90ec4b1a6e94e760a893b2" {
		t.Errorf("Expected newrev to be %s, got %s", "a3d33576d686e7dc1d90ec4b1a6e94e760a893b2", info.NewRev)
	}

	if info.Ref != "refs/heads/master" {
		t.Errorf("Expected ref to be %s, got %s", "refs/heads/master", info.Ref)
	}
}
