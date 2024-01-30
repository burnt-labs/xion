package types_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/dvsekhvalnov/jose2go/base64url"

	"github.com/burnt-labs/xion/x/xion/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndAuthenticate(t *testing.T) {
	config := sdktypes.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	addr, err := sdktypes.AccAddressFromBech32("xion1cyyld62ly828e2xnp0c0ckpyz68wwfs26tjpscmqlaum2jcj8zdstlxvya")
	require.NoError(t, err)

	rp, err := url.Parse("https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app")
	require.NoError(t, err)

	challengeStr := "test-challenge"
	challenge := base64url.Encode([]byte(challengeStr))
	const registerStr = `{"type":"public-key","id":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","rawId":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZEdWemRDMWphR0ZzYkdWdVoyVSIsIm9yaWdpbiI6Imh0dHBzOi8veGlvbi1kYXBwLWV4YW1wbGUtZ2l0LWZlYXQtZmFjZWlkLWJ1cm50ZmluYW5jZS52ZXJjZWwuYXBwIiwiY3Jvc3NPcmlnaW4iOmZhbHNlfQ","attestationObject":"o2NmbXRkbm9uZWdhdHRTdG10oGhhdXRoRGF0YViksGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1BAAAAAK3OAAI1vMYKZIsLJfHwVQMAIOgZ6Uh5SF8Dp3R4cXz8OJd0spbqZ2SL01T_Vaf2it-MpQECAyYgASFYINnBKEMfG6wkb9W1grSXgNAQ8lx6H7j6EcMyTSbZ91-XIlggdk2OOxV_bISxCsqFac6ZE8-gEurV4xQd7kFFYdfMqtE","transports":["internal"]},"clientExtensionResults":{}}`

	data, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(registerStr))
	require.NoError(t, err)

	cred, err := types.VerifyRegistration(rp, addr, challenge, data)
	require.NoError(t, err)

	t.Logf("credential: %v", cred)

	authenticateStr := `{"type":"public-key","id":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","rawId":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uZ2V0IiwiY2hhbGxlbmdlIjoiZEdWemRDMWphR0ZzYkdWdVoyVSIsIm9yaWdpbiI6Imh0dHBzOi8veGlvbi1kYXBwLWV4YW1wbGUtZ2l0LWZlYXQtZmFjZWlkLWJ1cm50ZmluYW5jZS52ZXJjZWwuYXBwIiwiY3Jvc3NPcmlnaW4iOmZhbHNlfQ","authenticatorData":"sGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw0BAAAAAA","signature":"MEQCIF1Fm_XjFV5FjBRYXNN1WcDm0V4xbPn3sQ85gC34_FGmAiBzLYGsat3HwDcn4jh50gTW4mgGcmYqkvT2g1bfdFxElA","userHandle":null},"clientExtensionResults":{}}`

	authData, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(authenticateStr))
	require.NoError(t, err)

	verified, err := types.VerifyAuthentication(rp, addr, challenge, cred, authData)
	require.NoError(t, err)
	require.True(t, verified)
}
