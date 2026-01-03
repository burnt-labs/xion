package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
)

func TestMsgServer_AddVKey(t *testing.T) {
	f := SetupTest(t)

	t.Run("successfully add vkey", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   f.govModAddr,
			Name:        "test_vkey",
			VkeyBytes:   createTestVKeyBytes("test_vkey"),
			Description: "Test verification key",
		}

		resp, err := f.msgServer.AddVKey(f.ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, resp.Id, uint64(0))

		// Verify vkey was stored
		vkey, err := f.k.GetVKeyByName(f.ctx, "test_vkey")
		require.NoError(t, err)
		require.Equal(t, "test_vkey", vkey.Name)
		require.Equal(t, "Test verification key", vkey.Description)

		// Verify event was emitted
		events := f.ctx.EventManager().Events()
		require.NotEmpty(t, events)

		found := false
		for _, event := range events {
			if event.Type == types.EventTypeAddVKey {
				found = true
				for _, attr := range event.Attributes {
					switch attr.Key {
					case types.AttributeKeyVKeyName:
						require.Equal(t, "test_vkey", attr.Value)
					case types.AttributeKeyAuthority:
						require.Equal(t, f.govModAddr, attr.Value)
					}
				}
			}
		}
		require.True(t, found, "AddVKey event not found")
	})

	t.Run("successfully add with non-governance authority", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   f.addrs[0].String(),
			Name:        "user_vkey",
			VkeyBytes:   createTestVKeyBytes("user_vkey"),
			Description: "User added key",
		}

		resp, err := f.msgServer.AddVKey(f.ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("fail with empty name", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   f.govModAddr,
			Name:        "",
			VkeyBytes:   createTestVKeyBytes("empty_name"),
			Description: "Empty name",
		}

		resp, err := f.msgServer.AddVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("fail with empty vkey bytes", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   f.govModAddr,
			Name:        "empty_bytes",
			VkeyBytes:   []byte{},
			Description: "Empty bytes",
		}

		resp, err := f.msgServer.AddVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "vkey_bytes cannot be empty")
	})

	t.Run("fail with nil vkey bytes", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   f.govModAddr,
			Name:        "nil_bytes",
			VkeyBytes:   nil,
			Description: "Nil bytes",
		}

		resp, err := f.msgServer.AddVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "vkey_bytes cannot be empty")
	})

	t.Run("fail with invalid vkey bytes", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   f.govModAddr,
			Name:        "invalid_bytes",
			VkeyBytes:   []byte("not valid json"),
			Description: "Invalid bytes",
		}

		resp, err := f.msgServer.AddVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "invalid vkey_bytes")
	})

	t.Run("fail with duplicate name", func(t *testing.T) {
		// First add a vkey
		msg1 := &types.MsgAddVKey{
			Authority:   f.govModAddr,
			Name:        "duplicate_test",
			VkeyBytes:   createTestVKeyBytes("duplicate_test"),
			Description: "First",
		}

		resp1, err := f.msgServer.AddVKey(f.ctx, msg1)
		require.NoError(t, err)
		require.NotNil(t, resp1)

		// Try to add another with the same name
		msg2 := &types.MsgAddVKey{
			Authority:   f.govModAddr,
			Name:        "duplicate_test",
			VkeyBytes:   createTestVKeyBytes("duplicate_test"),
			Description: "Second",
		}

		resp2, err := f.msgServer.AddVKey(f.ctx, msg2)
		require.Error(t, err)
		require.Nil(t, resp2)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("fail with invalid authority address format", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   "invalid-address",
			Name:        "invalid_authority",
			VkeyBytes:   createTestVKeyBytes("invalid_authority"),
			Description: "Invalid authority format",
		}

		resp, err := f.msgServer.AddVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "invalid authority address")
	})
}

