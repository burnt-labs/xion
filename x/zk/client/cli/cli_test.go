// client/cli/cli_test.go
package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/zk/client/cli"
	"github.com/burnt-labs/xion/x/zk/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	// DefaultConfig now requires an app constructor function
	cfg := network.DefaultConfig(app.NewTestNetworkFixture)
	cfg.NumValidators = 1

	s.cfg = cfg

	net, err := network.New(s.T(), s.T().TempDir(), cfg)
	s.Require().NoError(err)
	s.network = net

	s.Require().NotNil(s.network)

	_, err = s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

// Helper to create a temporary vkey file
func (s *IntegrationTestSuite) createTempVKeyFile() string {
	vkeyJSON := `{
		"protocol": "groth16",
		"curve": "bn128",
		"nPublic": 2,
		"vk_alpha_1": [
			"20491192805390485299153009773594534940189261866228447918068658471970481763042",
			"9383485363053290200918347156157836566562967994039712273449902621266178545958",
			"1"
		],
		"vk_beta_2": [
			["6375614351688725206403948262868962793625744043794305715222011528459656738731", "4252822878758300859123897981450591353533073413197771768651442665752259397132"],
			["10505242626370262277552901082094356697409835680220590971873171140371331206856", "21847035105528745403288232691147584728191162732299865338377159692350059136679"],
			["1", "0"]
		],
		"vk_gamma_2": [
			["10857046999023057135944570762232829481370756359578518086990519993285655852781", "11559732032986387107991004021392285783925812861821192530917403151452391805634"],
			["8495653923123431417604973247489272438418190587263600148770280649306958101930", "4082367875863433681332203403145435568316851327593401208105741076214120093531"],
			["1", "0"]
		],
		"vk_delta_2": [
			["7408543996799841808823674318962923691422846694508104677211507255777183761346", "17378314708652486082434193052153411074104970941065581812653446685054220492752"],
			["20934765493363178521480199624017210946632719146191129233788277268880988392769", "9933248257943163684434361179172132751107201169345727211797322171844177096469"],
			["1", "0"]
		],
		"IC": [
			["5449013234494434531196202102845211237542489505716355090765771488165044993949", "4910919431725277797191489997138444712176878647014509270723700672161925471159", "1"],
			["12345678901234567890123456789012345678901234567890123456789012345678901234567", "98765432109876543210987654321098765432109876543210987654321098765432109876543", "1"],
			["11111111111111111111111111111111111111111111111111111111111111111111111111111", "22222222222222222222222222222222222222222222222222222222222222222222222222222", "1"]
		]
	}`

	tmpFile := filepath.Join(s.T().TempDir(), "test_vkey.json")
	err := os.WriteFile(tmpFile, []byte(vkeyJSON), 0o600)
	s.Require().NoError(err)

	return tmpFile
}

func (s *IntegrationTestSuite) TestQueryVKeysCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "query all vkeys",
			args: []string{
				fmt.Sprintf("--%s=json", flags.FlagOutput),
			},
			expectErr: false,
		},
		{
			name: "query with pagination",
			args: []string{
				fmt.Sprintf("--%s=5", flags.FlagLimit),
				fmt.Sprintf("--%s=json", flags.FlagOutput),
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryVKeys()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				var resp types.QueryVKeysResponse
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &resp))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestQueryHasVKeyCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "check non-existent vkey",
			args: []string{
				"non_existent_key",
				fmt.Sprintf("--%s=json", flags.FlagOutput),
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryHasVKey()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				var resp types.QueryHasVKeyResponse
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &resp))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestQueryVKeyByIDCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "query non-existent vkey by ID",
			args: []string{
				"9999",
				fmt.Sprintf("--%s=json", flags.FlagOutput),
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryVKey()
			clientCtx := val.ClientCtx

			_, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestQueryVKeyByNameCmd() {
	val := s.network.Validators[0]

	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "query non-existent vkey by name",
			args: []string{
				"non_existent",
				fmt.Sprintf("--%s=json", flags.FlagOutput),
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryVKeyByName()
			clientCtx := val.ClientCtx

			_, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestAddVKeyCmd() {
	val := s.network.Validators[0]

	vkeyFile := s.createTempVKeyFile()
	defer os.Remove(vkeyFile)

	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "invalid vkey file path",
			args: []string{
				"test_key",
				"/non/existent/file.json",
				"Test description",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdkmath.NewInt(10))).String()),
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.GetCmdAddVKey()
			clientCtx := val.ClientCtx

			_, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
