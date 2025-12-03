package keeper_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	module "github.com/burnt-labs/xion/x/zk"
	"github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
)

type TestFixture struct {
	suite.Suite

	ctx         sdk.Context
	k           keeper.Keeper
	msgServer   types.MsgServer
	queryServer types.QueryServer
	appModule   *module.AppModule

	addrs      []sdk.AccAddress
	govModAddr string
}

func SetupTest(t *testing.T) *TestFixture {
	t.Helper()
	f := new(TestFixture)
	require := require.New(t)

	// Base setup
	logger := log.NewTestLogger(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()

	f.govModAddr = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	f.addrs = simtestutil.CreateIncrementalAccounts(3)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	f.ctx = testCtx.Ctx

	// Register SDK modules.
	registerBaseSDKModules(f, encCfg, storeService, logger, require)

	// Setup Keeper.
	f.k = keeper.NewKeeper(encCfg.Codec, storeService, logger, f.govModAddr)
	f.msgServer = keeper.NewMsgServerImpl(f.k)
	f.queryServer = keeper.NewQuerier(f.k)
	f.appModule = module.NewAppModule(encCfg.Codec, f.k)
	_, err := f.k.NextVKeyID.Next(f.ctx)
	require.NoError(err)
	return f
}

func registerModuleInterfaces(encCfg moduletestutil.TestEncodingConfig) {
	authtypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
}

func registerBaseSDKModules(
	_ *TestFixture,
	encCfg moduletestutil.TestEncodingConfig,
	_ store.KVStoreService,
	_ log.Logger,
	_ *require.Assertions,
) {
	registerModuleInterfaces(encCfg)
}

// ============================================================================
// Helper Functions
// ============================================================================

// createTestVKeyBytes creates test verification key bytes
func createTestVKeyBytes(name string) []byte {
	vkeyJSON := map[string]interface{}{
		"protocol": "groth16",
		"curve":    "bn128",
		"nPublic":  38,
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
			{"7850174206215699470265447232896181995827290198270289015740465586234900717907", "16523678720873355024606200259490897814230777287090684564045992432305305903916"},
			{"17922376841490468422230719141981555125977000092903241124840405511300695693879", "16879908878873540452403616204635150073676235604757590406356228597446555508506"},
			{"1", "0"},
		},
		"IC": [][]string{
			{"16345761349507568841945448715352487251156082486249641241269333890395550092325", "15639888734627413031771174683539829923528159462683758621112624369184646096104", "1"},
			{"11055012332354458918978931261095193553610674110386470193839168977147769291318", "18495181687409540921974137017149935367916420995207668759877518086349313830343", "1"},
			{"21150936161280233869743649126494505925478198551726979346300402824882356810646", "9652031166467489060281375211359383333949699987687697749491386176667729623453", "1"},
			{"814538308063826844589527308100561430629327822681875151161809692483074821175", "7924978536273527665402413351072311090675674492503585643503634583769692612886", "1"},
			{"18619945096400403667109423869334832457471682179137807584137260789044770391778", "8788843430093945770702138845771434096845590339746013080921126677546401494918", "1"},
			{"6274022586650790171911056567461002996586291372688456055847182843483118308987", "6231820191930914440377630168153320146676541846340321081330956425032026876025", "1"},
			{"18839856754997717173797006378963055889087172401106050673027539843896439156245", "18271572978220313014857821983623895354269701541194322275515242546873337354113", "1"},
			{"15910481176578110073830382434009271800790966218570125920773115027407050876730", "3250429508429500099737535512675659781645830846041769594871746149392807879968", "1"},
			{"8468672304749547457747267642558742432650700969899099023233114090807368349450", "18457481819988392622486879501614673228246467191729285043916115802477912581216", "1"},
			{"15437509996473162119581972421248996859564552982657956313221389848721744153426", "9578795941588911106722615889589888524656865843220921236375401979865773761603", "1"},
			{"1423117882269395438854951089789048427705265881032782841466513229482735141363", "18380380076605567800199619783229922607248976810991167112690984436440299654598", "1"},
			{"11576817459020046336356174755082456814868049594856133147984636805802909538076", "17651690559265030862457279936509996548735611449496276989375105140039902808619", "1"},
			{"14119324369752786692992581869532914264428095363772601939625677839787380741347", "19255029787764927858322304178168707288625510729084495684624339878298621054734", "1"},
			{"10455014369457260953546603901350293317280986968125853654161803080833112405194", "4579718390321875674190323324716719456879392506737588386482364111554674461324", "1"},
			{"17716457639409127002637582688391734831695122235098509075001528831931040487787", "15621980485806848526953240052210440986430790187471634183789179499751366191382", "1"},
			{"12039284734065295889311099284956411407998141514601107238734059103426236960218", "744308534108922101084534463376317739736931393446960744933852929340098124610", "1"},
			{"11589630136064225398841660269306381067380563303389468250521227438151382187514", "11488271967746856498800594969676359639393953020704023083961342547330735051472", "1"},
			{"18210645144057551519956204079662446899865086488143068803612147902518038132718", "18825237190536897753290580449486630518155037225877478388376155310855401604959", "1"},
			{"17995163892933294898868933664456213983640203857843745250821309684977987234517", "17410584977051901689796287433594778530582180872224433345653612887933443548678", "1"},
			{"350998552295525269718581845718589444527278418255448570019968443272111988127", "11892312518903890530512756912579921298678211438665632630222038716240042559851", "1"},
			{"13779580755838145230526612541553377206413022786289746343995225130253514460429", "15354351655566648337846749798914824294623812690245897514068472541463576814164", "1"},
			{"3010472827953762079945921531097058210041771380957169827665900226078738277773", "8884234412837336998822030793672220304421586930418225359376480879044815837260", "1"},
			{"18606009014109632177397924576960268910280472551765608628367566909623910118843", "10590045310747419948596961070195759226380724343634346496801620264269694134197", "1"},
			{"6338188197161649035797533562330402021886796259260260628223362736337968296423", "11763285367280838024358581673235720346127853476993785727234015450206419772940", "1"},
			{"18875274157545000206022655180377048709726769372627153664937062481564654286476", "15341449418018748750820528723349739262514268105978797146909496228706744992244", "1"},
			{"11823992809645627684822189431832292512025109384688150633062762222266559580443", "11369187393816539418647204156854821005365339455838924934875675647858631047622", "1"},
			{"9812407345688292533932709521711725202169771301540212056197133153658661052775", "9136601873625296951690940380356980805280288924792812953065910032529882904251", "1"},
			{"1766368643610160125679759048696626915818408165013426674646247697894733604169", "834554792554942560857910546095033687603355392730521652446771224751378121474", "1"},
			{"1796399370669564392925837331998817082267517416822002217494911737745099951234", "8588733018030157627116948211846218299209181830812662932320241951020688978892", "1"},
			{"11430723416565818706828349984154578819374312081586497979711251657344456111685", "15834355459294033206396986273803560288872622076070576567426990987286887884902", "1"},
			{"8876068927482580143464909224108136213632703581566102211617588819432989364303", "20848653408538406002979776199010027584654679072054999599936733265775441118561", "1"},
			{"18832379149337440760184131618723556370869648318430074876367129691118580455390", "9279684790573247490400970472675287465913318781853652384665377267023361198582", "1"},
			{"7199356662717213079174458016360121516235914798767853029777960063953649228247", "18186490680262757315429817865292767694505403548243939074721099532291925131483", "1"},
			{"14633306338498605465864268428779401253122507096291987982853913371852915245018", "4107442644635514805202208352499004660299725520580543704522646956225187095994", "1"},
			{"7728877955463075176104963237835954391273887401579835299270862135573395602919", "8992706051388210358768568811796669063338609460416399310146788655497856264176", "1"},
			{"17062281298779159348545619251348260218946228880347019330075723167684324738463", "12346588036783902545245050254447598440241793498427540625082014422367149841803", "1"},
			{"3052083114386799609987786524197093449821876275086753917295286467381459617898", "7258157583143883276533730080998930945592064334678140571638548882313971094594", "1"},
			{"17445392488294526940127819135041489991406588715587561269002232988064021286538", "12809678333115167851327338778897701127307701244739187998306646733588257286237", "1"},
			{"5946391734983842145337214606402429382596068410314432369963113583059054092133", "15553443032034361859495915995424871773221169139193313662684119863019424284916", "1"},
		},
	}

	bytes, _ := json.Marshal(vkeyJSON)
	return bytes
}

