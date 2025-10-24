// types/genesis_test.go
package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/zk/types"
)

// createValidVKeyBytes creates valid verification key bytes for testing
func createValidVKeyBytes() []byte {
	vkeyJSON := map[string]interface{}{
		"protocol": "groth16",
		"curve":    "bn128",
		"nPublic":  2,
		"vk_alpha_1": []string{
			"20491192805390485299153009773594534940189261866228447918068658471970481763042",
			"9383485363053290200918347156157836566562967994039712273449902621266178545958",
			"1",
		},
		"vk_beta_2": [][]string{
			{"6375614351688725206403948262868962793625744043794305715222011528459656738731", "4252822878758300859123897981450591353533073413197771768651442665752259397132"},
			{"10505242626370262277552901082094356697409835680220590971873171140371331206856", "21847035105528745403288232691147584728191162732299865338377159692350059136679"},
			{"1", "0"},
		},
		"vk_gamma_2": [][]string{
			{"10857046999023057135944570762232829481370756359578518086990519993285655852781", "11559732032986387107991004021392285783925812861821192530917403151452391805634"},
			{"8495653923123431417604973247489272438418190587263600148770280649306958101930", "4082367875863433681332203403145435568316851327593401208105741076214120093531"},
			{"1", "0"},
		},
		"vk_delta_2": [][]string{
			{"7408543996799841808823674318962923691422846694508104677211507255777183761346", "17378314708652486082434193052153411074104970941065581812653446685054220492752"},
			{"20934765493363178521480199624017210946632719146191129233788277268880988392769", "9933248257943163684434361179172132751107201169345727211797322171844177096469"},
			{"1", "0"},
		},
		"IC": [][]string{
			{"5449013234494434531196202102845211237542489505716355090765771488165044993949", "4910919431725277797191489997138444712176878647014509270723700672161925471159", "1"},
			{"12345678901234567890123456789012345678901234567890123456789012345678901234567", "98765432109876543210987654321098765432109876543210987654321098765432109876543", "1"},
			{"11111111111111111111111111111111111111111111111111111111111111111111111111111", "22222222222222222222222222222222222222222222222222222222222222222222222222222", "1"},
		},
	}

	bytes, _ := json.Marshal(vkeyJSON)
	return bytes
}

func TestDefaultGenesisState(t *testing.T) {
	gs := types.DefaultGenesisState()
	require.NotNil(t, gs)
	require.Empty(t, gs.Vkeys)

	// Default genesis should be valid
	err := gs.Validate()
	require.NoError(t, err)
}

func TestNewGenesisState(t *testing.T) {
	vkeyBytes := createValidVKeyBytes()

	vkeys := []types.VKeyWithID{
		{
			Id: 0,
			Vkey: types.VKey{
				KeyBytes:    vkeyBytes,
				Name:        "test_key",
				Description: "Test key",
			},
		},
	}

	gs := types.NewGenesisState(vkeys)
	require.NotNil(t, gs)
	require.Len(t, gs.Vkeys, 1)
	require.Equal(t, vkeys[0].Id, gs.Vkeys[0].Id)
	require.Equal(t, vkeys[0].Vkey.Name, gs.Vkeys[0].Vkey.Name)
}

