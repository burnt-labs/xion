package testlib

import "testing"

func TestParseStableReleaseVersion(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want releaseVersion
		ok   bool
	}{
		{name: "prefixed", tag: "v30.1.2", want: releaseVersion{major: 30, minor: 1, patch: 2}, ok: true},
		{name: "unprefixed", tag: "29.0.1", want: releaseVersion{major: 29, minor: 0, patch: 1}, ok: true},
		{name: "prerelease", tag: "v30.0.0-rc1", ok: false},
		{name: "incomplete", tag: "v30", ok: false},
		{name: "invalid", tag: "latest", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseStableReleaseVersion(tt.tag)
			if ok != tt.ok {
				t.Fatalf("parseStableReleaseVersion(%q) validity = %v, want %v", tt.tag, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("parseStableReleaseVersion(%q) = %+v, want %+v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestReleaseVersionLess(t *testing.T) {
	tests := []struct {
		name  string
		left  releaseVersion
		right releaseVersion
		want  bool
	}{
		{name: "major", left: releaseVersion{major: 29, minor: 9, patch: 9}, right: releaseVersion{major: 30}, want: true},
		{name: "minor", left: releaseVersion{major: 29, minor: 0, patch: 9}, right: releaseVersion{major: 29, minor: 1}, want: true},
		{name: "patch", left: releaseVersion{major: 29, minor: 0}, right: releaseVersion{major: 29, minor: 0, patch: 1}, want: true},
		{name: "equal", left: releaseVersion{major: 29, minor: 0, patch: 1}, right: releaseVersion{major: 29, minor: 0, patch: 1}, want: false},
		{name: "greater", left: releaseVersion{major: 30}, right: releaseVersion{major: 29, minor: 9, patch: 9}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.left.less(tt.right); got != tt.want {
				t.Fatalf("(%+v).less(%+v) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}
