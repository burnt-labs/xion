//go:build barretenberg_stub

package barretenberg

// #cgo linux,amd64  LDFLAGS: -L${SRCDIR}/lib/linux_amd64  -lbarretenberg_stub -lm -lpthread
// #cgo linux,arm64  LDFLAGS: -L${SRCDIR}/lib/linux_arm64  -lbarretenberg_stub -lm -lpthread
// #cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib/darwin_arm64 -lbarretenberg_stub
// #cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/darwin_amd64 -lbarretenberg_stub
import "C"
