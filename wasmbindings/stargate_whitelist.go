package wasmbinding

import (
	"fmt"
	"sync"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	"github.com/cosmos/gogoproto/proto"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	feegranttypes "cosmossdk.io/x/feegrant"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	jwktypes "github.com/burnt-labs/xion/x/jwk/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
)

// stargateWhitelist keeps whitelist and its deterministic
// response binding for stargate queries.
// CONTRACT: since results of queries go into blocks, queries being added here should always be
// deterministic or can cause non-determinism in the state machine.
//
// The query can be multi-thread, so we have to use
// thread safe sync.Map.
var stargateWhitelist sync.Map

// Note: When adding a migration here, we should also add it to the Async ICQ params in the upgrade.
// In the future we may want to find a better way to keep these in sync

func init() {
	// ibc queries
	setWhitelistedQuery("/ibc.applications.transfer.v1.Query/DenomTrace", &ibctransfertypes.QueryDenomTraceResponse{})

	// cosmos-sdk queries

	// auth
	setWhitelistedQuery("/cosmos.auth.v1beta1.Query/Account", &authtypes.QueryAccountResponse{})
	setWhitelistedQuery("/cosmos.auth.v1beta1.Query/Params", &authtypes.QueryParamsResponse{})
	setWhitelistedQuery("/cosmos.auth.v1beta1.Query/ModuleAccounts", &authtypes.QueryModuleAccountsResponse{})

	// authz
	setWhitelistedQuery("/cosmos.authz.v1beta1.Query/Grants", &authztypes.QueryGrantsResponse{})

	// bank
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/Balance", &banktypes.QueryBalanceResponse{})
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/DenomMetadata", &banktypes.QueryDenomMetadataResponse{})
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/DenomsMetadata", &banktypes.QueryDenomsMetadataResponse{})
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/Params", &banktypes.QueryParamsResponse{})
	setWhitelistedQuery("/cosmos.bank.v1beta1.Query/SupplyOf", &banktypes.QuerySupplyOfResponse{})

	// distribution
	setWhitelistedQuery("/cosmos.distribution.v1beta1.Query/Params", &distributiontypes.QueryParamsResponse{})
	setWhitelistedQuery("/cosmos.distribution.v1beta1.Query/DelegatorWithdrawAddress", &distributiontypes.QueryDelegatorWithdrawAddressResponse{})
	setWhitelistedQuery("/cosmos.distribution.v1beta1.Query/ValidatorCommission", &distributiontypes.QueryValidatorCommissionResponse{})

	// feegrant
	setWhitelistedQuery("/cosmos.feegrant.v1beta1.Query/Allowance", &feegranttypes.QueryAllowanceResponse{})
	setWhitelistedQuery("/cosmos.feegrant.v1beta1.Query/AllowancesByGranter", &feegranttypes.QueryAllowancesByGranterResponse{})

	// gov
	setWhitelistedQuery("/cosmos.gov.v1beta1.Query/Deposit", &govtypesv1.QueryDepositResponse{})
	setWhitelistedQuery("/cosmos.gov.v1beta1.Query/Params", &govtypesv1.QueryParamsResponse{})
	setWhitelistedQuery("/cosmos.gov.v1beta1.Query/Vote", &govtypesv1.QueryVoteResponse{})

	// slashing
	setWhitelistedQuery("/cosmos.slashing.v1beta1.Query/Params", &slashingtypes.QueryParamsResponse{})
	setWhitelistedQuery("/cosmos.slashing.v1beta1.Query/SigningInfo", &slashingtypes.QuerySigningInfoResponse{})

	// staking
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Delegation", &stakingtypes.QueryDelegationResponse{})
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Params", &stakingtypes.QueryParamsResponse{})
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Validator", &stakingtypes.QueryValidatorResponse{})

	// xion queries
	setWhitelistedQuery("/xion.v1.Query/WebAuthNVerifyRegister", &xiontypes.QueryWebAuthNVerifyRegisterResponse{})
	setWhitelistedQuery("/xion.v1.Query/WebAuthNVerifyAuthenticate", &xiontypes.QueryWebAuthNVerifyAuthenticateResponse{})
	setWhitelistedQuery("/xion.jwk.v1.Query/AudienceAll", &jwktypes.QueryAllAudienceResponse{})
	setWhitelistedQuery("/xion.jwk.v1.Query/Audience", &jwktypes.QueryGetAudienceResponse{})
	setWhitelistedQuery("/xion.jwk.v1.Query/Params", &jwktypes.QueryParamsResponse{})
	setWhitelistedQuery("/xion.jwk.v1.Query/ValidateJWT", &jwktypes.QueryValidateJWTResponse{})
}

// GetWhitelistedQuery returns the whitelisted query at the provided path.
// If the query does not exist, or it was set up wrong by the chain, this returns an error.
func GetWhitelistedQuery(queryPath string) (proto.Message, error) {
	protoResponseAny, isWhitelisted := stargateWhitelist.Load(queryPath)
	if !isWhitelisted {
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", queryPath)}
	}
	protoResponseType, ok := protoResponseAny.(proto.Message)
	if !ok {
		return nil, wasmvmtypes.Unknown{}
	}
	return protoResponseType, nil
}

func setWhitelistedQuery(queryPath string, protoType proto.Message) {
	stargateWhitelist.Store(queryPath, protoType)
}
