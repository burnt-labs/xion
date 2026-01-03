package types_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/zk/types"
)

// validVKeyJSON is a valid groth16 verification key for testing
var validVKeyJSON = []byte(`{"protocol":"groth16","curve":"bn128","nPublic":2,"vk_alpha_1":["20491192805390485299153009773594534940189261866228447918068658471970481763042","9383485363053290200918347156157836566562967994039712273449902621266178545958","1"],"vk_beta_2":[["6375614351688725206403948262868962793625744043794305715222011528459656738731","4252822878758300859123897981450591353533073413197771768651442665752259397132"],["10505242626370262277552901082094356697409835680220590971873171140371331206856","21847035105528745403288232691147584728191162732299865338377159692350059136679"],["1","0"]],"vk_gamma_2":[["10857046999023057135944570762232829481370756359578518086990519993285655852781","11559732032986387107991004021392285783925812861821192530917403151452391805634"],["8495653923123431417604973247489272438418190587263600148770280649306958101930","4082367875863433681332203403145435568316851327593401208105741076214120093531"],["1","0"]],"vk_delta_2":[["15077028419523802218068800711765892220007704101776825737873498462523243974011","21035432942633023563568649328632676616806345190265721806958729811352423617078"],["19557127063776667950420308021890134701214219557891223581756933481287479566633","846877808733141898933269498060926622823200991369227170080032302022497893186"],["1","0"]],"IC":[["15862126421713956100993553801681385807071251923202096057307112802014733741378","13108645471998862676498660499963950655089157405079199301326521022095491980050","1"],["21715368734116209877917737472988550416427108747767751044339655343738713294081","16899680151032020137316253115554651362124948058456073805307693594940894296737","1"],["15262990638665614060530787821677588373695066693663158063982759218486893877667","2420030994339659044877551774878879579539478107604084928775920288418445748893","1"]]}`)
var validVKeyBase64 = base64.StdEncoding.EncodeToString(validVKeyJSON)

func TestValidateVKeyBytes(t *testing.T) {
	t.Run("valid base64 vkey bytes", func(t *testing.T) {
		err := types.ValidateVKeyBytes([]byte(validVKeyBase64), types.DefaultMaxVKeySizeBytes)
		require.NoError(t, err)
	})

	t.Run("base64 vkey bytes with whitespace rejected", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString(validVKeyJSON) + "\n"
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "whitespace")
	})

	t.Run("base64 vkey bytes too large", func(t *testing.T) {
		tooLarge := make([]byte, int(types.DefaultMaxVKeySizeBytes)+1)
		encoded := base64.StdEncoding.EncodeToString(tooLarge)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrVKeyTooLarge)
	})

	t.Run("empty vkey bytes", func(t *testing.T) {
		err := types.ValidateVKeyBytes([]byte{}, types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty vkey data")
	})

	t.Run("nil vkey bytes", func(t *testing.T) {
		err := types.ValidateVKeyBytes(nil, types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty vkey data")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("{invalid json"))
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid verification key JSON")
	})

	t.Run("unsupported protocol", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "plonk",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported protocol")
	})

	t.Run("invalid nPublic zero", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    0,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid nPublic")
	})

	t.Run("invalid nPublic negative", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    -1,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid nPublic")
	})

	t.Run("invalid VkAlpha1 too few coordinates", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid VkAlpha1")
	})

	t.Run("invalid VkBeta2 too few rows", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid VkBeta2")
	})

	t.Run("invalid VkBeta2 wrong column count", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2", "3"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes )
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid VkBeta2")
	})

	t.Run("invalid VkGamma2 too few rows", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid VkGamma2")
	})

	t.Run("invalid VkGamma2 wrong column count", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid VkGamma2")
	})

	t.Run("invalid VkDelta2 too few rows", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid VkDelta2")
	})

	t.Run("invalid VkDelta2 wrong column count", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid VkDelta2")
	})

	t.Run("invalid IC length mismatch", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1", "2", "1"}, {"3", "4", "1"}}, // Should be 3 (nPublic + 1)
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid IC length")
	})

	t.Run("invalid IC point too few coordinates", func(t *testing.T) {
		vkey := map[string]interface{}{
			"protocol":   "groth16",
			"curve":      "bn128",
			"nPublic":    2,
			"vk_alpha_1": []string{"1", "2", "1"},
			"vk_beta_2":  [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_gamma_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"vk_delta_2": [][]string{{"1", "2"}, {"3", "4"}, {"1", "0"}},
			"IC":         [][]string{{"1"}, {"3", "4", "1"}, {"5", "6", "1"}},
		}
		data, _ := json.Marshal(vkey)
		encoded := base64.StdEncoding.EncodeToString(data)
		err := types.ValidateVKeyBytes([]byte(encoded), types.DefaultMaxVKeySizeBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid IC")
	})
}

func TestUnmarshalVKey(t *testing.T) {
	t.Run("valid vkey", func(t *testing.T) {
		vkey := &types.VKey{
			KeyBytes:    validVKeyJSON,
			Name:        "test-vkey",
			Description: "test description",
		}
		result, err := types.UnmarshalVKey(vkey)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "groth16", result.Protocol)
		require.Equal(t, 2, result.NPublic)
	})

	t.Run("nil vkey", func(t *testing.T) {
		result, err := types.UnmarshalVKey(nil)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "nil vkey")
	})

	t.Run("empty key bytes", func(t *testing.T) {
		vkey := &types.VKey{
			KeyBytes:    []byte{},
			Name:        "test-vkey",
			Description: "test description",
		}
		result, err := types.UnmarshalVKey(vkey)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "empty key_bytes")
	})

	t.Run("nil key bytes", func(t *testing.T) {
		vkey := &types.VKey{
			KeyBytes:    nil,
			Name:        "test-vkey",
			Description: "test description",
		}
		result, err := types.UnmarshalVKey(vkey)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "empty key_bytes")
	})

	t.Run("invalid key bytes", func(t *testing.T) {
		vkey := &types.VKey{
			KeyBytes:    []byte("invalid json"),
			Name:        "test-vkey",
			Description: "test description",
		}
		result, err := types.UnmarshalVKey(vkey)
		require.Error(t, err)
		require.Nil(t, result)
	})
}

