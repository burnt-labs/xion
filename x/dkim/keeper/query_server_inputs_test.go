package keeper_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
)

// encodeStringAsPublicInputs packs s into n decimal field elements the way the
// circuit does: 31-byte little-endian chunks, zero padded.
func encodeStringAsPublicInputs(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "0"
	}
	data := []byte(s)
	for i := 0; i < n && len(data) > 0; i++ {
		chunk := data
		if len(chunk) > 31 {
			chunk = chunk[:31]
		}
		data = data[len(chunk):]
		le := make([]byte, len(chunk))
		for j := range chunk {
			le[len(chunk)-1-j] = chunk[j]
		}
		out[i] = new(big.Int).SetBytes(le).String()
	}
	return out
}

// authenticateInputs builds a request whose public inputs are consistent up to
// the subject check, for a pubkey saved under domain with the given hash.
func authenticateInputs(t *testing.T, domain, host, subject string, poseidonHash *big.Int) *types.QueryAuthenticateRequest {
	t.Helper()
	idx := types.DefaultPublicInputIndices()
	inputs := make([]string, idx.MinLength)
	for i := range inputs {
		inputs[i] = "0"
	}
	copy(inputs[idx.DkimDomainRange.Start:idx.DkimDomainRange.End], encodeStringAsPublicInputs(domain, int(idx.DkimDomainRange.Length())))
	inputs[idx.DkimHashIndex] = poseidonHash.String()
	copy(inputs[idx.EmailHostRange.Start:idx.EmailHostRange.End], encodeStringAsPublicInputs(host, int(idx.EmailHostRange.Length())))
	copy(inputs[idx.EmailSubjectRange.Start:idx.EmailSubjectRange.End], encodeStringAsPublicInputs(subject, int(idx.EmailSubjectRange.Length())))

	txParts, err := types.ConvertStringArrayToBigInt(inputs[idx.TxBytesRange.Start:idx.TxBytesRange.End])
	require.NoError(t, err)
	txBytes, err := types.ConvertBigIntArrayToString(txParts)
	require.NoError(t, err)

	return &types.QueryAuthenticateRequest{
		TxBytes:           []byte(txBytes),
		EmailHash:         inputs[idx.EmailHashIndex],
		Proof:             []byte(`{}`),
		PublicInputs:      inputs,
		AllowedEmailHosts: []string{host},
	}
}

func TestEncodeStringAsPublicInputsRoundTrip(t *testing.T) {
	for _, s := range []string{"gmail.com", "[Reply Needed] a fairly long subject line that spans chunks"} {
		parts, err := types.ConvertStringArrayToBigInt(encodeStringAsPublicInputs(s, 9))
		require.NoError(t, err)
		decoded, err := types.ConvertBigIntArrayToString(parts)
		require.NoError(t, err)
		require.Equal(t, s, decoded)
	}
}

func TestAuthenticateInputValidation(t *testing.T) {
	f := SetupTest(t)

	hash, err := types.ComputePoseidonHash(testPubKey)
	require.NoError(t, err)
	_, err = keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
		Domain:       "gmail.com",
		Selector:     "s1",
		PubKey:       testPubKey,
		PoseidonHash: hash.Bytes(),
	}, &f.k)
	require.NoError(t, err)

	t.Run("proof larger than the cap is rejected before parsing", func(t *testing.T) {
		req := authenticateInputs(t, "gmail.com", "user@example.com", "[Reply Needed] hi", hash)
		req.Proof = bytes.Repeat([]byte("x"), int(types.MaxDKIMProofSizeBytes)+1)
		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.ErrorIs(t, err, types.ErrProofTooLarge)
		require.Nil(t, res)
	})

	t.Run("subject public inputs must be numeric", func(t *testing.T) {
		req := authenticateInputs(t, "gmail.com", "user@example.com", "[Reply Needed] hi", hash)
		req.PublicInputs[types.DefaultEmailSubjectRangeStart] = "not-a-number"
		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.ErrorIs(t, err, types.ErrInvalidPublicInput)
		require.Contains(t, err.Error(), "email subject")
		require.Nil(t, res)
	})

	t.Run("subject without the required tag is rejected", func(t *testing.T) {
		req := authenticateInputs(t, "gmail.com", "user@example.com", "just a subject", hash)
		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.ErrorIs(t, err, types.ErrInvalidEmailSubject)
		require.Nil(t, res)
	})

	t.Run("missing verification key surfaces as an error", func(t *testing.T) {
		p := types.DefaultParams()
		p.VkeyIdentifier = 999
		require.NoError(t, f.k.SetParams(f.ctx, p))
		t.Cleanup(func() { require.NoError(t, f.k.SetParams(f.ctx, types.DefaultParams())) })

		req := authenticateInputs(t, "gmail.com", "someone@example.org", "[Reply Needed] hi", hash)
		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("undecodable pubkey entry under the domain fails the lookup", func(t *testing.T) {
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "corrupt.com",
			Selector:     "s1",
			PubKey:       testPubKey,
			PoseidonHash: hash.Bytes(),
		}, &f.k)
		require.NoError(t, err)
		f.corruptDkimPubKey(t, "corrupt.com", "zz")
		req := authenticateInputs(t, "corrupt.com", "user@example.com", "[Reply Needed] hi", hash)
		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(t, err)
		require.Nil(t, res)
	})
}

