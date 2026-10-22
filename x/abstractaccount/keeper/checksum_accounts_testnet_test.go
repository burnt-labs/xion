package keeper_test

import (
	"encoding/hex"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
)

// testnetCode1880Checksum is the checksum of code 1880 on xion-testnet-2, the
// code app/upgrades.go backfills there.
const testnetCode1880Checksum = "D27A379FF65EB47A9E538E3A3D46101DE2A6C0B86BA3D0BF014C0403849414E6"

// TestChecksumAccountSaltDerivesRealTestnetAccounts checks the salt recovery
// and address derivation the v32 backfill relies on against accounts that
// exist on xion-testnet-2: one per authenticator kind, each registered under
// code 1880 at its checksum-derived address. The INIT messages are the
// accounts' own (contract history at v31.0.0, height 18,991,759); the JWT's
// token field is left out because the salt does not depend on it. The
// expected salts come from an independent recount of all 2,484 such accounts.
func TestChecksumAccountSaltDerivesRealTestnetAccounts(t *testing.T) {
	const creator = "xion1xrqz2wpt4rw8rtdvrc4n4yn5h54jm0nn4evn2x"
	cases := []struct {
		kind    string
		account string
		initMsg string
		salt    string
	}{
		{
			kind:    "Secp256K1",
			account: "xion1vmcakx3wpn2c8nvfz5kcwzzply6kt6p92y682uqv844q3xlfpvvsdl2qe2",
			initMsg: `{"authenticator":{"Secp256K1":{"id":0,"pubkey":"AoSIyiDHja9jTn9GTQgj6jCBjxus0sEGqzVSHjzC930B","signature":"+J27gk8+5CEXalqJZW1Ox+vT0m12ukE+GDG/H/lQ2MEL8Z3dLJdEsiYZlJ4xt+rZPH97ol5JopIkNhAkpBuEsQ=="}}}`,
			salt:    "7f236c1a79863dd68252090f31d132175a6b87de2607b46cd5e81bf03c01d29c",
		},
		{
			kind:    "EthWallet",
			account: "xion1en6htv0mqclpp07rq79nqph4xh77xwmxnk9s5dc4f6q897c2h23qqcyvfu",
			initMsg: `{"authenticator":{"EthWallet":{"id":0,"address":"0x6553a52e41f0e464914457fb2dacfade2e0ca83d","signature":"mp8S70mBG6ajzsM4LF/Tj756TbHdA56IkVcxAUHtlYtHaOWPxm6TXh8alavN5TNaJsi5qAq4d962A8IAIrw0tRs="}}}`,
			salt:    "182586cd7937be414d0044a9848dab03214635672001639d47b32aee4834aded",
		},
		{
			kind:    "Jwt",
			account: "xion1ec6v887hynzhath97dfpns9x0eft5ex53frx8376gp4yh0sg2tmq2zrymm",
			initMsg: `{"authenticator":{"Jwt":{"id":0,"aud":"project-test-5ae234a7-6b74-46af-a7b7-969f3df38cc0","sub":"user-test-e894a473-9ead-45a7-ba13-442f89207d96"}}}`,
			salt:    "00fbdf72c7e373aae153f18d93ba7c2810bec6ec4ebf15858d47297247fa6eaa",
		},
	}
	checksum := mustHex(t, testnetCode1880Checksum)
	creatorBytes, err := sdk.GetFromBech32(creator, "xion")
	require.NoError(t, err)
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			salt, ok := keeper.ChecksumAccountSalt([]byte(tc.initMsg))
			require.True(t, ok)
			require.Equal(t, tc.salt, hex.EncodeToString(salt))
			account, err := sdk.GetFromBech32(tc.account, "xion")
			require.NoError(t, err)
			derived := wasmkeeper.BuildContractAddressPredictable(checksum, creatorBytes, salt, nil)
			require.Equal(t, sdk.AccAddress(account), derived, "%s account must be the instantiate2 address of (code 1880 checksum, creator, salt)", tc.kind)
		})
	}
}
