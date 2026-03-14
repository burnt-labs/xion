//go:build linux && amd64 && !barretenberg_stub

package barretenberg

// #cgo LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -lbarretenberg -lc++ -lm -lpthread
import "C"
