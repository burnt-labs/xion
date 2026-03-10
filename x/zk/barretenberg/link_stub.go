//go:build barretenberg_stub

package barretenberg

// #cgo linux,amd64  LDFLAGS: -L${SRCDIR}/lib/linux_amd64  -lbarretenberg -lstdc++ -lm -lpthread
// #cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib/darwin_arm64 -lbarretenberg -lc++ -lm
// #cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/darwin_amd64 -lbarretenberg -lc++ -lm
import "C"
