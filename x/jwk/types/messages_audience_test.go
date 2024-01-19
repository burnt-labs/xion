package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"jwk/testutil/sample"
)

func TestMsgCreateAudience_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCreateAudience
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCreateAudience{
				Admin: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgCreateAudience{
				Admin: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgUpdateAudience_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUpdateAudience
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUpdateAudience{
				Admin: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgUpdateAudience{
				Admin: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgDeleteAudience_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgDeleteAudience
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgDeleteAudience{
				Admin: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgDeleteAudience{
				Admin: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
