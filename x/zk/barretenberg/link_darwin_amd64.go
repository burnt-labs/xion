//go:build darwin && amd64 && !barretenberg_stub

package barretenberg

// #cgo LDFLAGS: -L${SRCDIR}/lib/darwin_amd64 -lbarretenberg -lc++ -lm
import "C"
