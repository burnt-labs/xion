package mint

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/burnt-labs/xion/x/mint/types"
	minttypes "github.com/burnt-labs/xion/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
)

func TestBeginBlocker(t *testing.T) {
	type expected struct {
		annualProvisions sdk.Dec
		bondedRatio      sdk.Dec
		burnedAmount     uint64
		collectedAmount  uint64
		inflation        sdk.Dec
		minted           uint64
		needed           uint64
	}
	type parameters struct {
		bonded        sdk.Int
		bondedRatio   sdk.Dec
		fees          sdk.Coins
		collectedFees sdk.Coin
		burn          bool
		mint          bool
	}

	stakingTokenSupply := sdk.NewIntFromUint64(100000000000)

	tt := []struct {
		name       string
		parameters parameters
		expected   expected
	}{
		{
			name: "full bonded tokens",
			parameters: parameters{
				bonded:        stakingTokenSupply,
				bondedRatio:   sdk.NewDecWithPrec(1, 4),
				fees:          sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(109))),
				collectedFees: sdk.NewCoin("stake", sdk.NewInt(1000)),
				mint:          true,
				burn:          false,
			},
			expected: expected{
				annualProvisions: sdk.NewDecWithPrec(7000000000, 0),
				bondedRatio:      sdk.NewDecWithPrec(1, 4),
				burnedAmount:     0,
				collectedAmount:  1000,
				inflation:        sdk.NewDecWithPrec(7, 2),
				minted:           109,
				needed:           1109,
			},
		},
		{
			name: "less than ideal bonded tokens",
			parameters: parameters{
				bonded:        sdk.NewInt(int64(100000000000 * 0.33)),
				bondedRatio:   sdk.NewDecWithPrec(33, 2),
				fees:          sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1045))),
				collectedFees: sdk.NewCoin("stake", sdk.NewInt(0)),
				mint:          true,
				burn:          false,
			},
			expected: expected{
				annualProvisions: sdk.NewDec(6600000000),
				bondedRatio:      sdk.NewDecWithPrec(33, 2),
				burnedAmount:     0,
				collectedAmount:  0,
				inflation:        sdk.NewDecWithPrec(20, 2),
				minted:           1045,
				needed:           1045,
			},
		},
		{
			name: "above staking threshold, fee collector has values",
			parameters: parameters{
				bonded:        stakingTokenSupply,
				bondedRatio:   sdk.NewDecWithPrec(1, 4),
				fees:          sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(109))),
				collectedFees: sdk.NewCoin("stake", sdk.NewInt(10000)),
				mint:          false,
				burn:          true,
			},
			expected: expected{
				annualProvisions: sdk.NewDecWithPrec(7000000000, 0),
				bondedRatio:      sdk.NewDecWithPrec(1, 4),
				burnedAmount:     8891,
				collectedAmount:  10000,
				inflation:        sdk.NewDecWithPrec(7, 2),
				minted:           0,
				needed:           1109,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			testcontext, keeper, mocks := createTestBaseKeeperAndContextWithMocks(t)
			ctx := testcontext.Ctx
			stakingKeeper := mocks.MockStakingKeeper
			bankKeeper := mocks.MockBankKeeper

			/*
				Populate mock
			*/

			keeper.SetMinter(ctx, minttypes.InitialMinter(tc.parameters.bondedRatio))

			stakingKeeper.EXPECT().TotalBondedTokens(ctx).Return(tc.parameters.bonded)
			stakingKeeper.EXPECT().BondedRatio(ctx).Return(tc.parameters.bondedRatio)
			bankKeeper.EXPECT().GetBalance(ctx, mocks.moduleAccount.GetAddress(), "stake").Return(tc.parameters.collectedFees)

			if tc.parameters.mint {
				bankKeeper.EXPECT().MintCoins(ctx, minttypes.ModuleName, tc.parameters.fees).Return(nil)
				bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName, tc.parameters.fees).Return(nil)
			}

			if tc.parameters.burn {
				c := sdk.NewCoin("stake", sdk.NewInt(int64(tc.expected.needed)))
				bankKeeper.EXPECT().BurnCoins(ctx, authtypes.FeeCollectorName, sdk.NewCoins(tc.parameters.collectedFees.Sub(c))).Return(nil)
			}

			BeginBlocker(ctx, *keeper, minttypes.DefaultInflationCalculationFn)

			events := ctx.EventManager().Events()
			assert.Equalf(t, 1, len(events), "A single event must be emitted. However %d events were emitted", len(events))
			event := events[0]
			assert.Equalf(t, "xion.mint.v1.MintIncentiveTokens", event.Type, "Expected event to be xion.mint.v1.MintIncentiveTokens but found: %s", event.Type)
			assert.Equalf(t, 7, len(event.Attributes), "Expcted 7 attributes but found %d", len(event.Attributes))

			assert.Equal(t, "annual_provisions", event.Attributes[0].Key)
			assert.Equal(t, tc.expected.annualProvisions, sdk.MustNewDecFromStr(stripValue(t, event.Attributes[0].Value)))

			assert.Equal(t, "bonded_ratio", event.Attributes[1].Key)
			assert.Equal(t, tc.expected.bondedRatio, sdk.MustNewDecFromStr(stripValue(t, event.Attributes[1].Value)))

			assert.Equal(t, "burned_amount", event.Attributes[2].Key)
			assert.Equal(t, tc.expected.burnedAmount, stringToU64(t, event.Attributes[2].Value))

			assert.Equal(t, "collected_amount", event.Attributes[3].Key)
			assert.Equal(t, tc.expected.collectedAmount, stringToU64(t, event.Attributes[3].Value))

			assert.Equal(t, "inflation", event.Attributes[4].Key)
			assert.Equal(t, tc.expected.inflation, sdk.MustNewDecFromStr(stripValue(t, event.Attributes[4].Value)))

			assert.Equal(t, "minted_amount", event.Attributes[5].Key)
			assert.Equal(t, tc.expected.minted, stringToU64(t, event.Attributes[5].Value))

			assert.Equal(t, "needed_amount", event.Attributes[6].Key)
			assert.Equal(t, tc.expected.needed, stringToU64(t, event.Attributes[6].Value))
		})
	}
}
func stripValue(t *testing.T, s string) string {
	stripped := strings.Replace(s, "\\", "", -1)
	return strings.Replace(stripped, "\"", "", -1)
}

func stringToU64(t *testing.T, s string) uint64 {
	stripped := stripValue(t, s)
	ui64, err := strconv.ParseUint(stripped, 10, 64)
	if err != nil {
		fmt.Println(err)
		t.FailNow() // Could not convert
	}
	return ui64
}
