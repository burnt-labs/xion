// keeper/query_server_test.go
package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/zk/types"
)

// Define the proof data here to be used in the test
var proofData = []byte(`{
    "pi_a": [
        "2567498309095945123001915525425675597905999851760478825045526651681215626331",
        "14999488854001729096264262765481549520419110121706604091382799335768138359729",
        "1"
    ],
    "pi_b": [
        [
            "17898391853305250165364803572914046217143846059832421998113030577162188453310",
            "4497137125678880872219151037091068253258857082997424069216822431849925822836"
        ],
        [
            "19330055590884309950552162558742614535190676739309283167287289418499537555510",
            "36639813998385593976084071080638627426479836445528054913859022095575330980"
        ],
        [
            "1",
            "0"
        ]
    ],
    "pi_c": [
        "6376195530180454357718402630715779929757331091355181280995534997318492855333",
        "2057527013472228268989188433761933215313085128111815310161468273481706106794",
        "1"
    ],
    "protocol": "groth16",
    "curve": "bn128"
}`)

// Define the public inputs here to be used in the test
var publicInputs = []string{
	"2018721414038404820327",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"6632353713085157925504008443078919716322386156160602218536961028046468237192",
	"6488481959449533072223265512935826955293610794623716027306441809557838942137",
	"1761034954",
	"184361564063070453273685922136003966338692915846469267013988016589082740581",
	"156169086250226200330543370821913437019311556943728422938452698686684619377",
	"43933152500220616752048431712410451884662320338205006",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"8106355043968901587346579634598098765933160394002251948170420219958523220425",
	"1",
}

