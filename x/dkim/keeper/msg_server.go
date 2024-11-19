package keeper

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"cosmossdk.io/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	dkimv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
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

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	return nil, ms.k.Params.Set(ctx, msg.Params)
}

func SaveDkimPubKey(ctx context.Context, dkimKey types.DkimPubKey, store dkimv1.StateStore) (bool, error) {
	if err := store.DkimPubKeyTable().Save(ctx, &dkimv1.DkimPubKey{
		Domain:       dkimKey.Domain,
		PubKey:       dkimKey.PubKey,
		Selector:     dkimKey.Selector,
		PoseidonHash: dkimKey.PoseidonHash,
		Version:      dkimv1.Version(dkimKey.Version),
		KeyType:      dkimv1.KeyType(dkimKey.KeyType),
	}); err != nil {
		return false, err
	}

	return true, nil
}

func SaveDkimPubKeys(ctx context.Context, dkimKeys []types.DkimPubKey, store dkimv1.StateStore) (bool, error) {
	for _, dkimKey := range dkimKeys {
		if isSaved, err := SaveDkimPubKey(ctx, dkimKey, store); !isSaved {
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
	_, err := SaveDkimPubKeys(ctx, msg.DkimPubkeys, ms.k.OrmDB)
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
	dkimPubKey, err := ms.k.OrmDB.DkimPubKeyTable().Get(ctx, msg.Domain, msg.Selector)
	if err != nil {
		return nil, err
	}
	if err := ms.k.OrmDB.DkimPubKeyTable().Delete(ctx, &dkimv1.DkimPubKey{
		Domain:   dkimPubKey.Domain,
		PubKey:   dkimPubKey.PubKey,
		Selector: dkimPubKey.Selector,
	}); err != nil {
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
			privateKey = key.(*rsa.PrivateKey)
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

	err := ms.k.OrmDB.DkimPubKeyTable().DeleteBy(ctx,
		dkimv1.DkimPubKeyDomainPubKeyIndexKey{}.WithDomainPubKey(msg.Domain, pubKey))

	return &types.MsgRevokeDkimPubKeyResponse{}, err
}
