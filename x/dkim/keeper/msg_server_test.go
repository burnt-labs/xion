package keeper_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"

	"github.com/burnt-labs/xion/x/dkim/types"
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
)

func TestParams(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	testCases := []struct {
		name    string
		request *types.MsgUpdateParams
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.MsgUpdateParams{
				Authority: f.addrs[0].String(),
				Params:    types.DefaultParams(),
			},
			err: true,
		},
		{
			name: "success",
			request: &types.MsgUpdateParams{
				Authority: f.govModAddr,
				Params:    types.DefaultParams(),
			},
			err: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			_, err := f.msgServer.UpdateParams(f.ctx, tc.request)

			if tc.err {
				require.Error(err)
			} else {
				require.NoError(err)

				r, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
				require.NoError(err)

				require.EqualValues(tc.request.Params, *(r.Params))
			}
		})
	}
}

func TestAddDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	const PubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

	testCases := []struct {
		name    string
		request *types.MsgAddDkimPubKeys
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid keytype",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
						KeyType:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid version",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
						Version:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid pubkey",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.govModAddr,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   "123456789",
						Selector: "zkemail",
						Version:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "success",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.govModAddr,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
					},
				},
			},
			err: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			err := tc.request.ValidateBasic()
			if tc.err {
				require.Error(err)
			} else {
				_, err = f.msgServer.AddDkimPubKeys(f.ctx, tc.request)
				require.NoError(err)

				r, err := f.queryServer.DkimPubKey(f.ctx, &types.QueryDkimPubKeyRequest{
					Domain:   tc.request.DkimPubkeys[0].Domain,
					Selector: tc.request.DkimPubkeys[0].Selector,
				})
				require.NoError(err)

				require.EqualValues(tc.request.DkimPubkeys[0].PubKey, r.DkimPubKey.PubKey)
				require.EqualValues(types.Version_DKIM1, r.DkimPubKey.Version)
				require.EqualValues(types.KeyType_RSA, r.DkimPubKey.KeyType)
			}
		})
	}
}

func TestRemoveDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	const domain = "xion.burnt.com"
	const PubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	selector := "zkemail"

	_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
		Authority: f.govModAddr,
		DkimPubkeys: []types.DkimPubKey{
			{
				Domain:   domain,
				PubKey:   PubKey,
				Selector: selector,
			},
		},
	})
	require.NoError(err)

	testCases := []struct {
		name    string
		request *types.MsgRemoveDkimPubKey
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.addrs[0].String(),
				Domain:    domain,
				Selector:  selector,
			},
			err: true,
		},
		{
			name: "fail: remove non existing key",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.govModAddr,
				Domain:    domain,
				Selector:  "non-existing",
			},
			err: true,
		},
		{
			name: "success",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.govModAddr,
				Domain:    domain,
				Selector:  selector,
			},
			err: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			_, err := f.msgServer.RemoveDkimPubKey(f.ctx, tc.request)

			if tc.err {
				require.Error(err)
				if tc.name == "fail: remove non existing key" {
					require.ErrorIs(err, collections.ErrKeyNotFound)
				}
			} else {
				require.NoError(err)

				r, err := f.queryServer.DkimPubKey(f.ctx, &types.QueryDkimPubKeyRequest{
					Domain:   tc.request.Domain,
					Selector: tc.request.Selector,
				})
				require.Nil(r)
				require.Error(err)
				require.ErrorIs(err, collections.ErrKeyNotFound)
			}
		})
	}
}

func TestRevokeDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	// Generate a test RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	// Extract the public key
	publicKey := privateKey.PublicKey

	// Marshal the public key to PKCS1 DER format
	pubKeyDER := x509.MarshalPKCS1PublicKey(&publicKey)

	// Encode the public key in PEM format
	pubKeyPEM_1 := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyDER,
	})

	// Encode private key as PEM
	privKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)
	// remove the PEM header and footer from the public key
	after, _ := strings.CutPrefix(string(pubKeyPEM_1), "-----BEGIN RSA PUBLIC KEY-----\n")
	pubKey_1, _ := strings.CutSuffix(after, "\n-----END RSA PUBLIC KEY-----\n")
	pubKey_1 = strings.ReplaceAll(pubKey_1, "\n", "")
	hash_1, err := types.ComputePoseidonHash(pubKey_1)
	require.NoError(t, err)
	domain_1 := "x.com"

	privateKey_2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	// Extract the public key
	publicKey_2 := privateKey_2.PublicKey

	// Marshal the public key to PKCS1 DER format
	pubKeyDER_2 := x509.MarshalPKCS1PublicKey(&publicKey_2)

	// Encode the public key in PEM format
	pubKeyPEM_2 := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyDER_2,
	})

	privKeyPEM_2 := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey_2),
		},
	)
	// remove the PEM header and footer from the public key
	after, _ = strings.CutPrefix(string(pubKeyPEM_2), "-----BEGIN RSA PUBLIC KEY-----\n")
	pubKey_2, _ := strings.CutSuffix(after, "\n-----END RSA PUBLIC KEY-----\n")
	pubKey_2 = strings.ReplaceAll(pubKey_2, "\n", "")
	hash_2, err := types.ComputePoseidonHash(pubKey_2)
	require.NoError(t, err)
	domain_2 := "y.com"

	// Add in a DKIM public key
	addDkimKeysMsg := types.NewMsgAddDkimPubKeys(sdkTypes.MustAccAddressFromBech32(f.govModAddr), []types.DkimPubKey{
		{
			Domain:       domain_1,
			PubKey:       pubKey_1,
			Selector:     "dkim-202308",
			PoseidonHash: []byte(hash_1.String()),
		},
		{
			Domain:       domain_1,
			PubKey:       pubKey_2,
			Selector:     "dkim-202310",
			PoseidonHash: []byte(hash_2.String()),
		},
		{
			Domain:       domain_2,
			PubKey:       pubKey_1,
			Selector:     "dkim-202308",
			PoseidonHash: []byte(hash_1.String()),
		},
		{
			Domain:       domain_2,
			PubKey:       pubKey_2,
			Selector:     "dkim-202310",
			PoseidonHash: []byte(hash_2.String()),
		},
	})
	addDkimKeysMsg.Authority = f.govModAddr
	require.NoError(t, addDkimKeysMsg.ValidateBasic())
	_, err = f.msgServer.AddDkimPubKeys(f.ctx, addDkimKeysMsg)
	require.NoError(t, err)

	// Define test cases
	tests := []struct {
		name           string
		msg            *types.MsgRevokeDkimPubKey
		mockError      error
		expectedError  error
		expectedLength int8
	}{
		{
			name: "invalid private key",
			msg: &types.MsgRevokeDkimPubKey{
				Signer:  string(f.addrs[0]),
				Domain:  domain_1,
				PrivKey: []byte("invalid_key"),
			},
			mockError:     nil,
			expectedError: types.ErrParsingPrivKey,
		},
		{
			name: "successfully revoke 1 of domain 1 DKIM public key",
			msg: &types.MsgRevokeDkimPubKey{
				Signer:  string(f.addrs[0]),
				Domain:  domain_1,
				PrivKey: privKeyPEM,
			},
			mockError:      nil,
			expectedError:  nil,
			expectedLength: 1, // domain_1 has 1 keys, 1 with the matching pub key should be revoked, 1 with a different pubkey should be left
		},
		{
			name: "successfully revoke 1 of domain 2 DKIM public key",
			msg: &types.MsgRevokeDkimPubKey{
				Signer:  string(f.addrs[0]),
				Domain:  domain_2,
				PrivKey: privKeyPEM_2,
			},
			mockError:      nil,
			expectedError:  nil,
			expectedLength: 1, // domain_2 has 1 keys, 1 with the matching pub key should be revoked, 1 with a different pubkey should be left
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := f.ctx
			if strings.Contains(tt.name, "success") {
				require.NoError(t, tt.msg.ValidateBasic())
			}
			// Call the RevokeDkimPubKey method
			_, err := f.msgServer.RevokeDkimPubKey(ctx, tt.msg)
			// Validate results
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
			} else {
				res, err := f.queryServer.DkimPubKeys(ctx, &types.QueryDkimPubKeysRequest{Domain: tt.msg.Domain})
				require.NoError(t, err)
				require.Len(t, res.DkimPubKeys, int(tt.expectedLength))
			}
		})
	}
}