// Define the vkey here to be used in the test
var vkeyJSON = []byte(`{
    "protocol": "groth16",
    "curve": "bn128",
    "nPublic": 34,
    "vk_alpha_1": [
        "20491192805390485299153009773594534940189261866228447918068658471970481763042",
        "9383485363053290200918347156157836566562967994039712273449902621266178545958",
        "1"
    ],
    "vk_beta_2": [
        [
            "6375614351688725206403948262868962793625744043794305715222011528459656738731",
            "4252822878758300859123897981450591353533073413197771768651442665752259397132"
        ],
        [
            "10505242626370262277552901082094356697409835680220590971873171140371331206856",
            "21847035105528745403288232691147584728191162732299865338377159692350059136679"
        ],
        [
            "1",
            "0"
        ]
    ],
    "vk_gamma_2": [
        [
            "10857046999023057135944570762232829481370756359578518086990519993285655852781",
            "11559732032986387107991004021392285783925812861821192530917403151452391805634"
        ],
        [
            "8495653923123431417604973247489272438418190587263600148770280649306958101930",
            "4082367875863433681332203403145435568316851327593401208105741076214120093531"
        ],
        [
            "1",
            "0"
        ]
    ],
    "vk_delta_2": [
        [
            "7408543996799841808823674318962923691422846694508104677211507255777183761346",
            "17378314708652486082434193052153411074104970941065581812653446685054220492752"
        ],
        [
            "20934765493363178521480199624017210946632719146191129233788277268880988392769",
            "9933248257943163684434361179172132751107201169345727211797322171844177096469"
        ],
        [
            "1",
            "0"
        ]
    ],
    "vk_alphabeta_12": [
        [
            [
                "2029413683389138792403550203267699914886160938906632433982220835551125967885",
                "21072700047562757817161031222997517981543347628379360635925549008442030252106"
            ],
            [
                "5940354580057074848093997050200682056184807770593307860589430076672439820312",
                "12156638873931618554171829126792193045421052652279363021382169897324752428276"
            ],
            [
                "7898200236362823042373859371574133993780991612861777490112507062703164551277",
                "7074218545237549455313236346927434013100842096812539264420499035217050630853"
            ]
        ],
        [
            [
                "7077479683546002997211712695946002074877511277312570035766170199895071832130",
                "10093483419865920389913245021038182291233451549023025229112148274109565435465"
            ],
            [
                "4595479056700221319381530156280926371456704509942304414423590385166031118820",
                "19831328484489333784475432780421641293929726139240675179672856274388269393268"
            ],
            [
                "11934129596455521040620786944827826205713621633706285934057045369193958244500",
                "8037395052364110730298837004334506829870972346962140206007064471173334027475"
            ]
        ]
    ],
    "IC": [
        [
            "5449013234494434531196202102845211237542489505716355090765771488165044993949",
            "4910919431725277797191489997138444712176878647014509270723700672161925471159",
            "1"
        ],
        [
            "1042941570586670607216203624327203511181285800117974445407657862726542209364",
            "21292264478713151582046296922334342492237657525683026669790255619136803722319",
            "1"
        ],
        [
            "10613770212523647306892678839213336433758613146181871056915296771824847091092",
            "13205323641726628911889305865218463621756734356649589611559091778239352156527",
            "1"
        ],
        [
            "21027493177221374353572719253237598487755392258026658309284045911722674697008",
            "5342976149096616602386127244530271257284062143627856668312257521654456691824",
            "1"
        ],
        [
            "6821787947117018801042252294732093673411592885499209024991791369368982090641",
            "2284139314919869012676941451651347384248504086626782755512802081757843236992",
            "1"
        ],
        [
            "11435200949599879962170137482803341339526579057856717749566976284390370934077",
            "21673276099604790368428412051116222047830777891719327801087175041159900029407",
            "1"
        ],
        [
            "9509421953262664373127145431357970434917693613664282682823641552834889301831",
            "170878110043135288484806341864414014904214958909618756667730824826631740714",
            "1"
        ],
        [
            "4690166979653753691831464813099258002946096107732277722422205431814666342510",
            "7519806354784033944185549157769611503953663024021477606258592075096443488243",
            "1"
        ],
        [
            "11359947907291229388471195403978088867644736395681408547987866857366397389877",
            "18654828619639674814452294771122961247712591576826738460549241573010962337556",
            "1"
        ],
        [
            "12497684518299231737163562517102232962526447989122504032252820137192410295019",
            "21625071755731537300565265736220205650053655959982492312709032795888925641680",
            "1"
        ],
        [
            "639856633525195035417148119968210462662005581929105920184976580249337263675",
            "1303911675511133538453716149873001292601863325505660055639112603338994187513",
            "1"
        ],
        [
            "9929817038847081168556060828631438785970231764576961090800435947531764944705",
            "20909377524002983988540762303644408655812329892396371683450440541453248152221",
            "1"
        ],
        [
            "10150534440150744614023907819846875263872680836458178834089579692946908095350",
            "13651968648717144605361934826292707448213053895567684979174986322613920668655",
            "1"
        ],
        [
            "10632526416347516094520036245753779322430546328670034122623440495608904037810",
            "9102528815699498334464991363867419496022272863662928569711486930585329876763",
            "1"
        ],
        [
            "18227223693625813032686637745764794201781360048765116463552097758739219174551",
            "8486619629211288880803317659748455714249855076544338351202138993256155806533",
            "1"
        ],
        [
            "17469269818385789318854477951678479462167302001569590697444497159526056274822",
            "703888102814731542542043245395590509550605931460573366944657015913974942867",
            "1"
        ],
        [
            "3395454671108168638568775612532799010158524285246147076890077828084013329812",
            "13635316841426442197444680071699602348620632632716540715781868727490594642576",
            "1"
        ],
        [
            "16248191416219927151946233046968151753519209271294427310260694005336477911970",
            "17177977786775571420135628771021038913222292416079315532065680283503732172960",
            "1"
        ],
        [
            "10848938149995645836033652289628613081427673954303991024289050469140624601816",
            "5238636047010484176949043589567249845528363585228024985180125446504292459413",
            "1"
        ],
        [
            "1640695080564491414790312918295317082480580001494506424057910258325423961401",
            "10019482578421845089709583085239042844677727869099098416083792267909255914657",
            "1"
        ],
        [
            "4472403974296112816027698964628009061033461454207642564368220048925329150357",
            "4249237059490399232489541017717278484995198722596749352656443774856588851624",
            "1"
        ],
        [
            "1029066676527445678340004557075685708542033431455576846327407072900029682963",
            "15546762218917894970493951818776180613671613413704426576941020218538550726966",
            "1"
        ],
        [
            "16040522247884739632828558088285849801209576922157918848181514562970818876827",
            "1332109029477288621540091927038034765819117427807435547633165484793673522850",
            "1"
        ],
        [
            "20871217464738063538668378251841386081784486676269615962382803997356000707044",
            "2896359840725033945604444438736601893905714572071013917525263100165303844345",
            "1"
        ],
        [
            "15035460057904015529303755554307407930873743538060289322336582416573534605359",
            "20245159220969974654605894868381082394566115856436225185766604484084146021231",
            "1"
        ],
        [
            "2518131037805129248326685217436012240978457064719590356674124525772679782717",
            "19716665734299465665166365398912278121290822921565760578626174246069475532920",
            "1"
        ],
        [
            "14717213682028789513367386498719297504630482601968877565801004556683286248671",
            "11234408269966853532884757279809627500870338696022559123550928008898233982295",
            "1"
        ],
        [
            "14789022829545696314877182216953729433897647922419164950612204639934934509721",
            "4070413295585816111154788109749433300711333317552388303661704373893626509410",
            "1"
        ],
        [
            "14424577334948790297024913618693897475455128596717128415362884202321646034750",
            "3587050701637775869264917612434517522244142453721636232580044371903774279044",
            "1"
        ],
        [
            "2081812754771099082261473743649129655197525250957166063365198761994866065584",
            "2413930391942598562341479580687030191614920204675745274166847303572565682050",
            "1"
        ],
        [
            "8208494896859763607620991855678354918970044976780985776676582106010036343374",
            "2183743894701102458136536798057520247396193183014463286529942356805005648951",
            "1"
        ],
        [
            "223213163222255594379588772021234412106022659538702924243022409858590609042",
            "13231018155630996975174204514301927597103820568819635259253020356626005177172",
            "1"
        ],
        [
            "1412256121377497633668251290331281245148423561182632442292241299483409220466",
            "11138964720405665576657661641442665299810196963532259012656392145520052597279",
            "1"
        ],
        [
            "11010979441335662346639471989176904218963230299595511274754312371239142524812",
            "20401783817047613007922800457563465453549252646684636746483325562422663512175",
            "1"
        ],
        [
            "10623244848931481921695349407733157053134011967719483953012449616937404618645",
            "8779231572110654158986828194215557939293838391946454691845714237981588017427",
            "1"
        ]
	]
}`)

