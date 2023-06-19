package gitkit_test

import (
	"fmt"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/assert"
)

type gitReceiveMock struct {
	name            string
	masterOnly      bool
	allowedBranches []string
	ref             string
	err             error
}

func TestMasterOnly(t *testing.T) {
	testCases := []gitReceiveMock{
		{
			name:       "push to master, no error",
			masterOnly: true,
			ref:        "refs/heads/master",
			err:        nil,
		},
		{
			name:       "push to a branch, should trigger error",
			masterOnly: true,
			ref:        "refs/heads/branch",
			err:        fmt.Errorf("cannot push branch, allowed branches: refs/heads/master"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &gitkit.Receiver{
				MasterOnly: tc.masterOnly,
			}

			err := r.CheckAllowedBranch(&gitkit.HookInfo{
				Ref: tc.ref,
			})

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestAllowedBranches(t *testing.T) {
	testCases := []gitReceiveMock{
		{
			name:            "push to master, no error",
			allowedBranches: []string{"refs/heads/master"},
			ref:             "refs/heads/master",
			err:             nil,
		},
		{
			name:            "push to a branch, should trigger error",
			allowedBranches: []string{"refs/heads/master"},
			ref:             "refs/heads/some-branch",
			err:             fmt.Errorf("cannot push branch, allowed branches: refs/heads/master"),
		},
		{
			name:            "push to another-branch",
			allowedBranches: []string{"refs/heads/another-branch"},
			ref:             "refs/heads/another-branch",
			err:             nil,
		},
		{
			name:            "push to main and only allow main",
			allowedBranches: []string{"refs/heads/main"},
			ref:             "refs/heads/main",
			err:             nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &gitkit.Receiver{
				AllowedRefs: tc.allowedBranches,
			}

			err := r.CheckAllowedBranch(&gitkit.HookInfo{
				Ref: tc.ref,
			})

			assert.Equal(t, tc.err, err)
		})
	}
}
