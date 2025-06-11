package keeper_test

import (
	"testing"

	b64 "encoding/base64"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/orm/types/ormerrors"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestQueryDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	count := 10
	domain := "xion.burnt.com"
	pubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	createReq := CreateNDkimPubKey(t, domain, pubKey, types.Version_DKIM1, types.KeyType_RSA, count)
	hash, err := types.ComputePoseidonHash(pubKey)
	require.NoError(err)

	testCases := []struct {
		name    string
		request *types.QueryDkimPubKeyRequest
		err     bool
		errType error
		result  *types.QueryDkimPubKeyResponse
	}{
		{
			name: "fail; no such selector",
			request: &types.QueryDkimPubKeyRequest{
				Selector: "no-such-selector",
				Domain:   domain,
			},
			err:     true,
			errType: ormerrors.NotFound,
		},
		{
			name: "success",
			request: &types.QueryDkimPubKeyRequest{
				Domain:   "xion.burnt.com",
				Selector: createReq[0].Selector,
			},
			err: false,
			result: &types.QueryDkimPubKeyResponse{
				DkimPubKey: &types.DkimPubKey{
					Domain:       domain,
					PubKey:       pubKey,
					Selector:     createReq[0].Selector,
					Version:      types.Version_DKIM1,
					KeyType:      types.KeyType_RSA,
					PoseidonHash: hash.Bytes(),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
				Authority:   f.govModAddr,
				DkimPubkeys: createReq,
			})
			require.NoError(err)
			res, err := f.queryServer.DkimPubKey(f.ctx, tc.request)
			if tc.err {
				require.Error(err)
				require.Equal(err.Error(), tc.errType.Error())
			} else if tc.result != nil {
				require.NoError(err)
				require.EqualValues(tc.result, res)
			}
		})
	}
}

func TestQueryProofVerify(t *testing.T) {
	f := SetupTest(t)

	email, err := b64.StdEncoding.DecodeString("sAcYdn1nulpzJIM0RMaX6Vn5GPPGXuHxM//AfW7b7yU=")
	require.NoError(t, err)
	var emailBz [32]byte
	copy(emailBz[:], email)
	_, err = fr.LittleEndian.Element(&emailBz)
	require.NoError(t, err)

	proof64 := "eyJwaV9hIjpbIjg4MDE3Nzg4OTg5NTY3NTYxMDM5NDMwMTE1MzEzNjY0NjY4NTM3MzM1Nzk0MTI4MzI4ODM3NTQ0MjY4NzMwMDMyMjgzOTA0MDY2MzEiLCIxODA3NjgyNzE0MTg5Nzk1MTI3MTU4ODk2NTY1OTY5NjA5ODU0ODE0Njc1NTkyMjg5Nzk5Njk1MTc4NjA2NzE4NzcxOTA4OTg1NDQ5NSIsIjEiXSwicGlfYiI6W1siMTkzNDU4MDkxMzk5ODkxMzcwNTczMjQ0MTgzMjgzNDk3OTE0NjI1NDEwMjgyODczNjY1MzA1ODMwNTI1OTAzOTk4NDEzMDU4NDE0ODUiLCIyMTEzNDA5OTMxOTEwODcwMDQzMTQ2MDY2MTA3MTYzMDk4MzU4NDE1MjAxNTA5NjY2MjI0MjczOTE5Mjc2MTAzNTE4NzEyMTY4ODI5MiJdLFsiMTUwMDc2MzI4MjQ5MjQ3MjAyMjg5NTgzMjkwMDUxNzE2NzkwMjIyMTUzOTM1MzkyMDY3MTA4MDM0OTAwMjE3MDIzNjUxNTExNTk0MDEiLCIxMjczOTQyMTYzNzgxMjg3NDA3ODA5MDMzMzg0MTg0NTY1MzgwMjAyOTE0NTE2NDc3MDU4NDA4NDg2MDgwMDE0MDIxMDE1NTI3NjEyOCJdLFsiMSIsIjAiXV0sInBpX2MiOlsiNTcyNjkxMTk3Nzk1ODg2MjA0MTY3MjU3NTg1MjQxNDEwMTcyNTA4NTI5NzM2NDk4MDI4OTc5NzM1OTkyNjIyMjUzMjY4NTQ2MTgwNSIsIjQ0MzEyMjk1MDU2NTQ3MzcwMDkyMDM1NDMyNjM3NTk3MjM5NDk2MDYwMTg1MTY1MzczMTQ3NTk3MjkxNzA1NzU3NzEwOTQ0MDA3MTQiLCIxIl19"
	proofData, err := b64.StdEncoding.DecodeString(proof64)
	require.NoError(t, err)

	txB4s := "CqIBCp8BChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEn8KP3hpb24xczN1YWU1MDUydTVnNzd3ZmxheHd5aDJ1dHg5OWU3aDZjNjJhNjI5M3J5cnh0Z3F2djhncXhlZ3J0axIreGlvbjFxYWYyeGZseDVqM2FndGx2cWs1dmhqcGV1aGw2ZzQ1aHhzaHdxahoPCgV1eGlvbhIGMTAwMDAwEhYSFAoOCgV1eGlvbhIFNjAwMDAQwJoMGgZ4aW9uLTEgCw=="
	dkimBz, err := b64.StdEncoding.DecodeString("iEeNSGFNAiTctrIgoVuE40DFz/ATm+ip5RBx3HfHqQ4=")
	require.NoError(t, err)

	testCases := []struct {
		name    string
		proofBz []byte
		txBz    string
		dkimBz  []byte
		emailBz []byte
	}{
		{
			name:    "verify proof",
			proofBz: proofData,
			txBz:    txB4s,
			dkimBz:  dkimBz,
			emailBz: email,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			r := &types.QueryVerifyRequest{
				TxBytes:   []byte(tc.txBz),
				Proof:     tc.proofBz,
				DkimHash:  tc.dkimBz,
				EmailHash: tc.emailBz,
			}
			_, err := f.queryServer.ProofVerify(f.ctx, r)
			require.NoError(t, err)
		})
	}
}