var vkeyData = vkeyJSON

func TestQueryProofVerify(t *testing.T) {
	f := SetupTest(t)

	// Add valid vkey to the keeper for successful tests
	validVKeyID, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth_circuit", vkeyData, "Email authentication circuit")
	require.NoError(t, err)

	// Add an invalid vkey for error testing
	invalidVKey := []byte(`{
		"protocol": "groth16",
		"curve": "bn128",
		"nPublic": 2,
		"vk_alpha_1": ["1", "2", "1"],
		"vk_beta_2": [["3", "4"], ["5", "6"], ["1", "0"]],
		"vk_gamma_2": [["7", "8"], ["9", "10"], ["1", "0"]],
		"vk_delta_2": [["11", "12"], ["13", "14"], ["1", "0"]],
		"IC": [
			["15", "16", "1"],
			["17", "18", "1"],
			["19", "20", "1"]
		]
	}`)
	invalidVKeyID, err := f.k.AddVKey(f.ctx, f.govModAddr, "invalid_circuit", invalidVKey, "Invalid circuit for testing")
	require.NoError(t, err)

	testCases := []struct {
		name         string
		proofBz      []byte
		publicInputs []string
		vkeyName     string
		vkeyID       uint64
		shouldError  bool
		errorMsg     string
	}{
		{
			name:         "verify proof success with valid data using vkey name",
			proofBz:      proofData,
			publicInputs: publicInputs,
			vkeyName:     "email_auth_circuit",
			shouldError:  false,
		},
		{
			name:         "verify proof success with valid data using vkey ID",
			proofBz:      proofData,
			publicInputs: publicInputs,
			vkeyID:       validVKeyID,
			shouldError:  false,
		},
		{
			name:         "invalid proof data format",
			proofBz:      []byte("invalid-json-proof"),
			publicInputs: publicInputs,
			vkeyName:     "email_auth_circuit",
			shouldError:  true,
			errorMsg:     "failed to parse proof JSON",
		},
		{
			name:         "empty proof data",
			proofBz:      []byte{},
			publicInputs: publicInputs,
			vkeyName:     "email_auth_circuit",
			shouldError:  true,
			errorMsg:     "proof cannot be empty",
		},
		{
			name:         "non-existent vkey name",
			proofBz:      proofData,
			publicInputs: publicInputs,
			vkeyName:     "non_existent_circuit",
			shouldError:  true,
			errorMsg:     "not found",
		},
		{
			name:         "non-existent vkey ID",
			proofBz:      proofData,
			publicInputs: publicInputs,
			vkeyID:       9999,
			shouldError:  true,
			errorMsg:     "not found",
		},
		{
			name:         "neither vkey name nor ID provided",
			proofBz:      proofData,
			publicInputs: publicInputs,
			shouldError:  true,
			errorMsg:     "either vkey_name or vkey_id must be provided",
		},
		{
			name:         "mismatched public inputs count",
			proofBz:      proofData,
			publicInputs: []string{"1", "2"}, // Wrong number of inputs
			vkeyID:       validVKeyID,
			shouldError:  true,
			errorMsg:     "verification failed",
		},
		{
			name:         "empty public inputs",
			proofBz:      proofData,
			publicInputs: []string{},
			vkeyID:       validVKeyID,
			shouldError:  true,
			errorMsg:     "verification failed",
		},
		{
			name:         "invalid public input values",
			proofBz:      proofData,
			publicInputs: make([]string, len(publicInputs)), // Same count but all empty
			vkeyID:       validVKeyID,
			shouldError:  true,
			errorMsg:     "failed to parse public input",
		},
		{
			name:         "wrong vkey for proof",
			proofBz:      proofData,
			publicInputs: publicInputs,
			vkeyID:       invalidVKeyID, // Using wrong vkey
			shouldError:  true,
			errorMsg:     "invalid point",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &types.QueryVerifyRequest{
				Proof:        tc.proofBz,
				PublicInputs: tc.publicInputs,
			}

			// Set either vkey name or ID based on test case
			if tc.vkeyName != "" {
				req.VkeyName = tc.vkeyName
			}
			if tc.vkeyID != 0 {
				req.VkeyId = tc.vkeyID
			}

			res, err := f.queryServer.ProofVerify(f.ctx, req)

			if tc.shouldError {
				require.Error(t, err, "Expected error for test case: %s", tc.name)
				if tc.errorMsg != "" {
					require.Contains(t, err.Error(), tc.errorMsg, "Error message mismatch for test case: %s", tc.name)
				}
				require.Nil(t, res, "Response should be nil on error for test case: %s", tc.name)
			} else {
				require.NoError(t, err, "Unexpected error for test case: %s", tc.name)
				require.NotNil(t, res, "Response should not be nil for test case: %s", tc.name)
				require.True(t, res.Verified, "Proof should be verified for test case: %s", tc.name)
			}
		})
	}
}

