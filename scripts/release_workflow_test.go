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
	Name string         `json:"name"`
	Uses string         `json:"uses"`
	Run  string         `json:"run"`
	With map[string]any `json:"with"`
	Env  map[string]any `json:"env"`
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
	// The binary jobs build to dist/<os>_<arch>/xiond_<os>_<arch>_<variant>/bin/
	// and upload dist/**/xiond-<os>-<arch>, so the artifact root is dist/ and
	// merge-multiple extracts each binary under runner.temp/<os>_<arch>/.
	downloaded := func(name string) string {
		parts := strings.SplitN(strings.TrimPrefix(name, "xiond_"), "_", 3)
		return filepath.Join(parts[0]+"_"+parts[1], name)
	}
	runCase := func(t *testing.T, bad string, empty bool) {
		t.Helper()
		work := t.TempDir()
		input := filepath.Join(work, "input")
		for _, name := range files {
			if name == bad && !empty {
				continue
			}
			path := filepath.Join(input, downloaded(name))
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

// The Fury job builds the deb/rpm/apk packages. An rc tag can sit on the
// release commit (v31.0.1-rc1 and v31.0.1 on 3ef408a), and GoReleaser picks
// the highest-sorting tag on HEAD by itself, which is the rc, so the job must
// check out and name the resolved tag explicitly.
func TestPublishFuryPackagesResolvedTag(t *testing.T) {
	jobs := readReleaseWorkflow(t, "publish-release.yaml")
	fury, ok := jobs["publish-fury"]
	if !ok {
		t.Fatal("missing publish-fury job")
	}
	if !slices.Contains(releaseNeeds(t, fury), "resolve-tag") {
		t.Fatal("publish-fury does not wait for the resolved tag")
	}
	const tag = "${{ needs.resolve-tag.outputs.tag }}"
	checkout, _ := releaseStepByName(t, fury, "Checkout")
	if ref, _ := checkout.With["ref"].(string); ref != "refs/tags/"+tag {
		t.Fatalf("publish-fury checks out %q, not the resolved release tag", ref)
	}
	if depth := checkout.With["fetch-depth"]; depth != float64(0) {
		t.Fatalf("publish-fury needs full history for GoReleaser, got fetch-depth %v", depth)
	}
	goreleaser, _ := releaseStepByName(t, fury, "Run GoReleaser (packages + homebrew)")
	if cur, _ := goreleaser.Env["GORELEASER_CURRENT_TAG"].(string); cur != tag {
		t.Fatalf("GoReleaser current tag is %q, not the resolved release tag", cur)
	}
}

func TestPublishResolveTagGuard(t *testing.T) {
	jobs := readReleaseWorkflow(t, "publish-release.yaml")
	step, _ := releaseStepByName(t, jobs["resolve-tag"], "Determine release tag")
	cases := []struct {
		input, event, ref string
		want, rc          string
	}{
		{"", "v31.0.1", "v31.0.1", "v31.0.1", "false"},
		{"", "v31.0.2-rc2", "v31.0.2-rc2", "v31.0.2-rc2", "true"},
		{"v31.0.1", "", "release/v32", "v31.0.1", "false"},
		{"", "", "release/v32", "", ""},
		{"v31.0", "", "release/v32", "", ""},
		{"v31.0.1;exit 0", "", "release/v32", "", ""},
	}
	for _, c := range cases {
		t.Run(c.input+"|"+c.event+"|"+c.ref, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "output")
			cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", step.Run)
			cmd.Env = append(os.Environ(),
				"RELEASE_TAG_INPUT="+c.input,
				"RELEASE_TAG_EVENT="+c.event,
				"REF_NAME="+c.ref,
				"GITHUB_OUTPUT="+out,
			)
			output, err := cmd.CombinedOutput()
			if c.want == "" {
				if err == nil {
					t.Fatalf("accepted a non-release tag: %s", output)
				}
				return
			}
			if err != nil {
				t.Fatalf("rejected %s: %v: %s", c.want, err, output)
			}
			b, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(b); got != "tag="+c.want+"\nis_rc="+c.rc+"\n" {
				t.Fatalf("outputs %q", got)
			}
		})
	}
}

