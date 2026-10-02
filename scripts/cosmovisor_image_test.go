package scripts

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Container integration tests for the cosmovisor shipped in the release image.
// They run only when COSMOVISOR_TEST_IMAGE names a locally loaded image, e.g.
//
//	COSMOVISOR_TEST_IMAGE=xion:linux-amd64 COSMOVISOR_TEST_PLATFORM=linux/amd64 \
//	    go test -count=1 -v ./scripts -run '^TestCosmovisor'
//
// See scripts/cosmovisor.md.

const cosmovisorPatchPath = "cosmovisor-patches/0001-decode-db-backend-output.patch"

var containerSeq atomic.Int64

type cosmovisorImage struct {
	image    string
	platform string
	fixture  string
}

func requireCosmovisorImage(t *testing.T) cosmovisorImage {
	t.Helper()
	image := os.Getenv("COSMOVISOR_TEST_IMAGE")
	if image == "" {
		t.Skip("COSMOVISOR_TEST_IMAGE is not set")
	}
	platform := os.Getenv("COSMOVISOR_TEST_PLATFORM")
	if platform == "" {
		platform = "linux/" + runtime.GOARCH
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("COSMOVISOR_TEST_IMAGE is set but docker is unavailable: %v", err)
	}
	fixture, err := filepath.Abs(filepath.Join("testdata", "cosmovisor-daemon.sh"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("%s is not executable", fixture)
	}
	return cosmovisorImage{image: image, platform: platform, fixture: fixture}
}

// run executes script with sh -eu in a fresh, network-less container from the
// image, as the image's default user. It returns combined output and exit code.
func (c cosmovisorImage) run(t *testing.T, timeout time.Duration, script string, env ...string) (string, int) {
	t.Helper()
	name := fmt.Sprintf("cosmovisor-test-%d-%d", os.Getpid(), containerSeq.Add(1))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
	})
	args := []string{
		"run", "--rm", "--name", name, "--network", "none",
		"--platform", c.platform,
		"-v", c.fixture + ":/fixture/daemon:ro",
		"--entrypoint", "sh",
	}
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, c.image, "-euc", shellPrelude+script)

	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("container did not finish within %s:\n%s", timeout, out)
	}
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("docker run: %v\n%s", err, out)
		}
		code = exitErr.ExitCode()
	}
	return string(out), code
}

// shellPrelude gives every container script bounded waits and a 1/100 s clock.
const shellPrelude = `
F=/tmp/home/fixture
wait_for() {
	i=0
	while [ ! -e "$1" ]; do
		i=$((i + 1))
		if [ "$i" -ge 300 ]; then echo "timeout waiting for $1"; cat /tmp/cv.log 2>/dev/null || true; exit 70; fi
		sleep 0.1
	done
}
wait_exit() {
	i=0
	while kill -0 "$1" 2>/dev/null; do
		i=$((i + 1))
		if [ "$i" -ge 300 ]; then echo "timeout waiting for pid $1"; cat /tmp/cv.log 2>/dev/null || true; exit 70; fi
		sleep 0.1
	done
	set +e; wait "$1"; rc=$?; set -e
}
now_cs() { awk '{ printf "%d", $1 * 100 }' /proc/uptime; }
yn() { if [ -e "$1" ]; then echo yes; else echo no; fi; }
`

// keyValues collects the key=value lines a container script prints.
func keyValues(out string) map[string]string {
	kv := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), "="); ok && !strings.ContainsAny(k, " \t") {
			kv[k] = v
		}
	}
	return kv
}

func expectKV(t *testing.T, out string, want map[string]string) map[string]string {
	t.Helper()
	got := keyValues(out)
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if t.Failed() {
		t.Logf("container output:\n%s", out)
	}
	return got
}

// dockerfileCosmovisorArgs returns the cosmovisor ARG defaults in the Dockerfile.
func dockerfileCosmovisorArgs(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^ARG (COSMOVISOR_[A-Z0-9_]+)="([^"]*)"$`)
	args := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		args[m[1]] = m[2]
	}
	for _, k := range []string{"COSMOVISOR_COMMIT", "COSMOVISOR_SOURCE_SHA256", "COSMOVISOR_PATCH_SHA256", "COSMOVISOR_BUILD_IMAGE"} {
		if args[k] == "" {
			t.Fatalf("Dockerfile has no default for %s", k)
		}
	}
	return args
}