// TestQueryProofVerifyWithStoredVKey tests the complete flow of storing a vkey and using it for verification
func TestQueryProofVerifyWithStoredVKey(t *testing.T) {
	f := SetupTest(t)

	// 1. Add vkey to keeper
	vkeyID, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyData, "Email authentication circuit")
	require.NoError(t, err)
	require.Equal(t, uint64(1), vkeyID)

	// 2. Verify it was stored correctly
	storedVKey, err := f.k.GetVKeyByName(f.ctx, "email_auth")
	require.NoError(t, err)
	require.Equal(t, "email_auth", storedVKey.Name)
	require.Equal(t, vkeyJSON, storedVKey.KeyBytes)

	// 3. Verify we can retrieve it as CircomVerificationKey
	circomVKey, err := f.k.GetCircomVKeyByName(f.ctx, "email_auth")
	require.NoError(t, err)
	require.NotNil(t, circomVKey)
	require.Equal(t, "groth16", circomVKey.Protocol)
	require.Equal(t, "bn128", circomVKey.Curve)
	require.Equal(t, 34, circomVKey.NPublic)

	// 4. Verify proof using the stored vkey by name
	reqByName := &types.QueryVerifyRequest{
		Proof:        proofData,
		PublicInputs: publicInputs,
		VkeyName:     "email_auth",
	}

	respByName, err := f.queryServer.ProofVerify(f.ctx, reqByName)
	require.NoError(t, err)
	require.NotNil(t, respByName)
	require.True(t, respByName.Verified)

	// 5. Verify proof using the stored vkey by ID
	reqByID := &types.QueryVerifyRequest{
		Proof:        proofData,
		PublicInputs: publicInputs,
		VkeyId:       vkeyID,
	}

	respByID, err := f.queryServer.ProofVerify(f.ctx, reqByID)
	require.NoError(t, err)
	require.NotNil(t, respByID)
	require.True(t, respByID.Verified)
}