// Gemfury answers 409 for a version it already has, so the upload step must
// refuse anything but the resolved tag's six packages before uploading.
func TestPublishFuryUploadsOnlyResolvedVersion(t *testing.T) {
	jobs := readReleaseWorkflow(t, "publish-release.yaml")
	step, _ := releaseStepByName(t, jobs["publish-fury"], "Upload packages to Gemfury")
	if tag, _ := step.Env["RELEASE_TAG"].(string); tag != "${{ needs.resolve-tag.outputs.tag }}" {
		t.Fatalf("upload step checks against %q, not the resolved release tag", tag)
	}
	pkgs := func(version string) []string {
		var out []string
		for _, arch := range []string{"amd64", "arm64"} {
			for _, f := range []string{"deb", "rpm", "apk"} {
				out = append(out, "xiond_"+version+"_linux_"+arch+"."+f)
			}
		}
		return out
	}
	cases := []struct {
		name, tag string
		files     []string
		empty     string
		wantErr   string
	}{
		{"rc built for stable tag", "v31.0.1", pkgs("31.0.1-rc1"), "", "Missing package for v31.0.1"},
		{"extra rc package", "v31.0.1", append(pkgs("31.0.1"), "xiond_31.0.1-rc1_linux_amd64.deb"), "", "Package not built for v31.0.1"},
		{"one missing", "v31.0.1", pkgs("31.0.1")[1:], "", "Missing package for v31.0.1"},
		{"one empty", "v31.0.1", pkgs("31.0.1"), "xiond_31.0.1_linux_arm64.apk", "Missing package for v31.0.1"},
		{"stable", "v31.0.1", pkgs("31.0.1"), "", ""},
		{"rc", "v31.0.2-rc2", pkgs("31.0.2-rc2"), "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			work := t.TempDir()
			if err := os.MkdirAll(filepath.Join(work, "release"), 0755); err != nil {
				t.Fatal(err)
			}
			for _, f := range c.files {
				contents := []byte("package fixture\n")
				if f == c.empty {
					contents = nil
				}
				if err := os.WriteFile(filepath.Join(work, "release", f), contents, 0600); err != nil {
					t.Fatal(err)
				}
			}
			// A stub curl records each upload and answers 201, so the test
			// never reaches Gemfury.
			bin := filepath.Join(work, "bin")
			if err := os.MkdirAll(bin, 0755); err != nil {
				t.Fatal(err)
			}
			log := filepath.Join(work, "uploads")
			stub := "#!/usr/bin/env bash\nfor a in \"$@\"; do case \"$a\" in package=@*) echo \"${a#package=@}\" >> " + log + ";; esac; done\nprintf 'ok\\n201'\n"
			if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(stub), 0755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-e", "-c", step.Run)
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RELEASE_TAG="+c.tag, "FURY_TOKEN=test")
			output, err := cmd.CombinedOutput()
			uploads, _ := os.ReadFile(log)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(string(output), c.wantErr) {
					t.Fatalf("want failure %q, err=%v: %s", c.wantErr, err, output)
				}
				if len(uploads) != 0 {
					t.Fatalf("uploaded before refusing: %s", uploads)
				}
				return
			}
			if err != nil {
				t.Fatalf("rejected complete packages: %v: %s", err, output)
			}
			var want []string
			for _, f := range pkgs(strings.TrimPrefix(c.tag, "v")) {
				want = append(want, "release/"+f)
			}
			got := strings.Fields(string(uploads))
			slices.Sort(got)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("uploaded %v, want %v", got, want)
			}
		})
	}
}

// A packages-only dispatch republishes to Gemfury without re-firing the
// downstream release automation or the Homebrew tap; a published release
// still runs all of it.
func TestPublishPackagesOnlySkipsDownstream(t *testing.T) {
	jobs := readReleaseWorkflow(t, "publish-release.yaml")
	for _, name := range []string{"trigger-types", "update-chain-registry", "upgrade-network"} {
		j, ok := jobs[name]
		if !ok {
			t.Fatalf("missing job %s", name)
		}
		if j.If != "github.repository == 'burnt-labs/xion' && !inputs.packages_only" {
			t.Errorf("%s runs on a packages-only dispatch: if=%q", name, j.If)
		}
	}
	goreleaser, _ := releaseStepByName(t, jobs["publish-fury"], "Run GoReleaser (packages + homebrew)")
	args, _ := goreleaser.With["args"].(string)
	if !strings.Contains(args, "inputs.packages_only && 'announce,validate,homebrew' || 'announce,validate'") {
		t.Fatalf("GoReleaser does not skip Homebrew on a packages-only dispatch: %q", args)
	}
}
