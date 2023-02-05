package gitkit_test

import (
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/assert"
)

type gitReceiveMock struct {
	name            string
	masterOnly      bool
	allowedBranches []string
	ref             string
	isErr           bool
}

func TestMasterOnly(t *testing.T) {
	testCases := []gitReceiveMock{
		{
			name:       "push to master, no error",
			masterOnly: true,
			ref:        "refs/heads/master",
			isErr:      false,
		},
		{
			name:       "push to a branch, should trigger error",
			masterOnly: true,
			ref:        "refs/heads/branch",
			isErr:      true,
		},
	}

	for _, tc := range testCases {
		r := &gitkit.Receiver{
			MasterOnly: tc.masterOnly,
		}

		err := r.CheckAllowedBranch(&gitkit.HookInfo{
			Ref: tc.ref,
		})

		if !tc.isErr {
			assert.NoError(t, err, "expected no error: %s", tc.name)
		} else {
			assert.Error(t, err, "expected an error: %s", tc.name)
		}
	}
}

func TestAllowedBranches(t *testing.T) {
	testCases := []gitReceiveMock{
		{
			name:            "push to master, no error",
			allowedBranches: []string{"refs/heads/master"},
			ref:             "refs/heads/master",
			isErr:           false,
		},
		{
			name:            "push to a branch, should trigger error",
			allowedBranches: []string{"refs/heads/master"},
			ref:             "refs/heads/some-branch",
			isErr:           true,
		},
		{
			name:            "push to another-branch",
			allowedBranches: []string{"refs/heads/another-branch"},
			ref:             "refs/heads/another-branch",
			isErr:           false,
		},
		{
			name:            "push to main and only allow main",
			allowedBranches: []string{"refs/heads/main"},
			ref:             "refs/heads/main",
			isErr:           false,
		},
	}

	for _, tc := range testCases {
		r := &gitkit.Receiver{
			AllowedBranches: tc.allowedBranches,
		}

		err := r.CheckAllowedBranch(&gitkit.HookInfo{
			Ref: tc.ref,
		})

		if !tc.isErr {
			assert.NoError(t, err, "expected no error: %s", tc.name)
		} else {
			assert.Error(t, err, "expected an error: %s", tc.name)
		}
	}
}