func TestMsgServer_UpdateVKey(t *testing.T) {
	f := SetupTest(t)

	// First add a vkey to update
	addMsg := &types.MsgAddVKey{
		Authority:   f.govModAddr,
		Name:        "update_test",
		VkeyBytes:   createTestVKeyBytes("update_test"),
		Description: "Original description",
	}
	_, err := f.msgServer.AddVKey(f.ctx, addMsg)
	require.NoError(t, err)

	t.Run("successfully update vkey", func(t *testing.T) {
		// Reset event manager for this test
		ctx := f.ctx.WithEventManager(sdk.NewEventManager())

		msg := &types.MsgUpdateVKey{
			Authority:   f.govModAddr,
			Name:        "update_test",
			VkeyBytes:   createTestVKeyBytes("update_test"),
			Description: "Updated description",
		}

		resp, err := f.msgServer.UpdateVKey(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify vkey was updated
		vkey, err := f.k.GetVKeyByName(ctx, "update_test")
		require.NoError(t, err)
		require.Equal(t, "Updated description", vkey.Description)

		// Verify event was emitted
		events := ctx.EventManager().Events()
		require.NotEmpty(t, events)

		found := false
		for _, event := range events {
			if event.Type == types.EventTypeUpdateVKey {
				found = true
				for _, attr := range event.Attributes {
					switch attr.Key {
					case types.AttributeKeyVKeyName:
						require.Equal(t, "update_test", attr.Value)
					case types.AttributeKeyAuthority:
						require.Equal(t, f.govModAddr, attr.Value)
					}
				}
			}
		}
		require.True(t, found, "UpdateVKey event not found")
	})

	t.Run("successfully update with non-governance authority", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   f.addrs[0].String(),
			Name:        "update_test",
			VkeyBytes:   createTestVKeyBytes("update_test"),
			Description: "User update",
		}

		resp, err := f.msgServer.UpdateVKey(f.ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("fail with non-existent vkey", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   f.govModAddr,
			Name:        "non_existent",
			VkeyBytes:   createTestVKeyBytes("non_existent"),
			Description: "Does not exist",
		}

		resp, err := f.msgServer.UpdateVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("fail with empty name", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   f.govModAddr,
			Name:        "",
			VkeyBytes:   createTestVKeyBytes("empty"),
			Description: "Empty name",
		}

		resp, err := f.msgServer.UpdateVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("fail with empty vkey bytes", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   f.govModAddr,
			Name:        "update_test",
			VkeyBytes:   []byte{},
			Description: "Empty bytes",
		}

		resp, err := f.msgServer.UpdateVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "vkey_bytes cannot be empty")
	})

	t.Run("fail with invalid vkey bytes", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   f.govModAddr,
			Name:        "update_test",
			VkeyBytes:   []byte("invalid json"),
			Description: "Invalid bytes",
		}

		resp, err := f.msgServer.UpdateVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "invalid verification key")
	})

	t.Run("fail with invalid authority address format", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   "invalid-address",
			Name:        "update_test",
			VkeyBytes:   createTestVKeyBytes("update_test"),
			Description: "Invalid authority format",
		}

		resp, err := f.msgServer.UpdateVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "invalid authority address")
	})
}

