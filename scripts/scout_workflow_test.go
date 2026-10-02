package scripts

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

type scoutStep struct {
	Name            string            `json:"name"`
	Uses            string            `json:"uses"`
	If              string            `json:"if"`
	ContinueOnError any               `json:"continue-on-error"`
	With            map[string]any    `json:"with"`
	Env             map[string]string `json:"env"`
	Run             string            `json:"run"`
}

type scoutJob struct {
	If              string `json:"if"`
	ContinueOnError any    `json:"continue-on-error"`
	Strategy        struct {
		Matrix map[string]any `json:"matrix"`
	} `json:"strategy"`
	Steps []scoutStep `json:"steps"`
}

func readScoutJob(t *testing.T) scoutJob {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", ".github", "workflows", "docker-scout.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]scoutJob `json:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	job, ok := workflow.Jobs["docker-scout"]
	if !ok {
		t.Fatal("missing Docker Scout job")
	}
	return job
}

func canBypass(cond string, continueOnError any) bool {
	return cond != "" || (continueOnError != nil && continueOnError != false)
}

// scoutScanStep returns the single step that runs the Scout action.
func scoutScanStep(t *testing.T, job scoutJob) scoutStep {
	t.Helper()
	var found []scoutStep
	for _, step := range job.Steps {
		if strings.HasPrefix(step.Uses, "docker/scout-action@") {
			found = append(found, step)
		}
	}
	if len(found) != 1 {
		t.Fatalf("got %d docker/scout-action steps, want 1", len(found))
	}
	return found[0]
}

// TestScoutFailureContract fails if the release image scan stops failing the
// workflow on a fixable HIGH or CRITICAL vulnerability.
func TestScoutFailureContract(t *testing.T) {
	job := readScoutJob(t)
	if canBypass(job.If, job.ContinueOnError) {
		t.Fatal("Scout job can bypass the failure gate")
	}
	step := scoutScanStep(t, job)
	if step.Name != "Run Docker Scout" {
		t.Errorf("Scout step is named %q", step.Name)
	}
	if canBypass(step.If, step.ContinueOnError) {
		t.Fatal("Scout step can bypass the failure gate")
	}
	for key, want := range map[string]any{
		"command":         "cves",
		"only-fixed":      true,
		"exit-code":       true,
		"only-severities": "critical,high",
		"platform":        "${{ matrix.os }}/${{ matrix.arch }}",
		"image":           "${{ env.SCOUT_IMAGE }}",
	} {
		if got := step.With[key]; got != want {
			t.Errorf("Scout %s = %v, want %v", key, got, want)
		}
	}
	for _, key := range []string{
		"ignore-base", "only-package-types", "only-unfixed", "ignore-suppressed",
		"only-vex-affected", "vex-location", "vex-author", "exit-on", "only-cisa-kev",
	} {
		if _, exists := step.With[key]; exists {
			t.Errorf("unexpected scan restriction %s", key)
		}
	}
}

// TestScoutArchitectureContract fails if either released architecture stops
// being scanned.
func TestScoutArchitectureContract(t *testing.T) {
	job := readScoutJob(t)
	matrix := job.Strategy.Matrix
	if len(matrix) != 2 {
		t.Errorf("matrix dimensions = %v, want os and arch only", matrix)
	}
	asStrings := func(v any) []string {
		var out []string
		list, _ := v.([]any)
		for _, x := range list {
			s, _ := x.(string)
			out = append(out, s)
		}
		slices.Sort(out)
		return out
	}
	if got := asStrings(matrix["os"]); !slices.Equal(got, []string{"linux"}) {
		t.Errorf("matrix os = %v, want [linux]", got)
	}
	if got := asStrings(matrix["arch"]); !slices.Equal(got, []string{"amd64", "arm64"}) {
		t.Errorf("matrix arch = %v, want [amd64 arm64]", got)
	}
}

// TestScoutRunsOnPullRequests fails if Build and Test stops scanning the
// images it builds, which would leave the first Scout run to the release.
func TestScoutRunsOnPullRequests(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", ".github", "workflows", "build-test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Needs           any    `json:"needs"`
			If              string `json:"if"`
			Uses            string `json:"uses"`
			Secrets         any    `json:"secrets"`
			ContinueOnError any    `json:"continue-on-error"`
		} `json:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	var names []string
	for name, job := range workflow.Jobs {
		if job.Uses != "./.github/workflows/docker-scout.yaml" {
			continue
		}
		names = append(names, name)
		needs := job.Needs
		if s, ok := needs.(string); ok {
			needs = []any{s}
		}
		if list, _ := needs.([]any); !slices.Contains(list, any("build-docker")) {
			t.Errorf("%s needs = %v, want build-docker", name, job.Needs)
		}
		if job.Secrets != "inherit" {
			t.Errorf("%s secrets = %v, want inherit (Docker Hub login)", name, job.Secrets)
		}
		if job.ContinueOnError != nil && job.ContinueOnError != false {
			t.Errorf("%s can continue on error", name)
		}
		// Only the documented fork/Dependabot skip is allowed.
		for _, want := range []string{
			"github.event_name != 'pull_request'",
			"github.event.pull_request.head.repo.full_name == github.repository",
			"github.actor != 'dependabot[bot]'",
		} {
			if !strings.Contains(job.If, want) {
				t.Errorf("%s if = %q, missing %q", name, job.If, want)
			}
		}
		if strings.Contains(job.If, "always()") || strings.Contains(job.If, "failure()") {
			t.Errorf("%s if = %q runs regardless of the build", name, job.If)
		}
	}
	if len(names) != 1 {
		t.Fatalf("got %d docker-scout.yaml callers in build-test.yaml, want 1", len(names))
	}
}
