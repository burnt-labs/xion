package authz

import (
	sdkauthz "github.com/cosmos/cosmos-sdk/x/authz"
)

// Type aliases for compatibility with cosmos-sdk
type (
	// Authorization represents the interface of various Authorization types.
	Authorization = sdkauthz.Authorization

	// AcceptResponse instruments the controller of an authz message.
	AcceptResponse = sdkauthz.AcceptResponse

	// Grant gives permissions to execute the provided method with expiration time.
	Grant = sdkauthz.Grant

	// GrantAuthorization extends a grant with both the addresses of the grantee and granter.
	GrantAuthorization = sdkauthz.GrantAuthorization

	// GrantQueueItem contains the list of TypeURL of a grant.
	GrantQueueItem = sdkauthz.GrantQueueItem

	// GenesisState defines the authz module's genesis state.
	GenesisState = sdkauthz.GenesisState

	// GenericAuthorization gives the grantee unrestricted permissions to execute.
	GenericAuthorization = sdkauthz.GenericAuthorization

	// MsgGrant is a request type for Grant method.
	MsgGrant = sdkauthz.MsgGrant

	// MsgExec attempts to execute the provided messages using authorizations.
	MsgExec = sdkauthz.MsgExec

	// MsgRevoke revokes any authorization with the provided sdk.Msg type.
	MsgRevoke = sdkauthz.MsgRevoke

	// MsgGrantResponse defines the Msg/Grant response type.
	MsgGrantResponse = sdkauthz.MsgGrantResponse

	// MsgExecResponse defines the Msg/Exec response type.
	MsgExecResponse = sdkauthz.MsgExecResponse

	// MsgRevokeResponse defines the Msg/Revoke response type.
	MsgRevokeResponse = sdkauthz.MsgRevokeResponse

	// QueryClient is the client API for Query service.
	QueryClient = sdkauthz.QueryClient

	// QueryServer is the server API for Query service.
	QueryServer = sdkauthz.QueryServer

	// MsgClient is the client API for Msg service.
	MsgClient = sdkauthz.MsgClient

	// MsgServer is the server API for Msg service.
	MsgServer = sdkauthz.MsgServer

	// QueryGrantsRequest is the request type for the Query/Grants RPC method.
	QueryGrantsRequest = sdkauthz.QueryGrantsRequest

	// QueryGrantsResponse is the response type for the Query/Grants RPC method.
	QueryGrantsResponse = sdkauthz.QueryGrantsResponse

	// QueryGranterGrantsRequest is the request type for the Query/GranterGrants RPC method.
	QueryGranterGrantsRequest = sdkauthz.QueryGranterGrantsRequest

	// QueryGranterGrantsResponse is the response type for the Query/GranterGrants RPC method.
	QueryGranterGrantsResponse = sdkauthz.QueryGranterGrantsResponse

	// QueryGranteeGrantsRequest is the request type for the Query/GranteeGrants RPC method.
	QueryGranteeGrantsRequest = sdkauthz.QueryGranteeGrantsRequest

	// QueryGranteeGrantsResponse is the response type for the Query/GranteeGrants RPC method.
	QueryGranteeGrantsResponse = sdkauthz.QueryGranteeGrantsResponse

	// EventGrant is emitted on MsgGrant.
	EventGrant = sdkauthz.EventGrant

	// EventRevoke is emitted on MsgRevoke.
	EventRevoke = sdkauthz.EventRevoke
)

// Variable aliases for compatibility with cosmos-sdk
var (
	// Errors
	ErrNoAuthorizationFound      = sdkauthz.ErrNoAuthorizationFound
	ErrInvalidExpirationTime     = sdkauthz.ErrInvalidExpirationTime
	ErrUnknownAuthorizationType  = sdkauthz.ErrUnknownAuthorizationType
	ErrNoGrantKeyFound           = sdkauthz.ErrNoGrantKeyFound
	ErrAuthorizationExpired      = sdkauthz.ErrAuthorizationExpired
	ErrGranteeIsGranter          = sdkauthz.ErrGranteeIsGranter
	ErrAuthorizationNumOfSigners = sdkauthz.ErrAuthorizationNumOfSigners
	ErrNegativeMaxTokens         = sdkauthz.ErrNegativeMaxTokens

	// Codec registration
	RegisterLegacyAminoCodec = sdkauthz.RegisterLegacyAminoCodec
	RegisterInterfaces       = sdkauthz.RegisterInterfaces

	// Genesis
	NewGenesisState     = sdkauthz.NewGenesisState
	DefaultGenesisState = sdkauthz.DefaultGenesisState
	ValidateGenesis     = sdkauthz.ValidateGenesis

	// Constructors
	NewGrant                = sdkauthz.NewGrant
	NewGenericAuthorization = sdkauthz.NewGenericAuthorization
	NewMsgGrant             = sdkauthz.NewMsgGrant
	NewMsgRevoke            = sdkauthz.NewMsgRevoke
	NewMsgExec              = sdkauthz.NewMsgExec

	// Query service registration
	RegisterQueryServer        = sdkauthz.RegisterQueryServer
	RegisterQueryHandlerClient = sdkauthz.RegisterQueryHandlerClient
	NewQueryClient             = sdkauthz.NewQueryClient

	// Msg service registration
	RegisterMsgServer = sdkauthz.RegisterMsgServer
	MsgServiceDesc    = sdkauthz.MsgServiceDesc
)
