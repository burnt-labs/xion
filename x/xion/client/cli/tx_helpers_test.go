package cli

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	aatypes "github.com/burnt-labs/abstract-account/x/abstractaccount/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// decodeMaybeBase64Bytes decodes a value that might be a base64 string produced by json.Marshal of []byte.
func decodeMaybeBase64Bytes(t *testing.T, v interface{}) []byte {
	switch x := v.(type) {
	case string:
		b, err := base64.StdEncoding.DecodeString(x)
		if err == nil {
			return b
		}
		return []byte(x)
	case []byte:
		return x
	default:
		t.Fatalf("unexpected type %T", v)
	}
	return nil
}

func TestTypeURL(t *testing.T) {
	msg := &aatypes.MsgRegisterAccount{}
	// Use gogo proto name retrieval since cosmos-sdk types use gogo
	// Expect fully-qualified type URL for the message
	got := typeURL(msg)
	require.Equal(t, "/abstractaccount.v1.MsgRegisterAccount", got)
}

func TestRegisterMsg(t *testing.T) {
	funds := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(5)))
	instantiateJSON := `{"foo":1}`
	msg := registerMsg("sender", "salt", instantiateJSON, 7, funds)
	require.Equal(t, uint64(7), msg.CodeID)
	require.Equal(t, "sender", msg.Sender)
	require.Equal(t, []byte("salt"), msg.Salt)
	require.Equal(t, []byte(instantiateJSON), []byte(msg.Msg))
	require.True(t, funds.Equal(msg.Funds))
}

func TestNewInstantiateMsg(t *testing.T) {
	pub := []byte{0x01, 0x02}
	sig := []byte{0x03}
	s, err := newInstantiateMsg("webauthn", 4, sig, pub)
	require.NoError(t, err)
	var m map[string]map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(s), &m))
	auth := m["authenticator"]["webauthn"]
	require.Equal(t, float64(4), auth["id"]) // JSON numbers -> float64
	require.Equal(t, sig, decodeMaybeBase64Bytes(t, auth["signature"]))
	require.Equal(t, pub, decodeMaybeBase64Bytes(t, auth["pubkey"]))
}

func TestNewInstantiateJwtMsg(t *testing.T) {
	s, err := newInstantiateJwtMsg("token123", "jwt", "subv", "audv", 9)
	require.NoError(t, err)
	var m map[string]map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(s), &m))
	auth := m["authenticator"]["jwt"]
	require.Equal(t, float64(9), auth["id"]) // numeric JSON default
	require.Equal(t, "subv", auth["sub"])
	require.Equal(t, "audv", auth["aud"])
	tokenBytes := decodeMaybeBase64Bytes(t, auth["token"])
	require.Equal(t, []byte("token123"), tokenBytes)
}

// NOTE: getSignerOfTx requires mocking authtypes.QueryClient (generated gRPC interface). A minimal stub
// implementation can be added later if coverage for that function specifically is required.

// Compile-time guard to ensure AbstractAccount still implements auth AccountI.
// (compile guard intentionally omitted due to gogo/proto differences)
