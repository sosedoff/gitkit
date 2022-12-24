package gitkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadHookInput(t *testing.T) {
	input := "e285100b636ac67fa28d85685072158edaa01685 a3d33576d686e7dc1d90ec4b1a6e94e760a893b2 refs/heads/master\n"
	info, err := ReadHookInput(strings.NewReader(input))

	assert.NoError(t, err)
	assert.Equal(t, "e285100b636ac67fa28d85685072158edaa01685", info.OldRev)
	assert.Equal(t, "a3d33576d686e7dc1d90ec4b1a6e94e760a893b2", info.NewRev)
	assert.Equal(t, "refs/heads/master", info.Ref)
	assert.Equal(t, "heads", info.RefType)
	assert.Equal(t, "master", info.RefName)
}

func TestHookAction(t *testing.T) {
	examples := map[string]HookInfo{
		"branch.create": {
			OldRev:  "0000000000000000000000000000000000000000",
			NewRev:  "e285100b636ac67fa28d85685072158edaa01685",
			RefType: "heads",
		},
		"branch.delete": {
			OldRev:  "e285100b636ac67fa28d85685072158edaa01685",
			NewRev:  "0000000000000000000000000000000000000000",
			RefType: "heads",
		},
		"branch.push": {
			OldRev:  "e285100b636ac67fa28d85685072158edaa01685",
			NewRev:  "a3d33576d686e7dc1d90ec4b1a6e94e760a893b2",
			RefType: "heads",
		},
		"tag.create": {
			OldRev:  "0000000000000000000000000000000000000000",
			NewRev:  "e285100b636ac67fa28d85685072158edaa01685",
			RefType: "tags",
		},
		"tag.delete": {
			OldRev:  "e285100b636ac67fa28d85685072158edaa01685",
			NewRev:  "0000000000000000000000000000000000000000",
			RefType: "tags",
		},
	}

	for expected, hook := range examples {
		assert.Equal(t, expected, parseHookAction(hook))
	}
}