// TestCosmovisorPatchChecksum keeps the Dockerfile's pinned patch hash in step
// with the patch file, so an edited patch cannot ship without review.
func TestCosmovisorPatchChecksum(t *testing.T) {
	b, err := os.ReadFile(cosmovisorPatchPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	args := dockerfileCosmovisorArgs(t)
	if got := hex.EncodeToString(sum[:]); got != args["COSMOVISOR_PATCH_SHA256"] {
		t.Fatalf("sha256(%s) = %s, Dockerfile pins %s", cosmovisorPatchPath, got, args["COSMOVISOR_PATCH_SHA256"])
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(args["COSMOVISOR_COMMIT"]) {
		t.Errorf("COSMOVISOR_COMMIT %q is not a full commit SHA", args["COSMOVISOR_COMMIT"])
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(args["COSMOVISOR_SOURCE_SHA256"]) {
		t.Errorf("COSMOVISOR_SOURCE_SHA256 %q is not a SHA256", args["COSMOVISOR_SOURCE_SHA256"])
	}
	if !regexp.MustCompile(`@sha256:[0-9a-f]{64}$`).MatchString(args["COSMOVISOR_BUILD_IMAGE"]) {
		t.Errorf("COSMOVISOR_BUILD_IMAGE %q is not pinned by digest", args["COSMOVISOR_BUILD_IMAGE"])
	}
	// The patch may touch only the three reviewed upstream files.
	allowed := map[string]bool{
		"tools/cosmovisor/scanner.go":        true,
		"tools/cosmovisor/dbbackend.go":      true,
		"tools/cosmovisor/dbbackend_test.go": true,
	}
	for _, m := range regexp.MustCompile(`(?m)^diff --git a/(\S+) b/(\S+)$`).FindAllStringSubmatch(string(b), -1) {
		if !allowed[m[1]] || m[1] != m[2] {
			t.Errorf("patch touches %s", m[1])
		}
	}
}

func TestCosmovisorOperatorImage(t *testing.T) {
	img := requireCosmovisorImage(t)
	arch := strings.TrimPrefix(img.platform, "linux/")

	t.Run("image contract and provenance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		out, err := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", "{{json .Config}}", img.image).Output()
		if err != nil {
			t.Fatalf("docker image inspect: %v", err)
		}
		var cfg struct {
			User       string
			WorkingDir string
			Cmd        []string
			Labels     map[string]string
		}
		if err := json.Unmarshal(out, &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.User != "xiond:xiond" || cfg.WorkingDir != "/home/xiond/.xiond" || strings.Join(cfg.Cmd, " ") != "/usr/bin/xiond" {
			t.Errorf("image runtime contract changed: user %q, workdir %q, cmd %q", cfg.User, cfg.WorkingDir, cfg.Cmd)
		}
		args := dockerfileCosmovisorArgs(t)
		for label, arg := range map[string]string{
			"io.burnt.cosmovisor.revision":      "COSMOVISOR_COMMIT",
			"io.burnt.cosmovisor.source-sha256": "COSMOVISOR_SOURCE_SHA256",
			"io.burnt.cosmovisor.patch-sha256":  "COSMOVISOR_PATCH_SHA256",
		} {
			if cfg.Labels[label] != args[arg] {
				t.Errorf("label %s = %q, want %s %q", label, cfg.Labels[label], arg, args[arg])
			}
		}

		expectKV(t, imgRun(t, img, `
echo uid=$(id -u)
echo gid=$(id -g)
echo xiond=$(test -x /usr/bin/xiond && echo yes || echo no)
echo cosmovisor=$(test -x /usr/bin/cosmovisor && echo yes || echo no)
`), map[string]string{"uid": "1000", "gid": "1000", "xiond": "yes", "cosmovisor": "yes"})

		// The binary carries real Go build information rather than
		// pretending to be an upstream release tarball.
		dir := t.TempDir()
		name := fmt.Sprintf("cosmovisor-test-cp-%d-%d", os.Getpid(), containerSeq.Add(1))
		if out, err := exec.CommandContext(ctx, "docker", "create", "--name", name, "--platform", img.platform, img.image).CombinedOutput(); err != nil {
			t.Fatalf("docker create: %v\n%s", err, out)
		}
		t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", name).Run() })
		bin := filepath.Join(dir, "cosmovisor")
		if out, err := exec.CommandContext(ctx, "docker", "cp", name+":/usr/bin/cosmovisor", bin).CombinedOutput(); err != nil {
			t.Fatalf("docker cp: %v\n%s", err, out)
		}
		info, err := exec.CommandContext(ctx, "go", "version", "-m", bin).Output()
		if err != nil {
			t.Fatalf("go version -m: %v", err)
		}
		for _, want := range []string{
			"\tpath\tcosmossdk.io/tools/cosmovisor/cmd/cosmovisor\n",
			"\tmod\tcosmossdk.io/tools/cosmovisor\t(devel)",
			"\tbuild\tCGO_ENABLED=0\n",
			"\tbuild\tGOARCH=" + arch + "\n",
		} {
			if !strings.Contains(string(info), want) {
				t.Errorf("build info lacks %q:\n%s", want, info)
			}
		}
	})

	t.Run("required environment", func(t *testing.T) {
		expectKV(t, imgRun(t, img, `
set +e
env -u DAEMON_NAME DAEMON_HOME=/tmp/home cosmovisor run version >/dev/null 2>&1; echo no_name=$?
DAEMON_NAME=xiond DAEMON_HOME=relative/home cosmovisor run version >/dev/null 2>&1; echo relative_home=$?
`), map[string]string{"no_name": "1", "relative_home": "1"})
	})

	t.Run("init layout", func(t *testing.T) {
		expectKV(t, imgRun(t, img, `
export DAEMON_NAME=xiond DAEMON_HOME=/tmp/home
cosmovisor init /usr/bin/xiond >/dev/null
echo genesis_bin=$(cmp -s /usr/bin/xiond $DAEMON_HOME/cosmovisor/genesis/bin/xiond && echo same || echo differs)
echo current_link=$(readlink $DAEMON_HOME/cosmovisor/current)
echo current_resolved=$(readlink -f $DAEMON_HOME/cosmovisor/current)
echo config=$(yn $DAEMON_HOME/cosmovisor/config.toml)
`), map[string]string{
			"genesis_bin":      "same",
			"current_link":     "genesis",
			"current_resolved": "/tmp/home/cosmovisor/genesis",
			"config":           "yes",
		})
	})

	t.Run("config defaults, overrides and file precedence", func(t *testing.T) {
		expectKV(t, imgRun(t, img, `
export DAEMON_NAME=xiond DAEMON_HOME=/tmp/home
cosmovisor init /usr/bin/xiond >/dev/null
show() { prefix=$1; shift; cosmovisor config "$@" 2>&1 | sed -n "s/^  \([A-Z_]*\): \(.*\)$/$prefix.\1=\2/p"; }
show default
DAEMON_ALLOW_DOWNLOAD_BINARIES=true DAEMON_RESTART_AFTER_UPGRADE=false DAEMON_POLL_INTERVAL=1s \
	DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true DAEMON_SHUTDOWN_GRACE=7s show env
cfg=$DAEMON_HOME/cosmovisor/config.toml
sed -i 's/^daemon_poll_interval = .*/daemon_poll_interval = 2000000000/; s/^daemon_restart_after_upgrade = .*/daemon_restart_after_upgrade = false/' $cfg
show file --cosmovisor-config $cfg
DAEMON_POLL_INTERVAL=1s show envfile --cosmovisor-config $cfg
`), map[string]string{
			"default.DAEMON_HOME":                        "/tmp/home",
			"default.DAEMON_NAME":                        "xiond",
			"default.DAEMON_ALLOW_DOWNLOAD_BINARIES":     "false",
			"default.DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM": "false",
			"default.DAEMON_RESTART_AFTER_UPGRADE":       "true",
			"default.DAEMON_POLL_INTERVAL":               "300ms",
			"default.DAEMON_SHUTDOWN_GRACE":              "0s",
			"default.UNSAFE_SKIP_BACKUP":                 "false",
			"env.DAEMON_ALLOW_DOWNLOAD_BINARIES":         "true",
			"env.DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM":     "true",
			"env.DAEMON_RESTART_AFTER_UPGRADE":           "false",
			"env.DAEMON_POLL_INTERVAL":                   "1s",
			"env.DAEMON_SHUTDOWN_GRACE":                  "7s",
			"file.DAEMON_POLL_INTERVAL":                  "2s",
			"file.DAEMON_RESTART_AFTER_UPGRADE":          "false",
			// An environment variable still overrides the file.
			"envfile.DAEMON_POLL_INTERVAL":         "1s",
			"envfile.DAEMON_RESTART_AFTER_UPGRADE": "false",
		})
	})

	t.Run("run reaches real xiond", func(t *testing.T) {
		got := keyValues(imgRun(t, img, `
export DAEMON_NAME=xiond DAEMON_HOME=/tmp/home COSMOVISOR_DISABLE_LOGS=true
cosmovisor init /usr/bin/xiond >/dev/null
mkdir -p $DAEMON_HOME/data
echo direct=$(xiond version 2>&1)
echo via=$(cosmovisor run version 2>&1)
`))
		if got["direct"] == "" || got["via"] != got["direct"] {
			t.Errorf("cosmovisor run version = %q, xiond version = %q", got["via"], got["direct"])
		}
	})

	t.Run("add-upgrade stages binary and plan", func(t *testing.T) {
		got := expectKV(t, imgRun(t, img, `
export DAEMON_NAME=xiond DAEMON_HOME=/tmp/home
cosmovisor init /usr/bin/xiond >/dev/null
mkdir -p $DAEMON_HOME/data
cosmovisor add-upgrade smoke /usr/bin/xiond --upgrade-height 10 >/dev/null
echo upgrade_bin=$(cmp -s /usr/bin/xiond $DAEMON_HOME/cosmovisor/upgrades/smoke/bin/xiond && echo same || echo differs)
echo plan=$(cat $DAEMON_HOME/data/upgrade-info.json)
`), map[string]string{"upgrade_bin": "same"})
		var plan struct {
			Name   string `json:"name"`
			Height int64  `json:"height"`
		}
		if err := json.Unmarshal([]byte(got["plan"]), &plan); err != nil || plan.Name != "smoke" || plan.Height != 10 {
			t.Errorf("upgrade-info.json = %q (%v), want name smoke height 10", got["plan"], err)
		}
	})

	// launch starts the fixture daemon under cosmovisor and writes an upgrade
	// plan once it is running. upgrade_at is the uptime (1/100 s) at that point.
	const launch = `
export DAEMON_NAME=xiond DAEMON_HOME=/tmp/home DAEMON_POLL_INTERVAL=100ms UNSAFE_SKIP_BACKUP=true COSMOVISOR_COLOR_LOGS=false
cosmovisor init /fixture/daemon >/dev/null
mkdir -p $DAEMON_HOME/data
`
	const startAndUpgrade = `
cosmovisor run start --home $DAEMON_HOME --flag value >/tmp/cv.log 2>&1 &
pid=$!
wait_for $F/start-genesis.args
upgrade_at=$(now_cs)
printf '%s' "$PLAN" > $DAEMON_HOME/data/upgrade-info.json
`

	t.Run("upgrade switches binary and restarts with the same arguments", func(t *testing.T) {
		expectKV(t, imgRun(t, img, launch+`
cosmovisor add-upgrade smoke /fixture/daemon >/dev/null
PLAN='{"name":"smoke","height":10}'
`+startAndUpgrade+`
wait_for $F/start-smoke.args
echo current=$(readlink $DAEMON_HOME/cosmovisor/current)
echo pre_upgrade=$(cat $F/pre-upgrade)
echo genesis_args=$(cat $F/start-genesis.args | tr '\n' ' ')
echo smoke_args=$(cat $F/start-smoke.args | tr '\n' ' ')
echo genesis_term=$(yn $F/term-genesis)
echo genesis_int=$(yn $F/int-genesis)
`), map[string]string{
			"current":      "upgrades/smoke",
			"pre_upgrade":  "smoke",
			"genesis_args": "start --home /tmp/home --flag value",
			"smoke_args":   "start --home /tmp/home --flag value",
			// With no shutdown grace the old daemon is killed outright.
			"genesis_term": "no",
			"genesis_int":  "no",
		})
	})

	t.Run("upgrade without restart exits after switching", func(t *testing.T) {
		expectKV(t, imgRun(t, img, launch+`
export DAEMON_RESTART_AFTER_UPGRADE=false
cosmovisor add-upgrade smoke /fixture/daemon >/dev/null
PLAN='{"name":"smoke","height":10}'
`+startAndUpgrade+`
wait_exit $pid
echo rc=$rc
echo current=$(readlink $DAEMON_HOME/cosmovisor/current)
echo pre_upgrade=$(cat $F/pre-upgrade)
echo smoke_started=$(yn $F/start-smoke.args)
`), map[string]string{"rc": "0", "current": "upgrades/smoke", "pre_upgrade": "smoke", "smoke_started": "no"})
	})

	t.Run("shutdown grace sends SIGTERM and is bounded", func(t *testing.T) {
		got := expectKV(t, imgRun(t, img, launch+`
export DAEMON_RESTART_AFTER_UPGRADE=false DAEMON_SHUTDOWN_GRACE=2s FIXTURE_IGNORE_TERM=1
cosmovisor add-upgrade smoke /fixture/daemon >/dev/null
PLAN='{"name":"smoke","height":10}'
`+startAndUpgrade+`
wait_exit $pid
echo elapsed_cs=$(( $(now_cs) - upgrade_at ))
echo rc=$rc
echo term=$(yn $F/term-genesis)
echo int=$(yn $F/int-genesis)
echo current=$(readlink $DAEMON_HOME/cosmovisor/current)
`), map[string]string{"rc": "0", "term": "yes", "int": "no", "current": "upgrades/smoke"})
		elapsed, err := strconv.Atoi(got["elapsed_cs"])
		if err != nil || elapsed < 200 || elapsed > 1500 {
			t.Errorf("upgrade with a 2s grace for a daemon ignoring SIGTERM took %s/100 s, want between 2 s and 15 s", got["elapsed_cs"])
		}
	})

	t.Run("missing upgrade binary is not downloaded", func(t *testing.T) {
		out := imgRun(t, img, launch+`
PLAN='{"name":"missing","height":10,"info":"{\"binaries\":{\"linux/`+arch+`\":\"http://127.0.0.1:9/xiond\"}}"}'
`+startAndUpgrade+`
wait_exit $pid
echo rc=$rc
echo current=$(readlink $DAEMON_HOME/cosmovisor/current)
echo upgrade_dir=$(yn $DAEMON_HOME/cosmovisor/upgrades/missing)
echo disabled=$(grep -c 'downloading disabled' /tmp/cv.log || true)
`)
		got := expectKV(t, out, map[string]string{"current": "genesis", "upgrade_dir": "no"})
		if got["rc"] == "0" || got["disabled"] == "0" {
			t.Errorf("rc = %s, 'downloading disabled' lines = %s; want a non-zero exit refusing the download\n%s", got["rc"], got["disabled"], out)
		}
	})
}

func imgRun(t *testing.T, img cosmovisorImage, script string) string {
	t.Helper()
	out, code := img.run(t, 2*time.Minute, script)
	if code != 0 {
		t.Fatalf("container script exited %d:\n%s", code, out)
	}
	return out
}

// realStateScript produces blocks with a one-validator xiond node, stops it,
// and runs cosmovisor's after-exit upgrade check against that real CometBFT
// state: the daemon command fails (RPC is down), so cosmovisor has to open the
// blockstore itself using the db_backend xiond reports.
const realStateScript = `
H=/tmp/node
xiond init rs --chain-id rs-1 --home $H >/dev/null 2>&1
xiond keys add val --keyring-backend test --home $H >/dev/null 2>&1
xiond genesis add-genesis-account val 1000000000000stake --keyring-backend test --home $H >/dev/null
xiond genesis gentx val 100000000stake --chain-id rs-1 --keyring-backend test --home $H >/dev/null 2>&1
xiond genesis collect-gentxs --home $H >/dev/null 2>&1
sed -i 's/^timeout_commit = .*/timeout_commit = "200ms"/' $H/config/config.toml

committed() { grep 'committed state' "$1" | grep -oE ' height=[0-9]+' | grep -oE '[0-9]+$' | tail -1 || true; }
wait_height() {
	i=0
	while :; do
		h=$(committed "$1")
		if [ "${h:-0}" -ge "$2" ]; then return 0; fi
		if ! kill -0 "$3" 2>/dev/null; then echo "error=process exited before height $2"; tail -30 "$1"; exit 70; fi
		i=$((i + 1))
		if [ "$i" -ge 1800 ]; then echo "error=height $2 not reached"; tail -30 "$1"; exit 70; fi
		sleep 0.1
	done
}

xiond start --home $H --minimum-gas-prices 0stake --log_no_color >/tmp/start.log 2>&1 &
pid=$!
wait_height /tmp/start.log 5 $pid
kill -TERM $pid
wait_exit $pid
last=$(committed /tmp/start.log)
echo last=$last
echo db_backend_output=$(xiond config get config db_backend --home $H | od -An -c | tr -s ' ')
if xiond status --home $H >/dev/null 2>&1; then echo rpc=up; else echo rpc=down; fi

for c in reached future; do
	C=/tmp/$c
	cp -a $H $C
	if [ $c = reached ]; then uh=$last; else uh=$((last + 1000)); fi
	export DAEMON_NAME=xiond DAEMON_HOME=$C DAEMON_RESTART_AFTER_UPGRADE=false UNSAFE_SKIP_BACKUP=true COSMOVISOR_COLOR_LOGS=false
	cosmovisor init /usr/bin/xiond >/dev/null 2>&1
	cosmovisor add-upgrade smoke /usr/bin/xiond >/dev/null 2>&1
	printf '{"name":"smoke","height":%d}' $uh > $C/data/upgrade-info.json
	set +e; cosmovisor run status --home $C >/tmp/$c.log 2>&1; rc=$?; set -e
	echo $c.rc=$rc
	echo $c.current=$(readlink $C/cosmovisor/current)
	echo $c.parse_errors=$(grep -c 'parse db_backend' /tmp/$c.log || true)
done

C=/tmp/reached
export DAEMON_HOME=$C
cosmovisor run start --home $C --minimum-gas-prices 0stake --log_no_color >/tmp/restart.log 2>&1 &
pid=$!
wait_height /tmp/restart.log $((last + 3)) $pid
kill -TERM $pid
wait_exit $pid
echo restart.height=$(committed /tmp/restart.log)
echo restart.path=$(grep 'running app' /tmp/restart.log | grep -oE 'path=[^ ]+' | tail -1)
`

func TestCosmovisorRealStateFallback(t *testing.T) {
	img := requireCosmovisorImage(t)
	out, code := img.run(t, 10*time.Minute, realStateScript)
	if code != 0 {
		t.Fatalf("real-state script exited %d:\n%s", code, out)
	}
	got := expectKV(t, out, map[string]string{
		// Precondition: the daemon is down, so cosmovisor must read the DB.
		"rpc": "down",
		// Reached upgrade height: the blockstore is read and the binary switched.
		"reached.rc":           "0",
		"reached.current":      "upgrades/smoke",
		"reached.parse_errors": "0",
		// Future upgrade height: no switch, the daemon's own failure is returned.
		"future.current":      "genesis",
		"future.parse_errors": "0",
	})
	t.Logf("last committed height %s, db_backend output %q", got["last"], got["db_backend_output"])
	if got["future.rc"] == "0" {
		t.Errorf("future.rc = 0, want the failing daemon's non-zero exit")
	}
	last, err := strconv.Atoi(got["last"])
	if err != nil || last < 5 {
		t.Fatalf("last committed height %q, want >= 5", got["last"])
	}
	if h, err := strconv.Atoi(got["restart.height"]); err != nil || h < last+3 {
		t.Errorf("selected binary reached height %q after restart, want >= %d", got["restart.height"], last+3)
	}
	if !strings.Contains(got["restart.path"], "/cosmovisor/upgrades/smoke/bin/xiond") && !strings.Contains(got["restart.path"], "/cosmovisor/current/bin/xiond") {
		t.Errorf("restarted node ran %q, want the upgraded binary", got["restart.path"])
	}
}