func TestGenesisStateValidate(t *testing.T) {
	validVKeyBytes := createValidVKeyBytes()
	fmt.Println(validVKeyBytes)

	tests := []struct {
		name        string
		gs          *types.GenesisState
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid empty genesis",
			gs:          types.DefaultGenesisState(),
			expectError: false,
		},
		{
			name: "valid single vkey",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key1",
							Description: "Key 1",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid multiple vkeys",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key1",
							Description: "Key 1",
						},
					},
					{
						Id: 1,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key2",
							Description: "Key 2",
						},
					},
					{
						Id: 2,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key3",
							Description: "Key 3",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "duplicate vkey IDs",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key1",
							Description: "Key 1",
						},
					},
					{
						Id: 0, // Duplicate ID
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key2",
							Description: "Key 2",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate vkey ID",
		},
		{
			name: "duplicate vkey names",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "duplicate_name",
							Description: "Key 1",
						},
					},
					{
						Id: 1,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "duplicate_name", // Duplicate name
							Description: "Key 2",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate vkey name",
		},
		{
			name: "empty vkey name",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "", // Empty name
							Description: "Key 1",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "has empty name",
		},
		{
			name: "empty key bytes",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    []byte{}, // Empty key bytes
							Name:        "key1",
							Description: "Key 1",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "has empty key_bytes",
		},
		{
			name: "invalid key bytes - not json",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    []byte("not valid json"),
							Name:        "key1",
							Description: "Key 1",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "has invalid key_bytes",
		}, /*
			/*NOTE: WE need a way to validate vkeys
			{
				name: "invalid key bytes - missing required fields",
				gs: &types.GenesisState{
					Vkeys: []types.VKeyWithID{
						{
							Id: 0,
							Vkey: types.VKey{
								KeyBytes: []byte(`{
									"protocol": "groth16",
									"curve": "bn128"
								}`),
								Name:        "key1",
								Description: "Key 1",
							},
						},
					},
				},
				expectError: true,
				errorMsg:    "has invalid key_bytes",
			},
			{
				name: "invalid key bytes - wrong protocol",
				gs: &types.GenesisState{
					Vkeys: []types.VKeyWithID{
						{
							Id: 0,
							Vkey: types.VKey{
								KeyBytes: []byte(`{
									"protocol": "plonk",
									"curve": "bn128",
									"nPublic": 2,
									"vk_alpha_1": ["1", "2", "1"],
									"vk_beta_2": [["3", "4"], ["5", "6"], ["1", "0"]],
									"vk_gamma_2": [["7", "8"], ["9", "10"], ["1", "0"]],
									"vk_delta_2": [["11", "12"], ["13", "14"], ["1", "0"]],
									"IC": [["15", "16", "1"], ["17", "18", "1"], ["19", "20", "1"]]
								}`),
								Name:        "key1",
								Description: "Key 1",
							},
						},
					},
				},
				expectError: true,
				errorMsg:    "has invalid key_bytes",
			},*/
		{
			name: "non-sequential IDs are valid",
			gs: &types.GenesisState{
				Vkeys: []types.VKeyWithID{
					{
						Id: 0,
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key1",
							Description: "Key 1",
						},
					},
					{
						Id: 5, // Non-sequential but unique
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key2",
							Description: "Key 2",
						},
					},
					{
						Id: 10, // Non-sequential but unique
						Vkey: types.VKey{
							KeyBytes:    validVKeyBytes,
							Name:        "key3",
							Description: "Key 3",
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gs.Validate()

			if tt.expectError {
				require.Error(t, err, "Expected error for test: %s", tt.name)
				require.Contains(t, err.Error(), tt.errorMsg, "Error message mismatch for test: %s", tt.name)
			} else {
				require.NoError(t, err, "Unexpected error for test: %s", tt.name)
			}
		})
	}
}

func TestGenesisStateValidateComplex(t *testing.T) {
	validVKeyBytes := createValidVKeyBytes()

	// Test with 100 vkeys to ensure performance and correctness at scale
	t.Run("validate large genesis state", func(t *testing.T) {
		vkeys := make([]types.VKeyWithID, 100)
		for i := 0; i < 100; i++ {
			vkeys[i] = types.VKeyWithID{
				Id: uint64(i),
				Vkey: types.VKey{
					KeyBytes:    validVKeyBytes,
					Name:        string(rune('a'+i%26)) + string(rune('0'+i/26)), // Generate unique names
					Description: "Key " + string(rune('0'+i)),
				},
			}
		}

		gs := &types.GenesisState{Vkeys: vkeys}
		err := gs.Validate()
		require.NoError(t, err)
	})

	// Test duplicate detection in large set
	t.Run("detect duplicate in large set", func(t *testing.T) {
		vkeys := make([]types.VKeyWithID, 50)
		for i := 0; i < 50; i++ {
			vkeys[i] = types.VKeyWithID{
				Id: uint64(i),
				Vkey: types.VKey{
					KeyBytes:    validVKeyBytes,
					Name:        string(rune('a' + i)),
					Description: "Key " + string(rune('0'+i)),
				},
			}
		}

		// Add duplicate ID in the middle
		vkeys[25].Id = 10 // Duplicate of vkeys[10]

		gs := &types.GenesisState{Vkeys: vkeys}
		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate vkey ID")
	})
}

func TestGenesisStateJSON(t *testing.T) {
	validVKeyBytes := createValidVKeyBytes()

	gs := &types.GenesisState{
		Vkeys: []types.VKeyWithID{
			{
				Id: 0,
				Vkey: types.VKey{
					KeyBytes:    validVKeyBytes,
					Name:        "test_key",
					Description: "Test key",
				},
			},
		},
	}

	// Test marshaling
	jsonBytes, err := json.Marshal(gs)
	require.NoError(t, err)
	require.NotEmpty(t, jsonBytes)

	// Test unmarshaling
	var unmarshaled types.GenesisState
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	require.Len(t, unmarshaled.Vkeys, 1)
	require.Equal(t, gs.Vkeys[0].Id, unmarshaled.Vkeys[0].Id)
	require.Equal(t, gs.Vkeys[0].Vkey.Name, unmarshaled.Vkeys[0].Vkey.Name)
	require.Equal(t, gs.Vkeys[0].Vkey.KeyBytes, unmarshaled.Vkeys[0].Vkey.KeyBytes)

	// Validate unmarshaled genesis
	err = unmarshaled.Validate()
	require.NoError(t, err)
}
