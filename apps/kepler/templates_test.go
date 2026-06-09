package main

import "testing"

func TestBuildTemplates(t *testing.T) {
	tmpls, err := buildTemplates()
	if err != nil {
		t.Fatalf("buildTemplates: %v", err)
	}
	for _, name := range []string{"index.html", "repo.html", "tree.html", "blob.html", "log.html", "commit.html", "refs.html", "404.html"} {
		if _, ok := tmpls[name]; !ok {
			t.Errorf("missing template: %s", name)
		}
	}
}

func TestParseArchiveName(t *testing.T) {
	cases := []struct {
		in     string
		ref    string
		format string
		ok     bool
	}{
		{"main.tar.gz", "main", "tar.gz", true},
		{"v1.0.0.zip", "v1.0.0", "zip", true},
		{"abc.tgz", "abc", "tar.gz", true},
		{"main", "", "", false},
	}
	for _, c := range cases {
		ref, format, ok := parseArchiveName(c.in)
		if ok != c.ok || ref != c.ref || format != c.format {
			t.Errorf("%q → (%q, %q, %v), want (%q, %q, %v)", c.in, ref, format, ok, c.ref, c.format, c.ok)
		}
	}
}
