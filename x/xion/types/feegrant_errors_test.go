package types_test

import (
	"context"
	"errors"
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
)

// unmarshalableAllowance is a fee allowance whose protobuf encoding always
// fails, which is the only way an allowance can be accepted but not packed
// back into an Any afterwards.
type unmarshalableAllowance struct{}

func (unmarshalableAllowance) Reset()                   {}
func (unmarshalableAllowance) String() string           { return "unmarshalable" }
func (unmarshalableAllowance) ProtoMessage()            {}
func (unmarshalableAllowance) Marshal() ([]byte, error) { return nil, errors.New("encoding refused") }
func (unmarshalableAllowance) ValidateBasic() error     { return nil }
func (unmarshalableAllowance) ExpiresAt() (*time.Time, error) {
	return nil, nil
}

func (unmarshalableAllowance) Accept(context.Context, sdk.Coins, []sdk.Msg) (bool, error) {
	return false, nil
}

var (
	errorsGrantee  = sdk.AccAddress("grantee_____________")
	errorsContract = sdk.AccAddress("contract____________")
	errorsFee      = sdk.NewCoins(sdk.NewInt64Coin("uxion", 10))
)

func gasContext() sdk.Context {
	return sdk.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
}

// anyWithoutValue looks like a packed allowance but was never unpacked, so
// the cached value every accessor relies on is missing.
func anyWithoutValue() *codectypes.Any {
	return &codectypes.Any{TypeUrl: "/cosmos.feegrant.v1beta1.BasicAllowance"}
}

func execMsg(t *testing.T, grantee sdk.AccAddress, msgs ...sdk.Msg) *authz.MsgExec {
	t.Helper()
	exec := authz.NewMsgExec(grantee, msgs)
	return &exec
}

func TestAllowanceConstructorsRejectUnencodableAllowance(t *testing.T) {
	broken := unmarshalableAllowance{}

	_, err := xiontypes.NewAuthzAllowance(broken, errorsGrantee)
	require.ErrorContains(t, err, "encoding refused")

	_, err = xiontypes.NewContractsAllowance(broken, []sdk.AccAddress{errorsContract})
	require.ErrorContains(t, err, "encoding refused")

	_, err = xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{&feegrant.BasicAllowance{}, broken})
	require.ErrorContains(t, err, "encoding refused")
}

func TestSetAllowanceRejectsUnencodableAllowance(t *testing.T) {
	broken := unmarshalableAllowance{}

	authzAllowance, err := xiontypes.NewAuthzAllowance(&feegrant.BasicAllowance{}, errorsGrantee)
	require.NoError(t, err)
	require.ErrorContains(t, authzAllowance.SetAllowance(broken), "cannot proto marshal")

	contractsAllowance, err := xiontypes.NewContractsAllowance(&feegrant.BasicAllowance{}, []sdk.AccAddress{errorsContract})
	require.NoError(t, err)
	require.ErrorContains(t, contractsAllowance.SetAllowance(broken), "cannot proto marshal")

	multi, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{&feegrant.BasicAllowance{}})
	require.NoError(t, err)
	require.ErrorContains(t, multi.SetAllowance(0, broken), "cannot proto marshal")
}

func TestAuthzAllowanceAcceptFailures(t *testing.T) {
	ctx := gasContext()
	send := &banktypes.MsgSend{}

	t.Run("message is not an authz exec", func(t *testing.T) {
		allowance, err := xiontypes.NewAuthzAllowance(&feegrant.BasicAllowance{}, errorsGrantee)
		require.NoError(t, err)

		_, err = allowance.Accept(ctx, errorsFee, []sdk.Msg{send})
		require.ErrorIs(t, err, feegrant.ErrMessageNotAllowed)
	})

	t.Run("exec carries undecodable messages", func(t *testing.T) {
		allowance, err := xiontypes.NewAuthzAllowance(&feegrant.BasicAllowance{}, errorsGrantee)
		require.NoError(t, err)

		exec := &authz.MsgExec{Grantee: errorsGrantee.String(), Msgs: []*codectypes.Any{{TypeUrl: "/cosmos.bank.v1beta1.MsgSend"}}}
		_, err = allowance.Accept(ctx, errorsFee, []sdk.Msg{exec})
		require.ErrorIs(t, err, feegrant.ErrMessageNotAllowed)
	})

	t.Run("inner allowance was never unpacked", func(t *testing.T) {
		allowance := &xiontypes.AuthzAllowance{Allowance: anyWithoutValue(), AuthzGrantee: errorsGrantee.String()}

		_, err := allowance.Accept(ctx, errorsFee, []sdk.Msg{execMsg(t, errorsGrantee, send)})
		require.ErrorContains(t, err, "failed to get allowance")
		require.ErrorContains(t, allowance.ValidateBasic(), "failed to get allowance")
	})

	t.Run("updated inner allowance cannot be repacked", func(t *testing.T) {
		allowance := &xiontypes.AuthzAllowance{
			Allowance:    codectypes.UnsafePackAny(unmarshalableAllowance{}),
			AuthzGrantee: errorsGrantee.String(),
		}

		_, err := allowance.Accept(ctx, errorsFee, []sdk.Msg{execMsg(t, errorsGrantee, send)})
		require.ErrorContains(t, err, "cannot proto marshal")
	})
}

