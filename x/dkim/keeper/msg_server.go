package keeper

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func SaveDkimPubKey(ctx context.Context, dkimKey types.DkimPubKey, k *Keeper) (bool, error) {
	key := collections.Join(dkimKey.Domain, dkimKey.Selector)
	//nolint:govet // copylocks: unavoidable when storing protobuf messages in collections.Map
	if err := k.DkimPubKeys.Set(ctx, key, dkimKey); err != nil {
		return false, err
	}
	return true, nil
}

func SaveDkimPubKeys(ctx context.Context, dkimKeys []types.DkimPubKey, k *Keeper) (bool, error) {
	for _, dkimKey := range dkimKeys {
		if isSaved, err := SaveDkimPubKey(ctx, dkimKey, k); !isSaved {
			return false, err
		}
	}
	return true, nil
}

// AddDkimPubKey implements types.MsgServer.
func (ms msgServer) AddDkimPubKeys(ctx context.Context, msg *types.MsgAddDkimPubKeys) (*types.MsgAddDkimPubKeysResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Validate all DKIM public keys before saving
	if err := types.ValidateDkimPubKeysWithRevocation(ctx, msg.DkimPubkeys, params, func(c context.Context, pubKey string) (bool, error) {
		return ms.k.RevokedKeys.Has(c, pubKey)
	}); err != nil {
		return nil, err
	}

	_, err = SaveDkimPubKeys(ctx, msg.DkimPubkeys, &ms.k)
	if err != nil {
		return nil, err
	}
	return &types.MsgAddDkimPubKeysResponse{}, nil
}

// RemoveDkimPubKey implements types.MsgServer.
func (ms msgServer) RemoveDkimPubKey(ctx context.Context, msg *types.MsgRemoveDkimPubKey) (*types.MsgRemoveDkimPubKeyResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}
	key := collections.Join(msg.Domain, msg.Selector)
	if err := ms.k.DkimPubKeys.Remove(ctx, key); err != nil {
		return nil, err
	}
	return &types.MsgRemoveDkimPubKeyResponse{}, nil
}

// RevokeDkimPubKey implements types.MsgServer.
func (ms msgServer) RevokeDkimPubKey(ctx context.Context, msg *types.MsgRevokeDkimPubKey) (*types.MsgRevokeDkimPubKeyResponse, error) {
	// providing a domain and private key revokes all pubkeys for that domain
	// that match the private key

	var privateKey *rsa.PrivateKey
	d, _ := pem.Decode(msg.PrivKey)
	if d == nil {
		return nil, errors.Wrap(types.ErrParsingPrivKey, "failed to decode pem private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(d.Bytes); err != nil {
		if key, err := x509.ParsePKCS8PrivateKey(d.Bytes); err != nil {
			return nil, errors.Wrap(types.ErrParsingPrivKey, "failed to parse private key")
		} else {
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return nil, errors.Wrap(types.ErrParsingPrivKey, "key is not an RSA private key")
			}
			privateKey = rsaKey
		}
	} else {
		privateKey = key
	}

	publicKey := privateKey.PublicKey
	// Marshal the public key to PKCS1 DER format
	pubKeyDER := x509.MarshalPKCS1PublicKey(&publicKey)

	// Encode the public key in PEM format
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyDER,
	})
	// remove the PEM header and footer from the public key
	after, _ := strings.CutPrefix(string(pubKeyPEM), "-----BEGIN RSA PUBLIC KEY-----\n")
	pubKey, _ := strings.CutSuffix(after, "\n-----END RSA PUBLIC KEY-----\n")
	pubKey = strings.ReplaceAll(pubKey, "\n", "")

	iter, err := ms.k.DkimPubKeys.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	kvs, err := iter.KeyValues()
	if err != nil {
		return nil, err
	}
	for i := range kvs {
		if kvs[i].Value.Domain == msg.Domain && kvs[i].Value.PubKey == pubKey {
			if err := ms.k.DkimPubKeys.Remove(ctx, kvs[i].Key); err != nil {
				return nil, err
			}
			if err := ms.k.RevokedKeys.Set(ctx, pubKey, true); err != nil {
				return nil, err
			}
		}
	}

	return &types.MsgRevokeDkimPubKeyResponse{}, nil
}

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	if msg.Params.MaxPubkeySizeBytes == 0 {
		msg.Params.MaxPubkeySizeBytes = types.DefaultMaxPubKeySizeBytes
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, errors.Wrap(types.ErrInvalidParams, err.Error())
	}

	return nil, ms.k.SetParams(ctx, msg.Params)
}