// TestQueryProofVerifyMultipleVKeys tests verification with multiple stored vkeys
func TestQueryProofVerifyMultipleVKeys(t *testing.T) {
	f := SetupTest(t)

	// Add multiple vkeys
	vkey1ID, err := f.k.AddVKey(f.ctx, f.govModAddr, "circuit_1", vkeyData, "Circuit 1")
	require.NoError(t, err)

	vkey2ID, err := f.k.AddVKey(f.ctx, f.govModAddr, "circuit_2", vkeyData, "Circuit 2")
	require.NoError(t, err)

	vkey3ID, err := f.k.AddVKey(f.ctx, f.govModAddr, "circuit_3", vkeyData, "Circuit 3")
	require.NoError(t, err)

	// Verify proof with each vkey
	for i, vkeyID := range []uint64{vkey1ID, vkey2ID, vkey3ID} {
		t.Run(fmt.Sprintf("verify_with_vkey_%d", i+1), func(t *testing.T) {
			req := &types.QueryVerifyRequest{
				Proof:        proofData,
				PublicInputs: publicInputs,
				VkeyId:       vkeyID,
			}

			resp, err := f.queryServer.ProofVerify(f.ctx, req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.Verified)
		})
	}
}

// TestQueryProofVerifyNilRequest tests nil request handling
func TestQueryProofVerifyNilRequest(t *testing.T) {
	f := SetupTest(t)

	resp, err := f.queryServer.ProofVerify(f.ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty request")
	require.Nil(t, resp)
}

// ============================================================================
// VKey Query Tests
// ============================================================================

func TestQueryVKey(t *testing.T) {
	f := SetupTest(t)

	// Add a test vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test verification key")
	require.NoError(t, err)

	tests := []struct {
		name        string
		req         *types.QueryVKeyRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "successfully query vkey by ID",
			req: &types.QueryVKeyRequest{
				Id: id,
			},
			expectError: false,
		},
		{
			name: "fail to query non-existent vkey",
			req: &types.QueryVKeyRequest{
				Id: 9999,
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorMsg:    "empty request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := f.queryServer.VKey(f.ctx, tt.req)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, "test_key", resp.Vkey.Name)
				require.Equal(t, "Test verification key", resp.Vkey.Description)
			}
		})
	}
}

func TestQueryVKeyByName(t *testing.T) {
	f := SetupTest(t)

	// Add a test vkey
	vkeyBytes := createTestVKeyBytes("email_auth")
	expectedID, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication")
	require.NoError(t, err)

	tests := []struct {
		name        string
		req         *types.QueryVKeyByNameRequest
		expectError bool
		errorMsg    string
		expectedID  uint64
	}{
		{
			name: "successfully query vkey by name",
			req: &types.QueryVKeyByNameRequest{
				Name: "email_auth",
			},
			expectError: false,
			expectedID:  expectedID,
		},
		{
			name: "fail to query non-existent vkey",
			req: &types.QueryVKeyByNameRequest{
				Name: "non_existent",
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "fail with empty name",
			req: &types.QueryVKeyByNameRequest{
				Name: "",
			},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorMsg:    "empty request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := f.queryServer.VKeyByName(f.ctx, tt.req)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, "email_auth", resp.Vkey.Name)
				require.Equal(t, "Email authentication", resp.Vkey.Description)
				require.Equal(t, tt.expectedID, resp.Id)
			}
		})
	}
}

