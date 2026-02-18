package wasmbinding

import (
	"fmt"
	"sync"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"

	"github.com/cosmos/gogoproto/proto"

	feegranttypes "cosmossdk.io/x/feegrant"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	dkimtypes "github.com/burnt-labs/xion/x/dkim/types"
	jwktypes "github.com/burnt-labs/xion/x/jwk/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	zktypes "github.com/burnt-labs/xion/x/zk/types"
)

// stargateWhitelist keeps whitelist and its deterministic
// response binding for stargate queries.
// CONTRACT: since results of queries go into blocks, queries being added here should always be
// deterministic or can cause non-determinism in the state machine.
//
// The whitelist stores factory functions that return new proto.Message instances
// to ensure thread-safety. Each query gets its own instance to avoid race conditions
// when concurrent queries unmarshal into the same object.
var stargateWhitelist sync.Map

// Note: When adding a migration here, we should also add it to the Async ICQ params in the upgrade.
// In the future we may want to find a better way to keep these in sync

func init() {
	// cosmos-sdk queries

	// auth
	setWhitelistedQuery("/cosmos.auth.v1beta1.Query/Account", func() proto.Message { return &authtypes.QueryAccountResponse{} })
	setWhitelistedQuery("/cosmos.auth.v1beta1.Query/Params", func() proto.Message { return &authtypes.QueryParamsResponse{} })
	setWhitelistedQuery("/cosmos.auth.v1beta1.Query/ModuleAccounts", func() proto.Message { return &authtypes.QueryModuleAccountsResponse{} })

	// authz
	setWhitelistedQuery("/cosmos.authz.v1beta1.Query/Grants", func() proto.Message { return &authztypes.QueryGrantsResponse{} })

	// bank
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/Balance", func() proto.Message { return &banktypes.QueryBalanceResponse{} })
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/DenomMetadata", func() proto.Message { return &banktypes.QueryDenomMetadataResponse{} })
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/DenomsMetadata", func() proto.Message { return &banktypes.QueryDenomsMetadataResponse{} })
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/Params", func() proto.Message { return &banktypes.QueryParamsResponse{} })
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/SupplyOf", func() proto.Message { return &banktypes.QuerySupplyOfResponse{} })

	// distribution
	setWhitelistedQuery("/cosmos.distribution.v1beta1.Query/Params", func() proto.Message { return &distributiontypes.QueryParamsResponse{} })
	setWhitelistedQuery("/cosmos.distribution.v1beta1.Query/DelegatorWithdrawAddress", func() proto.Message { return &distributiontypes.QueryDelegatorWithdrawAddressResponse{} })
	setWhitelistedQuery("/cosmos.distribution.v1beta1.Query/ValidatorCommission", func() proto.Message { return &distributiontypes.QueryValidatorCommissionResponse{} })

	// feegrant
	setWhitelistedQuery("/cosmos.feegrant.v1beta1.Query/Allowance", func() proto.Message { return &feegranttypes.QueryAllowanceResponse{} })
	setWhitelistedQuery("/cosmos.feegrant.v1beta1.Query/AllowancesByGranter", func() proto.Message { return &feegranttypes.QueryAllowancesByGranterResponse{} })

	// gov
	setWhitelistedQuery("/cosmos.gov.v1beta1.Query/Deposit", func() proto.Message { return &govtypesv1.QueryDepositResponse{} })
	setWhitelistedQuery("/cosmos.gov.v1beta1.Query/Params", func() proto.Message { return &govtypesv1.QueryParamsResponse{} })
	setWhitelistedQuery("/cosmos.gov.v1beta1.Query/Vote", func() proto.Message { return &govtypesv1.QueryVoteResponse{} })

	// slashing
	setWhitelistedQuery("/cosmos.slashing.v1beta1.Query/Params", func() proto.Message { return &slashingtypes.QueryParamsResponse{} })
	setWhitelistedQuery("/cosmos.slashing.v1beta1.Query/SigningInfo", func() proto.Message { return &slashingtypes.QuerySigningInfoResponse{} })

	// staking
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Delegation", func() proto.Message { return &stakingtypes.QueryDelegationResponse{} })
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Params", func() proto.Message { return &stakingtypes.QueryParamsResponse{} })
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Validator", func() proto.Message { return &stakingtypes.QueryValidatorResponse{} })

	// xion queries
	setWhitelistedQuery("/xion.v1.Query/WebAuthNVerifyRegister", func() proto.Message { return &xiontypes.QueryWebAuthNVerifyRegisterResponse{} })
	setWhitelistedQuery("/xion.v1.Query/WebAuthNVerifyAuthenticate", func() proto.Message { return &xiontypes.QueryWebAuthNVerifyAuthenticateResponse{} })
	setWhitelistedQuery("/xion.jwk.v1.Query/AudienceAll", func() proto.Message { return &jwktypes.QueryAllAudienceResponse{} })
	setWhitelistedQuery("/xion.jwk.v1.Query/Audience", func() proto.Message { return &jwktypes.QueryGetAudienceResponse{} })
	setWhitelistedQuery("/xion.jwk.v1.Query/Params", func() proto.Message { return &jwktypes.QueryParamsResponse{} })
	setWhitelistedQuery("/xion.jwk.v1.Query/ValidateJWT", func() proto.Message { return &jwktypes.QueryValidateJWTResponse{} })
	setWhitelistedQuery("/xion.dkim.v1.Query/DkimPubKeys", func() proto.Message { return &dkimtypes.QueryDkimPubKeysResponse{} })
	setWhitelistedQuery("/xion.dkim.v1.Query/Params", func() proto.Message { return &dkimtypes.QueryParamsResponse{} })
	setWhitelistedQuery("/xion.dkim.v1.Query/DkimPubKey", func() proto.Message { return &dkimtypes.QueryDkimPubKeyResponse{} })
	setWhitelistedQuery("/xion.dkim.v1.Query/Authenticate", func() proto.Message { return &dkimtypes.AuthenticateResponse{} })
	setWhitelistedQuery("/xion.zk.v1.Query/ProofVerify", func() proto.Message { return &zktypes.ProofVerifyResponse{} })
}

// ProtoMessageFactory is a function that creates a new proto.Message instance.
// This ensures thread-safety by giving each caller their own instance.
type ProtoMessageFactory func() proto.Message

// GetWhitelistedQuery returns a new proto.Message instance for the whitelisted query at the provided path.
// If the query does not exist, or it was set up wrong by the chain, this returns an error.
// Each call returns a fresh instance to ensure thread-safety.
func GetWhitelistedQuery(queryPath string) (proto.Message, error) {
	factoryAny, isWhitelisted := stargateWhitelist.Load(queryPath)
	if !isWhitelisted {
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", queryPath)}
	}
	factory, ok := factoryAny.(ProtoMessageFactory)
	if !ok {
		return nil, wasmvmtypes.Unknown{}
	}
	return factory(), nil
}

func setWhitelistedQuery(queryPath string, factory ProtoMessageFactory) {
	stargateWhitelist.Store(queryPath, factory)
}
