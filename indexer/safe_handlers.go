package indexer

import (
	"context"
	"encoding/hex"

	"cosmossdk.io/collections"
	"cosmossdk.io/log/v2"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

// SafeAuthzHandlerUpdate wraps the authz handler update with safe error handling
func SafeAuthzHandlerUpdate(ctx context.Context, ah *AuthzHandler, pair *storetypes.StoreKVPair, logger log.Logger) error {
	granterAddr, granteeAddr, msgType := parseGrantStoreKey(pair.Key)

	if pair.Delete {
		// Check if exists
		has, err := ah.Authorizations.Has(ctx, collections.Join3(granterAddr, granteeAddr, msgType))
		if err != nil {
			logger.Error("Failed to check grant existence",
				"granter", granterAddr.String(),
				"grantee", granteeAddr.String(),
				"msg_type", msgType,
				"error", err)
			return nil // Don't halt on read errors
		}

		if !has {
			// Already handled gracefully in our previous fix
			return nil
		}

		// Try to remove
		if err := ah.Authorizations.Remove(ctx, collections.Join3(granterAddr, granteeAddr, msgType)); err != nil {
			logger.Error("Failed to remove grant from index",
				"granter", granterAddr.String(),
				"grantee", granteeAddr.String(),
				"msg_type", msgType,
				"error", err)
			return nil // Don't halt on removal errors
		}

		return nil
	}

	// Handle create/update
	grant := authz.Grant{}
	err := ah.cdc.Unmarshal(pair.Value, &grant)
	if err != nil {
		logger.Warn("Failed to unmarshal authz grant, skipping",
			"key", hex.EncodeToString(pair.Key),
			"granter", granterAddr.String(),
			"grantee", granteeAddr.String(),
			"msg_type", msgType,
			"error", err)
		return nil // Skip corrupted entries
	}

	// Try to set the grant
	if err := ah.SetGrant(ctx, granterAddr, granteeAddr, msgType, grant); err != nil {
		logger.Error("Failed to index authz grant",
			"granter", granterAddr.String(),
			"grantee", granteeAddr.String(),
			"msg_type", msgType,
			"error", err)
		return nil // Don't halt on write errors
	}

	return nil
}

// SafeFeeGrantHandlerUpdate wraps the feegrant handler update with safe error handling
func SafeFeeGrantHandlerUpdate(ctx context.Context, fh *FeeGrantHandler, pair *storetypes.StoreKVPair, logger log.Logger) error {
	granterAddr, granteeAddr, err := parseFeeAllowanceKey(pair.Key)
	if err != nil {
		logger.Warn("Failed to parse feegrant key addresses",
			"error", err,
			"key", hex.EncodeToString(pair.Key))
		return nil
	}

	if pair.Delete {
		// Check if exists
		has, err := fh.FeeAllowances.Has(ctx, collections.Join(granterAddr, granteeAddr))
		if err != nil {
			logger.Error("Failed to check allowance existence",
				"granter", granterAddr.String(),
				"grantee", granteeAddr.String(),
				"error", err)
			return nil // Don't halt on read errors
		}

		if !has {
			// Already handled gracefully in our previous fix
			return nil
		}

		// Try to remove
		if err := fh.FeeAllowances.Remove(ctx, collections.Join(granterAddr, granteeAddr)); err != nil {
			logger.Error("Failed to remove allowance from index",
				"granter", granterAddr.String(),
				"grantee", granteeAddr.String(),
				"error", err)
			return nil // Don't halt on removal errors
		}

		return nil
	}

	// Handle create/update
	grant := feegrant.Grant{}
	err = fh.cdc.Unmarshal(pair.Value, &grant)
	if err != nil {
		logger.Warn("Failed to unmarshal feegrant allowance, skipping",
			"key", hex.EncodeToString(pair.Key),
			"granter", granterAddr.String(),
			"grantee", granteeAddr.String(),
			"error", err)
		return nil // Skip corrupted entries
	}

	// Try to set the grant
	if err := fh.SetGrant(ctx, granterAddr, granteeAddr, grant); err != nil {
		logger.Error("Failed to index feegrant allowance",
			"granter", granterAddr.String(),
			"grantee", granteeAddr.String(),
			"error", err)
		return nil // Don't halt on write errors
	}

	return nil
}