// loadVKeyFromJSON loads a vkey from a JSON file
func loadVKeyFromJSON(t *testing.T, filepath string) []byte {
	data, err := os.ReadFile(filepath)
	require.NoError(t, err)
	return data
}

// ============================================================================
// Basic Tests
// ============================================================================

func TestKeeperLogger(t *testing.T) {
	f := SetupTest(t)
	logger := f.k.Logger()
	require.NotNil(t, logger)
}

// ============================================================================
// VKey CRUD Tests
// ============================================================================

func TestAddVKey(t *testing.T) {
	f := SetupTest(t)

	tests := []struct {
		name        string
		authority   string
		vkeyName    string
		vkeyBytes   []byte
		description string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully add vkey",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			vkeyBytes:   createTestVKeyBytes("email_auth"),
			description: "Email authentication circuit",
			expectError: false,
		},
		{
			name:        "successfully add second vkey",
			authority:   f.govModAddr,
			vkeyName:    "rollup_batch",
			vkeyBytes:   createTestVKeyBytes("rollup_batch"),
			description: "Rollup batch verification",
			expectError: false,
		},
		{
			name:        "fail to add duplicate vkey name",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			vkeyBytes:   createTestVKeyBytes("email_auth"),
			description: "Duplicate",
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name:        "fail to add with incorrect authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "unauthorized",
			vkeyBytes:   createTestVKeyBytes("unauthorized"),
			description: "Unauthorized",
			expectError: true,
			errorMsg:    "invalid authority",
		},
		{
			name:        "fail to add with empty vkey bytes",
			authority:   f.govModAddr,
			vkeyName:    "empty_vkey",
			vkeyBytes:   []byte{},
			description: "Empty vkey",
			expectError: true,
			errorMsg:    "invalid verification key",
		},
		{
			name:        "fail to add with invalid JSON",
			authority:   f.govModAddr,
			vkeyName:    "invalid_json",
			vkeyBytes:   []byte("not valid json"),
			description: "Invalid JSON",
			expectError: true,
			errorMsg:    "invalid verification key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := f.k.AddVKey(f.ctx, tt.authority, tt.vkeyName, tt.vkeyBytes, tt.description)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Equal(t, uint64(0), id)
			} else {
				require.NoError(t, err)
				require.GreaterOrEqual(t, id, uint64(0))

				// Verify the vkey was stored correctly
				storedVKey, err := f.k.GetVKeyByID(f.ctx, id)
				require.NoError(t, err)
				require.Equal(t, tt.vkeyName, storedVKey.Name)
				require.Equal(t, tt.description, storedVKey.Description)
				require.Equal(t, tt.vkeyBytes, storedVKey.KeyBytes)
			}
		})
	}
}