func TestContractsAllowanceAcceptFailures(t *testing.T) {
	ctx := gasContext()
	execute := &wasmtypes.MsgExecuteContract{Contract: errorsContract.String()}

	t.Run("message is not a contract execution", func(t *testing.T) {
		allowance, err := xiontypes.NewContractsAllowance(&feegrant.BasicAllowance{}, []sdk.AccAddress{errorsContract})
		require.NoError(t, err)

		_, err = allowance.Accept(ctx, errorsFee, []sdk.Msg{&banktypes.MsgSend{}})
		require.ErrorIs(t, err, feegrant.ErrMessageNotAllowed)
	})

	t.Run("inner allowance was never unpacked", func(t *testing.T) {
		allowance := &xiontypes.ContractsAllowance{Allowance: anyWithoutValue(), ContractAddresses: []string{errorsContract.String()}}

		_, err := allowance.Accept(ctx, errorsFee, []sdk.Msg{execute})
		require.ErrorContains(t, err, "failed to get allowance")
		require.ErrorContains(t, allowance.ValidateBasic(), "failed to get allowance")
	})

	t.Run("updated inner allowance cannot be repacked", func(t *testing.T) {
		allowance := &xiontypes.ContractsAllowance{
			Allowance:         codectypes.UnsafePackAny(unmarshalableAllowance{}),
			ContractAddresses: []string{errorsContract.String()},
		}

		_, err := allowance.Accept(ctx, errorsFee, []sdk.Msg{execute})
		require.ErrorContains(t, err, "cannot proto marshal")
	})
}

func TestMultiAnyAllowanceAcceptFailures(t *testing.T) {
	ctx := gasContext()
	msgs := []sdk.Msg{&banktypes.MsgSend{}}

	t.Run("inner allowance was never unpacked", func(t *testing.T) {
		allowance := &xiontypes.MultiAnyAllowance{Allowances: []*codectypes.Any{anyWithoutValue()}}

		_, err := allowance.Accept(ctx, errorsFee, msgs)
		require.ErrorContains(t, err, "failed to get allowance")
		require.ErrorContains(t, allowance.ValidateBasic(), "failed to get allowance")
	})

	t.Run("updated inner allowance cannot be repacked", func(t *testing.T) {
		allowance := &xiontypes.MultiAnyAllowance{Allowances: []*codectypes.Any{codectypes.UnsafePackAny(unmarshalableAllowance{})}}

		_, err := allowance.Accept(ctx, errorsFee, msgs)
		require.ErrorContains(t, err, "cannot proto marshal")
	})
}

func TestMultiAnyAllowanceDropsExhaustedAllowances(t *testing.T) {
	ctx := gasContext()
	msgs := []sdk.Msg{&banktypes.MsgSend{}}
	exhaustible := func() *feegrant.BasicAllowance {
		return &feegrant.BasicAllowance{SpendLimit: errorsFee}
	}

	t.Run("last allowance exhausted removes the grant", func(t *testing.T) {
		allowance, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{exhaustible()})
		require.NoError(t, err)

		remove, err := allowance.Accept(ctx, errorsFee, msgs)
		require.NoError(t, err)
		require.True(t, remove)
		require.Empty(t, allowance.Allowances)
	})

	t.Run("exhausted allowance is dropped while others remain", func(t *testing.T) {
		remaining := &feegrant.BasicAllowance{SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 50))}
		allowance, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{exhaustible(), remaining})
		require.NoError(t, err)

		remove, err := allowance.Accept(ctx, errorsFee, msgs)
		require.NoError(t, err)
		require.False(t, remove)
		require.Len(t, allowance.Allowances, 1)

		left, err := allowance.GetAllowance(0)
		require.NoError(t, err)
		require.Equal(t, remaining.SpendLimit, left.(*feegrant.BasicAllowance).SpendLimit)
	})
}

func nestMultiAnyAllowance(t *testing.T, inner feegrant.FeeAllowanceI, levels int) *xiontypes.MultiAnyAllowance {
	t.Helper()
	var multi *xiontypes.MultiAnyAllowance
	for i := 0; i < levels; i++ {
		var err error
		multi, err = xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{inner})
		require.NoError(t, err)
		inner = multi
	}
	return multi
}

func TestMultiAnyAllowanceValidateBasicNesting(t *testing.T) {
	t.Run("nesting up to the limit is accepted", func(t *testing.T) {
		require.NoError(t, nestMultiAnyAllowance(t, &feegrant.BasicAllowance{}, 6).ValidateBasic())
	})

	t.Run("nesting past the limit is rejected", func(t *testing.T) {
		err := nestMultiAnyAllowance(t, &feegrant.BasicAllowance{}, 7).ValidateBasic()
		require.ErrorContains(t, err, "nesting exceeds maximum depth")
	})

	t.Run("invalid nested allowance is reported", func(t *testing.T) {
		negative := &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(-5)}}}
		err := nestMultiAnyAllowance(t, negative, 2).ValidateBasic()
		require.ErrorContains(t, err, "send amount is invalid")
	})

	t.Run("nested expiry mismatch is reported", func(t *testing.T) {
		now := time.Now()
		later := now.Add(time.Hour)
		inner, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{
			&feegrant.BasicAllowance{Expiration: &now},
			&feegrant.BasicAllowance{Expiration: &later},
		})
		require.NoError(t, err)
		outer, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{inner})
		require.NoError(t, err)

		_, err = outer.ExpiresAt()
		require.ErrorIs(t, err, xiontypes.ErrInconsistentExpiry)
		require.ErrorIs(t, outer.ValidateBasic(), xiontypes.ErrInconsistentExpiry)
	})
}
