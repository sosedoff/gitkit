package gitkit

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_subCommand(t *testing.T) {
	cases := map[string]string{
		"git-receive-pack": "receive-pack",
		"git-upload-pack":  "upload-pack",
		"git-foobar":       "foobar",
		"git":              "git",
		"foobar":           "foobar",
	}

	for example, expected := range cases {
		result := subCommand(example)
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	}
}

func Test_packFlush(t *testing.T) {
	w := bytes.NewBuffer([]byte{})
	err := packFlush(w)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if w.String() != "0000" {
		t.Errorf("Expected 0000, got %v", w.String())
	}
}

func Test_packLine(t *testing.T) {
	cases := map[string]string{
		"":     "0004",
		"0":    "00050",
		"10":   "000610",
		"100":  "0007100",
		"1000": "00081000",
	}

	w := bytes.NewBuffer([]byte{})

	for example, expected := range cases {
		w.Reset()
		err := packLine(w, example)

		assert.NoError(t, err)
		assert.Equal(t, expected, w.String())
	}
}

func Test_getNamespaceAndRepo(t *testing.T) {
	cases := map[string][]string{
		"":                  {"", ""},
		"/":                 {"", ""},
		"///":               {"", ""},
		"/repo":             {"", "repo"},
		"/org/repo":         {"org", "repo"},
		"/org/suborg/repo":  {"org/suborg", "repo"},
		"//org//org///repo": {"org/org", "repo"},
	}

	for example, expected := range cases {
		namespace, repo := getNamespaceAndRepo(example)

		assert.Equal(t, expected[0], namespace)
		assert.Equal(t, expected[1], repo)
	}
}