func TestGetVKeyByID(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test verification key")
	require.NoError(t, err)

	tests := []struct {
		name        string
		id          uint64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully get existing vkey",
			id:          id,
			expectError: false,
		},
		{
			name:        "fail to get non-existent vkey",
			id:          9999,
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := f.k.GetVKeyByID(f.ctx, tt.id)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, "test_key", retrieved.Name)
				require.Equal(t, "Test verification key", retrieved.Description)
			}
		})
	}
}

func TestGetVKeyByName(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("email_auth")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication")
	require.NoError(t, err)

	tests := []struct {
		name        string
		vkeyName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully get vkey by name",
			vkeyName:    "email_auth",
			expectError: false,
		},
		{
			name:        "fail to get non-existent vkey",
			vkeyName:    "non_existent",
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := f.k.GetVKeyByName(f.ctx, tt.vkeyName)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.vkeyName, retrieved.Name)
				require.Equal(t, "Email authentication", retrieved.Description)
			}
		})
	}
}

func TestGetCircomVKeyByName(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("email_auth")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication")
	require.NoError(t, err)

	// Get as CircomVerificationKey
	circomVKey, err := f.k.GetCircomVKeyByName(f.ctx, "email_auth")
	require.NoError(t, err)
	require.NotNil(t, circomVKey)
	require.Equal(t, "groth16", circomVKey.Protocol)
	require.Equal(t, "bn128", circomVKey.Curve)
	require.Equal(t, 38, circomVKey.NPublic)
}