func TestQueryDkimPubKeysKeyPagination(t *testing.T) {
	f := SetupTest(t)
	domain := "paged.com"
	count := 5
	records := CreateNDkimPubKey(t, domain, testPubKey, types.Version_VERSION_DKIM1_UNSPECIFIED, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, count)
	_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
		Authority:   f.govModAddr,
		DkimPubkeys: records,
	})
	require.NoError(t, err)

	first, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
		Domain:     domain,
		Pagination: &query.PageRequest{Limit: 2},
	})
	require.NoError(t, err)
	require.Len(t, first.DkimPubKeys, 2)
	require.NotEmpty(t, first.Pagination.NextKey)

	t.Run("next key resumes without counting the remainder", func(t *testing.T) {
		second, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:     domain,
			Pagination: &query.PageRequest{Key: first.Pagination.NextKey, Limit: 2},
		})
		require.NoError(t, err)
		require.Len(t, second.DkimPubKeys, 2)
		require.NotEqual(t, first.DkimPubKeys[0].Selector, second.DkimPubKeys[0].Selector)
		require.NotEmpty(t, second.Pagination.NextKey)
		// iteration stops at the first record past the page, so the total
		// only reflects what was visited
		require.Equal(t, uint64(3), second.Pagination.Total)
	})

	t.Run("next key with count total walks every record", func(t *testing.T) {
		second, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:     domain,
			Pagination: &query.PageRequest{Key: first.Pagination.NextKey, Limit: 2, CountTotal: true},
		})
		require.NoError(t, err)
		require.Len(t, second.DkimPubKeys, 2)
		require.Equal(t, uint64(3), second.Pagination.Total)
	})

	t.Run("last page has no next key", func(t *testing.T) {
		second, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:     domain,
			Pagination: &query.PageRequest{Key: first.Pagination.NextKey, Limit: 10},
		})
		require.NoError(t, err)
		require.Len(t, second.DkimPubKeys, 3)
		require.Empty(t, second.Pagination.NextKey)
	})

	t.Run("key past the end yields nothing", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:     domain,
			Pagination: &query.PageRequest{Key: []byte{0xff, 0xff}, Limit: 2},
		})
		require.NoError(t, err)
		require.Empty(t, res.DkimPubKeys)
	})
}

func TestQueryDkimPubKeysMalformedKeys(t *testing.T) {
	t.Run("selector with an embedded delimiter breaks next key encoding", func(t *testing.T) {
		f := SetupTest(t)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{Domain: "a.com", Selector: "s1", PubKey: testPubKey}, &f.k)
		require.NoError(t, err)
		_, err = keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{Domain: "a.com", Selector: "s2\x00x", PubKey: testPubKey}, &f.k)
		require.NoError(t, err)

		// the record past the first page is the one that cannot be re-encoded
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Pagination: &query.PageRequest{Limit: 1},
		})
		require.Error(t, err)
		require.Nil(t, res)

		// fast-forwarding to a key has to encode every key it skips
		res, err = f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Pagination: &query.PageRequest{Key: []byte{0xff}, Limit: 1},
		})
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("raw key without a delimiter cannot be decoded", func(t *testing.T) {
		f := SetupTest(t)
		rawKey := append(append([]byte{}, types.DkimPrefix.Bytes()...), []byte("no-delimiter")...)
		f.ctx.KVStore(f.dkimKey).Set(rawKey, []byte{})

		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Pagination: &query.PageRequest{Key: []byte{0xff}, Limit: 1},
		})
		require.Error(t, err)
		require.Nil(t, res)
	})
}
