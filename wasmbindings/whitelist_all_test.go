package wasmbinding

import (
	"testing"
)

// TestAllWhitelistedPathsEnsured iterates every path we add in init() and ensures
// GetWhitelistedQuery returns a non-nil proto.Message without error.
func TestAllWhitelistedPathsEnsured(t *testing.T) {
	paths := []string{
		"/cosmos.auth.v1beta1.Query/Account",
		"/cosmos.auth.v1beta1.Query/Params",
		"/cosmos.auth.v1beta1.Query/ModuleAccounts",
		"/cosmos.authz.v1beta1.Query/Grants",
		"/cosmos.bank.v1beta1.Query/Balance",
		"/cosmos.bank.v1beta1.Query/DenomMetadata",
		"/cosmos.bank.v1beta1.Query/DenomsMetadata",
		"/cosmos.bank.v1beta1.Query/Params",
		"/cosmos.bank.v1beta1.Query/SupplyOf",
		"/cosmos.distribution.v1beta1.Query/Params",
		"/cosmos.distribution.v1beta1.Query/DelegatorWithdrawAddress",
		"/cosmos.distribution.v1beta1.Query/ValidatorCommission",
		"/cosmos.feegrant.v1beta1.Query/Allowance",
		"/cosmos.feegrant.v1beta1.Query/AllowancesByGranter",
		"/cosmos.gov.v1beta1.Query/Deposit",
		"/cosmos.gov.v1beta1.Query/Params",
		"/cosmos.gov.v1beta1.Query/Vote",
		"/cosmos.slashing.v1beta1.Query/Params",
		"/cosmos.slashing.v1beta1.Query/SigningInfo",
		"/cosmos.staking.v1beta1.Query/Delegation",
		"/cosmos.staking.v1beta1.Query/Params",
		"/cosmos.staking.v1beta1.Query/Validator",
		"/xion.v1.Query/WebAuthNVerifyRegister",
		"/xion.v1.Query/WebAuthNVerifyAuthenticate",
		"/xion.jwk.v1.Query/AudienceAll",
		"/xion.jwk.v1.Query/Audience",
		"/xion.jwk.v1.Query/Params",
		"/xion.jwk.v1.Query/ValidateJWT",
	}

	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		if _, dup := seen[p]; dup {
			// guard against accidental duplicates in test slice
			continue
		}
		seen[p] = struct{}{}
		msg, err := GetWhitelistedQuery(p)
		if err != nil {
			// fail fast with specific path context
			t.Fatalf("expected path %s to be whitelisted, got err: %v", p, err)
		}
		if msg == nil {
			t.Fatalf("expected non-nil proto message for path %s", p)
		}
	}

	if len(seen) != len(paths) {
		t.Fatalf("deduplicated path count mismatch: expected %d got %d", len(paths), len(seen))
	}
}