func TestHasVKey(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test key")
	require.NoError(t, err)

	tests := []struct {
		name     string
		vkeyName string
		expected bool
	}{
		{
			name:     "vkey exists",
			vkeyName: "test_key",
			expected: true,
		},
		{
			name:     "vkey does not exist",
			vkeyName: "non_existent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, err := f.k.HasVKey(f.ctx, tt.vkeyName)
			require.NoError(t, err)
			require.Equal(t, tt.expected, has)
		})
	}
}

func TestUpdateVKey(t *testing.T) {
	f := SetupTest(t)

	// Add initial vkey
	vkeyBytes := createTestVKeyBytes("email_auth")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Original description")
	require.NoError(t, err)

	tests := []struct {
		name        string
		authority   string
		vkeyName    string
		newBytes    []byte
		description string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully update vkey",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			newBytes:    createTestVKeyBytes("email_auth"),
			description: "Updated description",
			expectError: false,
		},
		{
			name:        "fail to update with incorrect authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "email_auth",
			newBytes:    createTestVKeyBytes("email_auth"),
			description: "Unauthorized",
			expectError: true,
			errorMsg:    "invalid authority",
		},
		{
			name:        "fail to update non-existent vkey",
			authority:   f.govModAddr,
			vkeyName:    "non_existent",
			newBytes:    createTestVKeyBytes("non_existent"),
			description: "Does not exist",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "fail with empty vkey bytes",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			newBytes:    []byte{},
			description: "Empty",
			expectError: true,
			errorMsg:    "invalid verification key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.k.UpdateVKey(f.ctx, tt.authority, tt.vkeyName, tt.newBytes, tt.description)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// Verify the update
				updated, err := f.k.GetVKeyByID(f.ctx, id)
				require.NoError(t, err)
				require.Equal(t, tt.description, updated.Description)
			}
		})
	}
}

func TestRemoveVKey(t *testing.T) {
	f := SetupTest(t)

	// Add test vkeys
	vkey1Bytes := createTestVKeyBytes("key1")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "Key 1")
	require.NoError(t, err)

	vkey2Bytes := createTestVKeyBytes("key2")
	_, err = f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Key 2")
	require.NoError(t, err)

	tests := []struct {
		name        string
		authority   string
		vkeyName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "fail to remove with incorrect authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "key1",
			expectError: true,
			errorMsg:    "invalid authority",
		},
		{
			name:        "successfully remove vkey",
			authority:   f.govModAddr,
			vkeyName:    "key1",
			expectError: false,
		},
		{
			name:        "fail to remove non-existent vkey",
			authority:   f.govModAddr,
			vkeyName:    "non_existent",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "fail to remove already removed vkey",
			authority:   f.govModAddr,
			vkeyName:    "key1",
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.k.RemoveVKey(f.ctx, tt.authority, tt.vkeyName)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// Verify the vkey was removed
				has, err := f.k.HasVKey(f.ctx, tt.vkeyName)
				require.NoError(t, err)
				require.False(t, has)
			}
		})
	}
}

func TestListVKeys(t *testing.T) {
	f := SetupTest(t)

	// Test with empty store
	vkeys, err := f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Empty(t, vkeys)

	// Add multiple vkeys
	for i := 0; i < 3; i++ {
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
		require.NoError(t, err)
	}

	// List all vkeys
	vkeys, err = f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Len(t, vkeys, 3)

	// Verify names are present
	names := make(map[string]bool)
	for _, vkey := range vkeys {
		names[vkey.Name] = true
	}
	require.True(t, names["key0"])
	require.True(t, names["key1"])
	require.True(t, names["key2"])
}

// ============================================================================
// Sequence Tests
// ============================================================================

