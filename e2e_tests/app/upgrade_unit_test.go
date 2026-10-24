package e2e_app

import "testing"

func TestReleaseContainsUpgrade(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		releaseVersion string
		upgradeName    string
		want           bool
	}{
		"matching release":       {releaseVersion: "30.0.0", upgradeName: "v30", want: true},
		"matching prefixed tag":  {releaseVersion: "v30.0.0", upgradeName: "v30", want: true},
		"new upgrade":            {releaseVersion: "30.0.0", upgradeName: "v31", want: false},
		"older upgrade":          {releaseVersion: "31.0.0", upgradeName: "v30", want: true},
		"v32 upgrade from v31":   {releaseVersion: "v31.0.2", upgradeName: "v32", want: false},
		"v32 already released":   {releaseVersion: "v32.0.0", upgradeName: "v32", want: true},
		"multi-digit major":      {releaseVersion: "100.2.3", upgradeName: "v100", want: true},
		"different upgrade name": {releaseVersion: "30.0.0", upgradeName: "release-30", want: false},
		"invalid release":        {releaseVersion: "latest", upgradeName: "v30", want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := releaseContainsUpgrade(tc.releaseVersion, tc.upgradeName); got != tc.want {
				t.Fatalf("releaseContainsUpgrade(%q, %q) = %t, want %t", tc.releaseVersion, tc.upgradeName, got, tc.want)
			}
		})
	}
}