func TestQueryVKeys(t *testing.T) {
	f := SetupTest(t)

	// Test with empty store
	resp, err := f.queryServer.VKeys(f.ctx, &types.QueryVKeysRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.Vkeys)

	// Add multiple vkeys
	for i := 0; i < 5; i++ {
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		req           *types.QueryVKeysRequest
		expectedCount int
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "query all vkeys without pagination",
			req:           &types.QueryVKeysRequest{},
			expectedCount: 5,
			expectError:   false,
		},
		{
			name: "query with pagination - first page",
			req: &types.QueryVKeysRequest{
				Pagination: &query.PageRequest{
					Limit: 2,
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "query with pagination - offset",
			req: &types.QueryVKeysRequest{
				Pagination: &query.PageRequest{
					Offset: 2,
					Limit:  2,
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "query with pagination - large offset",
			req: &types.QueryVKeysRequest{
				Pagination: &query.PageRequest{
					Offset: 10,
					Limit:  2,
				},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorMsg:    "empty request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := f.queryServer.VKeys(f.ctx, tt.req)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Len(t, resp.Vkeys, tt.expectedCount)

				// Verify structure
				for _, vkeyWithID := range resp.Vkeys {
					require.NotEmpty(t, vkeyWithID.Vkey.Name)
					require.NotEmpty(t, vkeyWithID.Vkey.KeyBytes)
					require.GreaterOrEqual(t, vkeyWithID.Id, uint64(0))
				}
			}
		})
	}
}

func TestQueryVKeysPagination(t *testing.T) {
	f := SetupTest(t)

	// Add 10 vkeys
	totalVKeys := 10
	for i := 0; i < totalVKeys; i++ {
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
		require.NoError(t, err)
	}

	// Test pagination - get all items in pages of 3
	var allVKeys []types.VKeyWithID
	var nextKey []byte
	pageSize := uint64(3)

	for {
		resp, err := f.queryServer.VKeys(f.ctx, &types.QueryVKeysRequest{
			Pagination: &query.PageRequest{
				Key:   nextKey,
				Limit: pageSize,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		allVKeys = append(allVKeys, resp.Vkeys...)

		if resp.Pagination.NextKey == nil {
			break
		}
		nextKey = resp.Pagination.NextKey
	}

	// Should have retrieved all 10 vkeys
	require.Len(t, allVKeys, totalVKeys)

	// Verify all names are unique
	names := make(map[string]bool)
	for _, vkey := range allVKeys {
		require.False(t, names[vkey.Vkey.Name], "duplicate vkey name: %s", vkey.Vkey.Name)
		names[vkey.Vkey.Name] = true
	}
}

func TestQueryHasVKey(t *testing.T) {
	f := SetupTest(t)

	// Add a test vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	expectedID, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test key")
	require.NoError(t, err)

	tests := []struct {
		name           string
		req            *types.QueryHasVKeyRequest
		expectError    bool
		errorMsg       string
		expectedExists bool
		expectedID     uint64
	}{
		{
			name: "vkey exists",
			req: &types.QueryHasVKeyRequest{
				Name: "test_key",
			},
			expectError:    false,
			expectedExists: true,
			expectedID:     expectedID,
		},
		{
			name: "vkey does not exist",
			req: &types.QueryHasVKeyRequest{
				Name: "non_existent",
			},
			expectError:    false,
			expectedExists: false,
			expectedID:     0,
		},
		{
			name: "empty name",
			req: &types.QueryHasVKeyRequest{
				Name: "",
			},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorMsg:    "empty request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := f.queryServer.HasVKey(f.ctx, tt.req)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.expectedExists, resp.Exists)
				require.Equal(t, tt.expectedID, resp.Id)
			}
		})
	}
}

func TestQueryNextVKeyID(t *testing.T) {
	f := SetupTest(t)

	t.Run("query next vkey id on empty store", func(t *testing.T) {
		resp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		// Initial sequence value after setup (SetupTest calls NextVKeyID.Next once)
		require.GreaterOrEqual(t, resp.NextId, uint64(1))
	})

	t.Run("next id increments after adding vkey", func(t *testing.T) {
		// Get initial next ID
		initialResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		initialNextID := initialResp.NextId

		// Add a vkey
		vkeyBytes := createTestVKeyBytes("key1")
		_, err = f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkeyBytes, "Key 1")
		require.NoError(t, err)

		// Check next ID has incremented
		resp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		require.Equal(t, initialNextID+1, resp.NextId)
	})

	t.Run("next id increments correctly after multiple additions", func(t *testing.T) {
		// Get current next ID
		beforeResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		beforeNextID := beforeResp.NextId

		// Add multiple vkeys
		numToAdd := 4
		for i := 0; i < numToAdd; i++ {
			vkeyBytes := createTestVKeyBytes(fmt.Sprintf("multi_key%d", i))
			_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("multi_key%d", i), vkeyBytes, fmt.Sprintf("Multi Key %d", i))
			require.NoError(t, err)
		}

		// Get next ID after adding vkeys
		afterResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		require.Equal(t, beforeNextID+uint64(numToAdd), afterResp.NextId)
	})

	t.Run("query does not consume sequence (peek behavior)", func(t *testing.T) {
		// Query next ID multiple times
		resp1, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)

		resp2, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)

		resp3, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)

		// All responses should be the same (query doesn't consume the sequence)
		require.Equal(t, resp1.NextId, resp2.NextId)
		require.Equal(t, resp2.NextId, resp3.NextId)
	})

	t.Run("next id unchanged after vkey removal", func(t *testing.T) {
		// Add a vkey to remove
		vkeyBytes := createTestVKeyBytes("to_remove")
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, "to_remove", vkeyBytes, "To Remove")
		require.NoError(t, err)

		// Get next ID before removal
		beforeResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		beforeNextID := beforeResp.NextId

		// Remove the vkey
		err = f.k.RemoveVKey(f.ctx, f.govModAddr, "to_remove")
		require.NoError(t, err)

		// Next ID should remain unchanged after removal
		afterResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		require.Equal(t, beforeNextID, afterResp.NextId)
	})

	t.Run("nil request returns error", func(t *testing.T) {
		resp, err := f.queryServer.NextVKeyID(f.ctx, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty request")
		require.Nil(t, resp)
	})
}