func TestSequenceIncrement(t *testing.T) {
	f := SetupTest(t)

	// Add multiple vkeys and verify IDs increment
	vkey1Bytes := createTestVKeyBytes("key1")
	id1, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "Key 1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id1)

	vkey2Bytes := createTestVKeyBytes("key2")
	id2, err := f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Key 2")
	require.NoError(t, err)
	require.Equal(t, uint64(2), id2)

	vkey3Bytes := createTestVKeyBytes("key3")
	id3, err := f.k.AddVKey(f.ctx, f.govModAddr, "key3", vkey3Bytes, "Key 3")
	require.NoError(t, err)
	require.Equal(t, uint64(3), id3)
}

func TestSequencePersistence(t *testing.T) {
	f := SetupTest(t)

	// Add first vkey
	vkey1Bytes := createTestVKeyBytes("key1")
	id1, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "Key 1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id1)

	// Add second vkey - sequence should increment
	vkey2Bytes := createTestVKeyBytes("key2")
	id2, err := f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Key 2")
	require.NoError(t, err)
	require.Equal(t, uint64(2), id2)

	// Verify both vkeys exist
	retrieved1, err := f.k.GetVKeyByID(f.ctx, id1)
	require.NoError(t, err)
	require.Equal(t, "key1", retrieved1.Name)

	retrieved2, err := f.k.GetVKeyByID(f.ctx, id2)
	require.NoError(t, err)
	require.Equal(t, "key2", retrieved2.Name)
}

// ============================================================================
// Index Tests
// ============================================================================

func TestNameIndexConsistency(t *testing.T) {
	f := SetupTest(t)

	// Add vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test")
	require.NoError(t, err)

	// Verify both ID and name access return the same vkey
	vkeyByID, err := f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)

	vkeyByName, err := f.k.GetVKeyByName(f.ctx, "test_key")
	require.NoError(t, err)

	require.Equal(t, vkeyByID.Name, vkeyByName.Name)
	require.Equal(t, vkeyByID.Description, vkeyByName.Description)
	require.Equal(t, vkeyByID.KeyBytes, vkeyByName.KeyBytes)
}

func TestNameIndexAfterRemoval(t *testing.T) {
	f := SetupTest(t)

	// Add vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test")
	require.NoError(t, err)

	// Remove vkey
	err = f.k.RemoveVKey(f.ctx, f.govModAddr, "test_key")
	require.NoError(t, err)

	// Verify both ID and name access fail
	_, err = f.k.GetVKeyByID(f.ctx, id)
	require.Error(t, err)

	_, err = f.k.GetVKeyByName(f.ctx, "test_key")
	require.Error(t, err)

	// Verify name is not in index
	has, err := f.k.HasVKey(f.ctx, "test_key")
	require.NoError(t, err)
	require.False(t, has)
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestEmptyName(t *testing.T) {
	f := SetupTest(t)

	vkeyBytes := createTestVKeyBytes("")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "", vkeyBytes, "Empty name test")

	// Empty names should be allowed at keeper level
	// Validation should happen at message level
	require.NoError(t, err)
	require.GreaterOrEqual(t, id, uint64(0))

	retrieved, err := f.k.GetVKeyByName(f.ctx, "")
	require.NoError(t, err)
	require.Equal(t, "", retrieved.Name)
}

func TestVeryLongName(t *testing.T) {
	f := SetupTest(t)

	longName := string(make([]byte, 1000))
	for i := 0; i < 1000; i++ {
		longName = longName[:i] + "a" + longName[i+1:]
	}

	vkeyBytes := createTestVKeyBytes(longName)
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, longName, vkeyBytes, "Long name test")
	require.NoError(t, err)
	require.GreaterOrEqual(t, id, uint64(0))

	// Verify retrieval works
	retrieved, err := f.k.GetVKeyByName(f.ctx, longName)
	require.NoError(t, err)
	require.Equal(t, longName, retrieved.Name)
}

