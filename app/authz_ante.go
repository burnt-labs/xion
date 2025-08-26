package app

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// AuthzLimiterDecorator prevents dangerous message types from being executed via authz
type AuthzLimiterDecorator struct {
	restrictedMessages map[string]bool
}

// NewAuthzLimiterDecorator creates a new AuthzLimiterDecorator with specified restricted message types
func NewAuthzLimiterDecorator(restrictedMsgTypes []string) AuthzLimiterDecorator {
	restricted := make(map[string]bool)
	for _, msgType := range restrictedMsgTypes {
		restricted[msgType] = true
	}
	return AuthzLimiterDecorator{
		restrictedMessages: restricted,
	}
}

// AnteHandle implements the AnteDecorator interface
func (ald AuthzLimiterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Check all messages in the transaction
	for _, msg := range tx.GetMsgs() {
		// If this is an authz MsgExec, inspect nested messages
		if authzMsg, ok := msg.(*authz.MsgExec); ok {
			if err := ald.ValidateAuthzMessages(authzMsg); err != nil {
				return ctx, err
			}
		}
	}
	
	return next(ctx, tx, simulate)
}

// ValidateAuthzMessages checks if any nested messages in authz.MsgExec are restricted
func (ald AuthzLimiterDecorator) ValidateAuthzMessages(authzMsg *authz.MsgExec) error {
	// Extract nested messages from authz
	nestedMsgs, err := authzMsg.GetMessages()
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "failed to get messages from authz: %v", err)
	}
	
	// Check each nested message against restricted types
	for _, nestedMsg := range nestedMsgs {
		msgType := sdk.MsgTypeURL(nestedMsg)
		if ald.restrictedMessages[msgType] {
			return errorsmod.Wrapf(
				sdkerrors.ErrUnauthorized,
				"message type %s is not allowed in authz execution due to security restrictions",
				msgType,
			)
		}
	}
	
	return nil
}