func TestMarshalVKey(t *testing.T) {
	t.Run("nil verification key", func(t *testing.T) {
		result, err := types.MarshalVKey(nil)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "nil verification key")
	})
}

func TestNewVKeyFromBytes(t *testing.T) {
	t.Run("valid bytes", func(t *testing.T) {
		vkey, err := types.NewVKeyFromBytes([]byte(validVKeyBase64), "test-vkey", "test description")
		require.NoError(t, err)
		require.NotNil(t, vkey)
		require.Equal(t, "test-vkey", vkey.Name)
		require.Equal(t, "test description", vkey.Description)
		require.Equal(t, validVKeyJSON, vkey.KeyBytes)
	})

	t.Run("empty bytes", func(t *testing.T) {
		vkey, err := types.NewVKeyFromBytes([]byte{}, "test-vkey", "test description")
		require.Error(t, err)
		require.Nil(t, vkey)
	})

	t.Run("nil bytes", func(t *testing.T) {
		vkey, err := types.NewVKeyFromBytes(nil, "test-vkey", "test description")
		require.Error(t, err)
		require.Nil(t, vkey)
	})

	t.Run("invalid bytes", func(t *testing.T) {
		vkey, err := types.NewVKeyFromBytes([]byte("invalid"), "test-vkey", "test description")
		require.Error(t, err)
		require.Nil(t, vkey)
	})

	t.Run("empty name is allowed", func(t *testing.T) {
		vkey, err := types.NewVKeyFromBytes([]byte(validVKeyBase64), "", "test description")
		require.NoError(t, err)
		require.NotNil(t, vkey)
		require.Equal(t, "", vkey.Name)
	})

	t.Run("empty description is allowed", func(t *testing.T) {
		vkey, err := types.NewVKeyFromBytes([]byte(validVKeyBase64), "test-vkey", "")
		require.NoError(t, err)
		require.NotNil(t, vkey)
		require.Equal(t, "", vkey.Description)
	})
}

func TestNewVKeyFromCircom(t *testing.T) {
	t.Run("nil verification key", func(t *testing.T) {
		vkey, err := types.NewVKeyFromCircom(nil, "test-vkey", "test description")
		require.Error(t, err)
		require.Nil(t, vkey)
	})
}
