package module

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query the current consensus parameters",
				},
				{
					RpcMethod: "DkimPubKey",
					Use:       "dkim-pubkey [flags] <domain> <selector>",
					Alias:     []string{"dkim"},
					Short:     "Query a DKIM public key",
					Example:   "dkim-pubkey --domain test.domain.com --selector test-domain",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // set to true if authority gated
				},
				{
					RpcMethod: "AddDkimPubKey",
					Short:     "Add a new DKIM public key",
					Long:      "Add a new DKIM public key",
					Alias:     []string{"adkim"},
					Use:       "add-dkim-pubkey [flags] <dkim_pubkeys>",
					Example:   "add-dkim-pubkey { domain: <domain>, pubKey: <pub-key>, selector: <selector> }...",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "dkim_pubkeys",
							Varargs:    true,
						},
					},
					Skip: true, // set to true if authority gated
				},
				{
					RpcMethod: "RemoveDkimPubKey",
					Short:     "Remove a new DKIM public key",
					Long:      "Remove a new DKIM public key",
					Alias:     []string{"rdkim"},
					Use:       "remove-dkim-pubkey [flags] dkim_pubkey",
					Example:   "remove-dkim-pubkey { domain: <domain>, selector: <selector> }",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "dkim_pubkey",
						},
					},
					Skip: true, // set to true if authority gated
				},
			},
		},
	}
}