func TestConcurrentAccess(t *testing.T) {
	f := SetupTest(t)

	// Add multiple vkeys
	for i := 1; i < 11; i++ {
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
		require.NoError(t, err)
	}

	// Access all vkeys by both ID and name
	for i := 1; i < 11; i++ {
		vkeyByID, err := f.k.GetVKeyByID(f.ctx, uint64(i))
		require.NoError(t, err)

		vkeyByName, err := f.k.GetVKeyByName(f.ctx, fmt.Sprintf("key%d", i))
		require.NoError(t, err)

		require.Equal(t, vkeyByID.Name, vkeyByName.Name)
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestFullVKeyLifecycle(t *testing.T) {
	f := SetupTest(t)

	// 1. Add vkey
	vkeyBytes := createTestVKeyBytes("lifecycle_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "lifecycle_key", vkeyBytes, "Initial version")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)

	// 2. Verify it exists
	has, err := f.k.HasVKey(f.ctx, "lifecycle_key")
	require.NoError(t, err)
	require.True(t, has)

	// 3. Get by ID
	retrieved, err := f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Initial version", retrieved.Description)

	// 4. Get by name
	retrievedByName, err := f.k.GetVKeyByName(f.ctx, "lifecycle_key")
	require.NoError(t, err)
	require.Equal(t, retrieved.Name, retrievedByName.Name)

	// 5. Update
	updatedBytes := createTestVKeyBytes("lifecycle_key")
	err = f.k.UpdateVKey(f.ctx, f.govModAddr, "lifecycle_key", updatedBytes, "Updated version")
	require.NoError(t, err)

	// 6. Verify update
	retrieved, err = f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Updated version", retrieved.Description)

	// 7. List all keys
	vkeys, err := f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Len(t, vkeys, 1)

	// 8. Remove
	err = f.k.RemoveVKey(f.ctx, f.govModAddr, "lifecycle_key")
	require.NoError(t, err)

	// 9. Verify removal
	has, err = f.k.HasVKey(f.ctx, "lifecycle_key")
	require.NoError(t, err)
	require.False(t, has)

	// 10. List should be empty
	vkeys, err = f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Empty(t, vkeys)
}

// ============================================================================
// Circom Integration Tests
// ============================================================================

func TestCircomVKeyConversion(t *testing.T) {
	f := SetupTest(t)

	// Add vkey
	vkeyBytes := createTestVKeyBytes("circom_test")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "circom_test", vkeyBytes, "Circom test")
	require.NoError(t, err)

	// Get as standard VKey
	vkey, err := f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "circom_test", vkey.Name)

	// Get as CircomVerificationKey
	circomVKey, err := f.k.GetCircomVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.NotNil(t, circomVKey)
	require.Equal(t, "groth16", circomVKey.Protocol)
	require.Equal(t, "bn128", circomVKey.Curve)

	// Verify it can be used with the parser
	require.Equal(t, 38, circomVKey.NPublic)
	require.Len(t, circomVKey.VkAlpha1, 3)
	require.Len(t, circomVKey.IC, 39)
}

func TestLoadActualVKeyFile(t *testing.T) {
	// Skip if test file doesn't exist
	if _, err := os.Stat("testdata/email_auth_vkey.json"); os.IsNotExist(err) {
		t.Skip("testdata/email_auth_vkey.json not found")
	}

	f := SetupTest(t)

	// Load actual vkey from file
	vkeyBytes := loadVKeyFromJSON(t, "testdata/email_auth_vkey.json")

	// Add to keeper
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication from file")
	require.NoError(t, err)

	// Retrieve and verify
	circomVKey, err := f.k.GetCircomVKeyByName(f.ctx, "email_auth")
	require.NoError(t, err)
	require.NotNil(t, circomVKey)
	require.Equal(t, "groth16", circomVKey.Protocol)
	require.Equal(t, "bn128", circomVKey.Curve)
	require.Equal(t, 34, circomVKey.NPublic)
	require.Len(t, circomVKey.IC, 35) // 34 public inputs + 1
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestValidateVKeyBytes(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid vkey",
			data:        createTestVKeyBytes("test"),
			expectError: false,
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectError: true,
			errorMsg:    "empty vkey data",
		},
		{
			name:        "invalid json",
			data:        []byte("not json"),
			expectError: true,
			errorMsg:    "invalid verification key JSON",
		},
		{
			name: "missing required fields",
			data: []byte(`{
				"protocol": "groth16",
				"curve": "bn128"
			}`),
			expectError: true,
			errorMsg:    "invalid nPublic: 0 (must be greater than 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateVKeyBytes(tt.data)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