func TestMsgServer_RemoveVKey(t *testing.T) {
	f := SetupTest(t)

	// Add a vkey to remove
	addMsg := &types.MsgAddVKey{
		Authority:   f.govModAddr,
		Name:        "remove_test",
		VkeyBytes:   createTestVKeyBytes("remove_test"),
		Description: "To be removed",
	}
	_, err := f.msgServer.AddVKey(f.ctx, addMsg)
	require.NoError(t, err)

	// Add another vkey for subsequent tests
	addMsg2 := &types.MsgAddVKey{
		Authority:   f.govModAddr,
		Name:        "keep_test",
		VkeyBytes:   createTestVKeyBytes("keep_test"),
		Description: "To be kept",
	}
	_, err = f.msgServer.AddVKey(f.ctx, addMsg2)
	require.NoError(t, err)

	t.Run("successfully remove vkey", func(t *testing.T) {
		// Reset event manager for this test
		ctx := f.ctx.WithEventManager(sdk.NewEventManager())

		msg := &types.MsgRemoveVKey{
			Authority: f.govModAddr,
			Name:      "remove_test",
		}

		resp, err := f.msgServer.RemoveVKey(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify vkey was removed
		has, err := f.k.HasVKey(ctx, "remove_test")
		require.NoError(t, err)
		require.False(t, has)

		// Verify event was emitted
		events := ctx.EventManager().Events()
		require.NotEmpty(t, events)

		found := false
		for _, event := range events {
			if event.Type == types.EventTypeRemoveVKey {
				found = true
				for _, attr := range event.Attributes {
					switch attr.Key {
					case types.AttributeKeyVKeyName:
						require.Equal(t, "remove_test", attr.Value)
					case types.AttributeKeyAuthority:
						require.Equal(t, f.govModAddr, attr.Value)
					}
				}
			}
		}
		require.True(t, found, "RemoveVKey event not found")
	})

	t.Run("verify other vkeys not affected", func(t *testing.T) {
		has, err := f.k.HasVKey(f.ctx, "keep_test")
		require.NoError(t, err)
		require.True(t, has)

		vkey, err := f.k.GetVKeyByName(f.ctx, "keep_test")
		require.NoError(t, err)
		require.Equal(t, "To be kept", vkey.Description)
	})

	t.Run("successfully remove with non-governance authority", func(t *testing.T) {
		msg := &types.MsgRemoveVKey{
			Authority: f.addrs[0].String(),
			Name:      "keep_test",
		}

		resp, err := f.msgServer.RemoveVKey(f.ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("fail with non-existent vkey", func(t *testing.T) {
		msg := &types.MsgRemoveVKey{
			Authority: f.govModAddr,
			Name:      "non_existent",
		}

		resp, err := f.msgServer.RemoveVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("fail to remove already removed vkey", func(t *testing.T) {
		msg := &types.MsgRemoveVKey{
			Authority: f.govModAddr,
			Name:      "remove_test",
		}

		resp, err := f.msgServer.RemoveVKey(f.ctx, msg)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestMsgServer_FullLifecycle(t *testing.T) {
	f := SetupTest(t)

	// 1. Add a vkey
	addMsg := &types.MsgAddVKey{
		Authority:   f.govModAddr,
		Name:        "lifecycle_vkey",
		VkeyBytes:   createTestVKeyBytes("lifecycle_vkey"),
		Description: "Initial description",
	}

	addResp, err := f.msgServer.AddVKey(f.ctx, addMsg)
	require.NoError(t, err)
	require.NotNil(t, addResp)
	id := addResp.Id

	// 2. Verify it exists
	vkey, err := f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "lifecycle_vkey", vkey.Name)
	require.Equal(t, "Initial description", vkey.Description)

	// 3. Update the vkey
	updateMsg := &types.MsgUpdateVKey{
		Authority:   f.govModAddr,
		Name:        "lifecycle_vkey",
		VkeyBytes:   createTestVKeyBytes("lifecycle_vkey"),
		Description: "Updated description",
	}

	updateResp, err := f.msgServer.UpdateVKey(f.ctx, updateMsg)
	require.NoError(t, err)
	require.NotNil(t, updateResp)

	// 4. Verify the update
	vkey, err = f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Updated description", vkey.Description)

	// 5. Remove the vkey
	removeMsg := &types.MsgRemoveVKey{
		Authority: f.govModAddr,
		Name:      "lifecycle_vkey",
	}

	removeResp, err := f.msgServer.RemoveVKey(f.ctx, removeMsg)
	require.NoError(t, err)
	require.NotNil(t, removeResp)

	// 6. Verify it's gone
	has, err := f.k.HasVKey(f.ctx, "lifecycle_vkey")
	require.NoError(t, err)
	require.False(t, has)

	_, err = f.k.GetVKeyByID(f.ctx, id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestNewMsgServerImpl(t *testing.T) {
	f := SetupTest(t)

	// Test that NewMsgServerImpl returns a valid MsgServer
	msgServer := keeper.NewMsgServerImpl(f.k)
	require.NotNil(t, msgServer)

	// Verify it implements types.MsgServer by using it
	msg := &types.MsgAddVKey{
		Authority:   f.govModAddr,
		Name:        "impl_test",
		VkeyBytes:   createTestVKeyBytes("impl_test"),
		Description: "Implementation test",
	}

	resp, err := msgServer.AddVKey(f.ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
