package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

type releaseStep struct {
	Name string `json:"name"`
	Run  string `json:"run"`
}

type releaseJob struct {
	Uses            string        `json:"uses"`
	Needs           any           `json:"needs"`
	If              string        `json:"if"`
	ContinueOnError any           `json:"continue-on-error"`
	Steps           []releaseStep `json:"steps"`
}

func readReleaseWorkflow(t *testing.T, name string) map[string]releaseJob {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]releaseJob `json:"jobs"`
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	if len(w.Jobs) == 0 {
		t.Fatal("no workflow jobs")
	}
	return w.Jobs
}

func releaseNeeds(t *testing.T, j releaseJob) []string {
	t.Helper()
	switch n := j.Needs.(type) {
	case nil:
		return nil
	case string:
		return []string{n}
	case []any:
		var out []string
		for _, v := range n {
			s, ok := v.(string)
			if !ok {
				t.Fatalf("non-string job dependency: %v", v)
			}
			out = append(out, s)
		}
		return out
	default:
		t.Fatalf("unexpected dependency shape: %T", n)
		return nil
	}
}

func releaseStepByName(t *testing.T, j releaseJob, name string) (releaseStep, int) {
	t.Helper()
	for i, step := range j.Steps {
		if step.Name == name {
			return step, i
		}
	}
	t.Fatalf("missing step %q", name)
	return releaseStep{}, -1
}

func TestReleasePublicationGates(t *testing.T) {
	jobs := readReleaseWorkflow(t, "create-release.yaml")
	scan, ok := jobs["vulncheck"]
	if !ok {
		t.Fatal("release publication has no vulnerability gate")
	}
	if scan.Uses != "./.github/workflows/govulncheck.yaml" {
		t.Fatal("release does not call the existing vulnerability scanner")
	}
	if !slices.Contains(releaseNeeds(t, scan), "check-ref") {
		t.Fatal("scanner can run before ref validation")
	}
	for _, name := range []string{"vulncheck", "push-docker", "create-release"} {
		j, ok := jobs[name]
		if !ok {
			t.Fatalf("missing job %s", name)
		}
		if j.If != "" || (j.ContinueOnError != nil && j.ContinueOnError != false) {
			t.Fatalf("%s overrides the successful-dependency gate", name)
		}
	}
	for _, name := range []string{"push-docker", "create-release"} {
		deps := releaseNeeds(t, jobs[name])
		for _, gate := range []string{"lint", "update-swagger", "unit-tests", "vulncheck", "docker-scout", "e2e-tests"} {
			if !slices.Contains(deps, gate) {
				t.Errorf("%s can publish without %s succeeding", name, gate)
			}
		}
	}
}

func TestReleaseTagGuard(t *testing.T) {
	jobs := readReleaseWorkflow(t, "create-release.yaml")
	step, _ := releaseStepByName(t, jobs["check-ref"], "Require a release tag ref")
	cases := []struct {
		kind, ref string
		pass      bool
	}{
		{"tag", "v31.0.2", true},
		{"tag", "v31.0.2-rc2", true},
		{"branch", "main", false},
		{"branch", "v31.0.2", false},
		{"tag", "v31.0", false},
		{"tag", "v31.0.2-rc", false},
		{"tag", "v31.0.2;exit 0", false},
	}
	for _, c := range cases {
		t.Run(c.kind+"_"+c.ref, func(t *testing.T) {
			cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", step.Run)
			cmd.Env = append(os.Environ(), "REF_TYPE="+c.kind, "REF_NAME="+c.ref)
			output, err := cmd.CombinedOutput()
			if (err == nil) != c.pass {
				t.Fatalf("pass=%v, err=%v: %s", c.pass, err, output)
			}
		})
	}
}

func TestReleaseArtifactsRequired(t *testing.T) {
	jobs := readReleaseWorkflow(t, "exec-goreleaser.yaml")
	step, moveIndex := releaseStepByName(t, jobs["build-release"], "Move artifacts")
	_, signingIndex := releaseStepByName(t, jobs["build-release"], "Import package signing GPG key")
	if moveIndex >= signingIndex {
		t.Fatal("artifacts are validated after signing-key import")
	}
	files := []string{
		"xiond_linux_amd64_v1/bin/xiond-linux-amd64",
		"xiond_linux_arm64_v8.0/bin/xiond-linux-arm64",
		"xiond_darwin_amd64_v1/bin/xiond-darwin-amd64",
		"xiond_darwin_arm64_v8.0/bin/xiond-darwin-arm64",
	}
	runCase := func(t *testing.T, bad string, empty bool) {
		t.Helper()
		work := t.TempDir()
		input := filepath.Join(work, "input")
		for _, name := range files {
			if name == bad && !empty {
				continue
			}
			path := filepath.Join(input, "artifacts", name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			contents := []byte("presence-check fixture\n")
			if name == bad {
				contents = nil
			}
			if err := os.WriteFile(path, contents, 0600); err != nil {
				t.Fatal(err)
			}
		}
		script := strings.ReplaceAll(step.Run, "${{ runner.temp }}", input)
		cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", script)
		cmd.Dir = work
		output, err := cmd.CombinedOutput()
		if bad != "" {
			if err == nil {
				t.Fatalf("artifact preparation accepted missing/empty %s", bad)
			}
			if !strings.Contains(string(output), "Missing prebuilt binary") {
				t.Fatalf("wrong failure, err=%v: %s", err, output)
			}
			return
		}
		if err != nil {
			t.Fatalf("complete artifacts rejected: %v: %s", err, output)
		}
		for _, name := range files {
			b, err := os.ReadFile(filepath.Join(work, "dist", name))
			if err != nil || string(b) != "presence-check fixture\n" {
				t.Fatalf("binary not preserved at expected prebuilt path %s: %v", name, err)
			}
		}
	}
	t.Run("complete", func(t *testing.T) { runCase(t, "", false) })
	for _, name := range files {
		t.Run("missing_"+name, func(t *testing.T) { runCase(t, name, false) })
		t.Run("empty_"+name, func(t *testing.T) { runCase(t, name, true) })
	}
}
