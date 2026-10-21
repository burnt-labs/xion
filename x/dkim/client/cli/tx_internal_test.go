package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
)

func TestPrintPrettyJSONFallsBackToRawOutput(t *testing.T) {
	var buf bytes.Buffer
	clientCtx := client.Context{}.WithOutput(&buf)

	require.NoError(t, printPrettyJSON(clientCtx, []byte("[1, 2]")))
	require.Equal(t, "[1, 2]\n", buf.String())

	buf.Reset()
	require.NoError(t, printPrettyJSON(clientCtx, []byte(`{"b":1,"a":2}`)))
	require.Equal(t, "{\n  \"a\": 2,\n  \"b\": 1\n}\n", buf.String())
}
