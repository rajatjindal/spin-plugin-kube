package debug

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMe(t *testing.T) {
	defer t.Fail()

	_, err := DoIt("ghcr.io/spinkube/containerd-shim-spin/examples/spin-rust-hello:v0.13.0")
	require.Nil(t, err)
}

func TestPull(t *testing.T) {
	defer t.Fail()

	tempdir, _ := os.MkdirTemp("", "")
	err := PullArtifact("ttl.sh/rajatjindal/wasm-console-debug-1721062385:24h", tempdir)
	require.Nil(t, err)
	fmt.Println(tempdir)
}
