package mqtt

import "testing"

func TestMatch(t *testing.T) {
	for _, run := range []struct {
		template string
		topic    string
		want     bool
	}{
		{"/#", "/a", true},
		{"/+", "/a", true},
		{"/+", "/aa", true},
		{"/aa", "/a", false},
		{"/a", "/aa", false},
		{"/aa", "/aa", true},
		{"/+", "/a/b", false},
		{"/+/+", "/a/b", true},
		{"/a/+/b", "/a/z/b", true},
		{"/a/b/+", "/a/b/cc", true},
		{"/a/b+", "/a/ba", true},
		{"/a/b+", "/a/aa", false},
	} {
		if have := Match(run.template, run.topic); have != run.want {
			t.Errorf("Match(%q, %q) = %t, want %t", run.template, run.topic, have, run.want)
		}
	}
}