func TestQueryNextVKeyIDPredictability(t *testing.T) {
	f := SetupTest(t)

	// Test that NextVKeyID correctly predicts the ID that will be assigned
	for i := 0; i < 5; i++ {
		// Get predicted next ID
		predResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		predictedID := predResp.NextId

		// Add vkey and get actual ID
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("predict_key%d", i))
		actualID, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("predict_key%d", i), vkeyBytes, fmt.Sprintf("Predict Key %d", i))
		require.NoError(t, err)

		// Verify prediction was correct
		require.Equal(t, predictedID, actualID, "NextVKeyID should correctly predict the ID for the next vkey")
	}
}

func TestQueryNextVKeyIDAfterGenesis(t *testing.T) {
	f := SetupTest(t)

	// Initialize genesis with vkeys that have high IDs
	gs := &types.GenesisState{
		Vkeys: []types.VKeyWithID{
			{
				Id: 50,
				Vkey: types.VKey{
					Name:        "genesis_key_50",
					Description: "Genesis key with ID 50",
					KeyBytes:    createTestVKeyBytes("genesis_key_50"),
				},
			},
			{
				Id: 100,
				Vkey: types.VKey{
					Name:        "genesis_key_100",
					Description: "Genesis key with ID 100",
					KeyBytes:    createTestVKeyBytes("genesis_key_100"),
				},
			},
		},
	}

	f.k.InitGenesis(f.ctx, gs)

	// Query next ID - should be 101 (one after highest genesis ID)
	resp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(101), resp.NextId)

	// Add a new key and verify it gets ID 101
	vkeyBytes := createTestVKeyBytes("new_key_after_genesis")
	newID, err := f.k.AddVKey(f.ctx, f.govModAddr, "new_key_after_genesis", vkeyBytes, "New key after genesis")
	require.NoError(t, err)
	require.Equal(t, uint64(101), newID)

	// Next ID should now be 102
	resp, err = f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(102), resp.NextId)
}

func TestQueryNextVKeyIDSequential(t *testing.T) {
	f := SetupTest(t)

	// Get initial next ID
	initialResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
	require.NoError(t, err)
	startID := initialResp.NextId

	// Add vkeys and verify IDs are sequential
	numVKeys := 10
	for i := 0; i < numVKeys; i++ {
		expectedID := startID + uint64(i)

		// Verify NextVKeyID returns expected ID
		resp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
		require.NoError(t, err)
		require.Equal(t, expectedID, resp.NextId)

		// Add vkey
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("seq_key%d", i))
		actualID, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("seq_key%d", i), vkeyBytes, fmt.Sprintf("Seq Key %d", i))
		require.NoError(t, err)
		require.Equal(t, expectedID, actualID)
	}

	// Final next ID should be startID + numVKeys
	finalResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
	require.NoError(t, err)
	require.Equal(t, startID+uint64(numVKeys), finalResp.NextId)
}
