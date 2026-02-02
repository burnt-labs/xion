package keeper_test

import (
	"encoding/base64"
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
func createTestVKeyBytes(_ string) []byte {
	vkeyJSON := map[string]interface{}{
		"protocol": "groth16",
		"curve":    "bn128",
		"nPublic":  88,
		"vk_alpha_1": []string{
			"20491192805390485299153009773594534940189261866228447918068658471970481763042",
			"9383485363053290200918347156157836566562967994039712273449902621266178545958",
			"1",
		},
		"vk_beta_2": [][]string{
			{
				"6375614351688725206403948262868962793625744043794305715222011528459656738731",
				"4252822878758300859123897981450591353533073413197771768651442665752259397132",
			},
			{
				"10505242626370262277552901082094356697409835680220590971873171140371331206856",
				"21847035105528745403288232691147584728191162732299865338377159692350059136679",
			},
			{
				"1",
				"0",
			},
		},
		"vk_gamma_2": [][]string{
			{
				"10857046999023057135944570762232829481370756359578518086990519993285655852781",
				"11559732032986387107991004021392285783925812861821192530917403151452391805634",
			},
			{
				"8495653923123431417604973247489272438418190587263600148770280649306958101930",
				"4082367875863433681332203403145435568316851327593401208105741076214120093531",
			},
			{
				"1",
				"0",
			},
		},
		"vk_delta_2": [][]string{
			{
				"17392078998052632343262307277610023502719756906213977655620834371545535983980",
				"21382662876262974785505842184655498074895683066652790606494907472138038564263",
			},
			{
				"19436381806731607522425069059625496157577116178088423901419643501207252678788",
				"3671213371011402617870213400215333594525403110203169640387998414321679236071",
			},
			{
				"1",
				"0",
			},
		},
		"vk_alphabeta_12": [][][]string{
			{
				{
					"2029413683389138792403550203267699914886160938906632433982220835551125967885",
					"21072700047562757817161031222997517981543347628379360635925549008442030252106",
				},
				{
					"5940354580057074848093997050200682056184807770593307860589430076672439820312",
					"12156638873931618554171829126792193045421052652279363021382169897324752428276",
				},
				{
					"7898200236362823042373859371574133993780991612861777490112507062703164551277",
					"7074218545237549455313236346927434013100842096812539264420499035217050630853",
				},
			},
			{
				{
					"7077479683546002997211712695946002074877511277312570035766170199895071832130",
					"10093483419865920389913245021038182291233451549023025229112148274109565435465",
				},
				{
					"4595479056700221319381530156280926371456704509942304414423590385166031118820",
					"19831328484489333784475432780421641293929726139240675179672856274388269393268",
				},
				{
					"11934129596455521040620786944827826205713621633706285934057045369193958244500",
					"8037395052364110730298837004334506829870972346962140206007064471173334027475",
				},
			},
		},
		"IC": [][]string{
			{
				"15950214535785561503685901885170416089208679241613946489241078754121531345368",
				"6413566741484414396601460146086570277052919216702699673845046940202545452712",
				"1",
			},
			{
				"16863092586897262433781345351732838892413475324733242808383514398789918257821",
				"2562926972726649639219893647901465073299431928632439273712843001810941814953",
				"1",
			},
			{
				"13068025862463096060363455359016956346236100286757263148377084234631824702773",
				"7737164861220347728330147009122685407385475738569408313096457178276957900800",
				"1",
			},
			{
				"4880502924625142882637218816341390548115741496372444113090118291337749751910",
				"20641211449938587878773621163821960030891368876242808289954092058040062388213",
				"1",
			},
			{
				"3862436174048768061564857650966663195191482254865068674374262483451806389520",
				"12402267693937063020550836252660463530985468695168451085487700610121047810951",
				"1",
			},
			{
				"13259997926828616484017101986753339350308361973527940017383814849072285699811",
				"10368672356259218839812719891499002458185249451833161710220630391964837445352",
				"1",
			},
			{
				"6397992820508516842241029094898516186725279286540672554116403160914176358443",
				"11195708633986858615714896781442268079030446466740337173653814606879461857562",
				"1",
			},
			{
				"16711396487645523624974699453994048850819233871032982904645504869858729525093",
				"16738482245786492820552168054381022310168489767902867788923385844539016279008",
				"1",
			},
			{
				"18549786394029224114545421187435575957887646018397541546865376163643142068707",
				"20052055145772687396769805410655970206023291130121819803408317971585826442303",
				"1",
			},
			{
				"9488244878405253066326398911806349476402820283916955909555088408451825216566",
				"14755712876564758833957923354105492023651788053211248807130155088310530601959",
				"1",
			},
			{
				"14373538572767172871472740985216352990517175912727836245348580109076322689434",
				"16551701169108805146510679977707499310951563344723188459880788910250556192396",
				"1",
			},
			{
				"21375041240635456273133847411338529361372641157826050078444226405949823351210",
				"3990701785499350931201013954804244253544512156381231628411657195236539105943",
				"1",
			},
			{
				"5392664266481004015100790636645876890061146810321246928426924028746789521386",
				"10749854649104785399017162103843486127753039621788468862658123655363901077181",
				"1",
			},
			{
				"9200155369151155561666333315183846584563973278547039380517493788730113043689",
				"3681808583763093472679418746401624806120862118346819377349076446950467066385",
				"1",
			},
			{
				"17530250430917406008626763464663663784525511871465031036020474163808056762909",
				"20509651622379631442933086061646947716120671868445047203776095938670039566844",
				"1",
			},
			{
				"19508957406013174741442608162562832887991931528284778091889421085900689127207",
				"10208609206252879202046625789094819803998748253603741410278686202757540798516",
				"1",
			},
			{
				"21686217944728010507586740480873321410941819443803188923658980698884670361261",
				"20943922276768364284451796397444156387069934574573272808623624956547122936735",
				"1",
			},
			{
				"20232461811568906700497150211824030936112074376197218907096759717503065821357",
				"15956506013354668088289770379353538815817784455167435761572956382456579635066",
				"1",
			},
			{
				"4045620713404972029797155330214849888029242650388151636874142954100049234692",
				"20523594583438851187849737856155340013317630404249704360690401323665466211571",
				"1",
			},
			{
				"14045008891890861821934552191744592053190701283630431621962332417749403017410",
				"12436059612224214699934430907774865184442829101183443713965971581594652198921",
				"1",
			},
			{
				"3653940122287964145237348199295673459992974603920669982495236791950070527626",
				"2577487249519353755769349964737706597907561017441061577131545294016853399202",
				"1",
			},
			{
				"9768531125480921709549417643253756436957790241271268561831249334077975552931",
				"18721409322580688507497216887702841485607828572976338424263073805186097780618",
				"1",
			},
			{
				"10385995135688946227255483643120865737554264560200280417726642622746663283640",
				"2500538262979246644639113090101849985701865785408589115890617441332273007320",
				"1",
			},
			{
				"11237527922926484074571415915644558055350198320930328841917242816906729921984",
				"9989064688222575788989699201390210701598360614811246368037826159811238215339",
				"1",
			},
			{
				"13602240994466841542593822177421187494135375480904595334407188100374738023279",
				"7979329358377217491130521234533807294864596933062464497366381252964663511911",
				"1",
			},
			{
				"1711092060580308217314239791744806143435492838955938811896790605109199849707",
				"6124000530364770163873726114714522226327235978456519872871971819440325839389",
				"1",
			},
			{
				"7799884938403008152118851385637692021433934443217070047257552408445155909530",
				"17995166091708851447356099736619972016980612736869467487878993961727582346305",
				"1",
			},
			{
				"6964723158213822055903020395775887935359512418864546589372028788008830715214",
				"10603142758860447408212637181621006651148702133594999090457906007787862741022",
				"1",
			},
			{
				"21585042147643659531478438797392857942466764344251065997734451941586050937147",
				"19206750508334847412700376800643503145601743211616957432602097915492480948284",
				"1",
			},
			{
				"19410826129283908627224600179433457093859454726250815431452936037403518129037",
				"21285690837378097546559937683636907348481424137844375530431400728616860751434",
				"1",
			},
			{
				"17632290282488082803101420074418252288083457791857528886767407390256669487402",
				"17601828943652226539740622761647617667562548009148250169032981750111383756903",
				"1",
			},
			{
				"3300800043068359501168906383251622915927319213819504738248596838440868491800",
				"9077020854714517489632578827611903104824643714690794811715918824746980034062",
				"1",
			},
			{
				"11539791592063540373565051781605532819743889181048325280516095215599278415284",
				"10770569126750218640631965776891557069905077747951579385656724402057932910688",
				"1",
			},
			{
				"13724223271717932146542793560988193267634645382104140426101652902821727110537",
				"4856436887855555488108070477685326609666056846400527879446335244723429476079",
				"1",
			},
			{
				"6241507949034324079800115382174737834566930959514884141318927681662216523943",
				"14393199679188906744606306198503537959676457383341152874934499775533009409606",
				"1",
			},
			{
				"13436319270221703070116084576472516502991447365941865176279504837843783543206",
				"12706323356935158720751079589081494910388874407711411527357229396762505544170",
				"1",
			},
			{
				"13598297511625991421031161846448598192312155440397910285175739915635261376057",
				"4426713382064984964487484322434624939479314861306325388834933222277300696381",
				"1",
			},
			{
				"4886370502050706893960406806786429893151412627823242468793057403822891056317",
				"5238856709761889474143528825087827219557312572431139070979300944579717464738",
				"1",
			},
			{
				"3134243413481044152526156202217921428466884823704454191944686448110742402342",
				"2599539807343292984997694185916996481169822480743916477712082096077653382969",
				"1",
			},
			{
				"6658643794409316793705291453949445823644555496249384507873716853332934668986",
				"13897297086987489580876479814537425419100995423569151661399608783311218870034",
				"1",
			},
			{
				"7639216203502438771749587700304160576480992048622128105295591906259506421984",
				"17901568302074394971546943159117851358859364213472838734249317970515017712332",
				"1",
			},
			{
				"3820430048383362344675685907365588952781700622139282165698846294285041050009",
				"14350371325690709315412811076968893225853472441680688955435272779527196071480",
				"1",
			},
			{
				"19741056833517215091019681519161084617255186700240895847113956533982894728044",
				"20284194974784615257288195801961634371687421809029601211109561213653381765343",
				"1",
			},
			{
				"7017169041508795614997911671897895339113193275820386036139280849575223455037",
				"11982810763519820597490797383127300808367511979686315546850445487989939786311",
				"1",
			},
			{
				"12414214262317235387766149426150079811656594287116042146646069084014600463564",
				"4467060607047064647048017348630058323081461935891076860278872208792869989541",
				"1",
			},
			{
				"14854719023038286333789214372998949285345259897348664731891459878596867915238",
				"21701895600760900652355401503873236516153623881639962228982145813293800165201",
				"1",
			},
			{
				"21467378938829605952320861456896682628734522207637254136870777587403196542707",
				"3323265524796620136486855393056291075067202206846266874550487930548803655889",
				"1",
			},
			{
				"1160354600852200592162660540201026590538244527515479441048140727478123678204",
				"10012930608119980354319250519856965362060251050066875588508607282136339410705",
				"1",
			},
			{
				"3583661491681735379690040574159705160493728586284550050052447927015896894459",
				"6003897836691435561922619171486538258987082170798801117795921005035674695745",
				"1",
			},
			{
				"11095556631362669461564420426853498980513453089428341728093742633312770162365",
				"6794271493907316768019285494104898486495396104699623580464075592380779089476",
				"1",
			},
			{
				"21666696947872480725445170872389123738667879006506166192433247345022644440734",
				"17493522665437407217479491711496237384680993522912159838803918309801075697925",
				"1",
			},
			{
				"5485673981926661082743526956233627869185248827866081723720465569764609233721",
				"4045083751858422052100717317149711478118055660858644263502264172717098386032",
				"1",
			},
			{
				"12155266854873373714579469422317219181806026601525137556258438418734246988116",
				"11797233548915192896028636876234557620060680507586158982666928251642539305423",
				"1",
			},
			{
				"16583520184362063182677074368201636605199859497032982479247000667311864104375",
				"10506907758813719094737636583503385800753433021331248728651677214258004088559",
				"1",
			},
			{
				"3712508399833140998060963899280362581681022870963525242397161522990742154646",
				"16280723390252623514700101989253312790264168456963172502708592074310968549591",
				"1",
			},
			{
				"17868698875581175628137099605490246473243401999252406999511512247460002901021",
				"1649168920263494223861701097863895560224608849426978375751850303377673819172",
				"1",
			},
			{
				"6810657939168348880087204018066475625605806147563193070841516632644765868480",
				"17861791586059032352097327375385385624473273394109074701894471379978074040784",
				"1",
			},
			{
				"4620016759046441678773492701523809923461409760417558746453181041208821583418",
				"4612521404064736144689514026357777275115244068514713805129035664178213979554",
				"1",
			},
			{
				"18771002108786903247959537255031401956476977840015721421148261463190095851113",
				"498886275661823072287718704049949258400991209968727350252554523778105782215",
				"1",
			},
			{
				"15326013501770747921174775482478715930743514297654100923596902576575671816849",
				"2883308288996168085983065367681663502068338699538440275816275978410323328406",
				"1",
			},
			{
				"21581684913076799209490557818483260110296790332918771795348816669651847000679",
				"19255873940850804451783260286343088032937552756774189816713586265529831908504",
				"1",
			},
			{
				"9422261690912571173986303028522296342606395637942219379666652619903201019131",
				"13157297488865573001303139061647034843599036423019952497061432740799741656842",
				"1",
			},
			{
				"1514157775351832733851600116508176403078970867542532240445777211872550685122",
				"10098819081512815519980061805077722208902467184247461515351240331734854518381",
				"1",
			},
			{
				"11782815250364193986475389178853147568516472982130840457112491737559788955101",
				"3335902146830715656021749998239646474827407353582699596885331049435108953201",
				"1",
			},
			{
				"19715141303756368931876618652294034157902362015546362095456319804205226826133",
				"13940583470143867039418224929910049911006893023789574099286250232307258688711",
				"1",
			},
			{
				"8716744272985801264882445926416318716002742018795335350713831221043480580032",
				"5981267136782955250495545666218546733118820751848196041477454671263163558897",
				"1",
			},
			{
				"21628952821580586497929833355825274220202057128991040228930331708145040660855",
				"3469516323364817710854129570092268160069632978883031941255303021154065607922",
				"1",
			},
			{
				"17501861154724193589590506891062483213646232948920678675545153811813856128573",
				"20254238639286400751749247937175107725010995373011052127004348867729068873966",
				"1",
			},
			{
				"8442995010652504441784521577785851339669684005049314495578925421517035610011",
				"15627528014249992938080566822784668676708451534871137221203003405263205622292",
				"1",
			},
			{
				"19414400530135975385492231648570111940243147307766181195188375717332269327959",
				"1499364253014807765434847446251511366276069422266546393509394742672423498642",
				"1",
			},
			{
				"9161864863925961293845383820344598372665960579564678665694408576515214580204",
				"13406476698756762066901676887188803910954348623494044661390967444811064032830",
				"1",
			},
			{
				"7008531358837212090350935476929706988333478235531943394907033017977192860610",
				"7774424843983567179012169757171980747968105070613972208975551310700117771542",
				"1",
			},
			{
				"14578588366366136567572216886935469954382916858356947811454442710467695870366",
				"17231497063137231274856192560145970477655535850141461878515723965739614607035",
				"1",
			},
			{
				"4319881357156202808731141185602438275838823213614732720325977789014374291149",
				"14351127588947523773520193838214020706347297025902615585223544665583857923393",
				"1",
			},
			{
				"18192254486173626791927949654743431767586920040690298573321190032488684795658",
				"6072114391506802460909542442713589973786175258256944676453994774586144868971",
				"1",
			},
			{
				"3798425761531572380130880687418099092838859498323200122859781833653604156767",
				"7084161216630305433056954952704057959691436560540571492579268107886758282590",
				"1",
			},
			{
				"12221482081437000836203647300597080175031889687813472200014154223123836620199",
				"20968736189291315653908935342644552914098668187863638497385415943421228733058",
				"1",
			},
			{
				"617400237964230416228804192601882940521456716046054781850543477724806091311",
				"2904398655777212823206942900445971875646587821051609522215605193631325136260",
				"1",
			},
			{
				"2630283627321778633778673545048506457244227283376747634599863527799915022425",
				"940513164069974613974034762133339181649727449337769637272141800475219688707",
				"1",
			},
			{
				"18163405159728587410831224788431815807681099737058317492533922873291080064040",
				"9031215561364014341835611435935312647306825311699176403239210668640380631758",
				"1",
			},
			{
				"1737647792914177177630507239053383779243024285042606473063461958504284885437",
				"8578877333890370354788498261638161376531459669187328779551688892176587772921",
				"1",
			},
			{
				"12354002330358693886969982374034740922082764644395640762532220758019139198399",
				"19575157564538221508252074540975523997474775427079762157651825427443387741334",
				"1",
			},
			{
				"18032566659806712997656749275422220696173189796125826963827833268809202086322",
				"16946040601798695321043126656806137519263966567822907008292341347796760038300",
				"1",
			},
			{
				"21427739805154086213649306614743044371728499524515544239186979041239237442117",
				"8923985632057383012782601268197398502317544291061078307272989411673950351474",
				"1",
			},
			{
				"7081324789641261935825569660606618609091430934854329039157458662925381113109",
				"21565060637886002596108024486962876241946928759218624744800835274778545826453",
				"1",
			},
			{
				"9474074458682123852946953048882083153831215474062305951197139598855191369887",
				"17076595853287865670607387197085589417258936334498202048370392739025133484947",
				"1",
			},
			{
				"5281705668899767051116648812837006764258612940205033154396824669172615411374",
				"21610060531524118652658079655974381794228027531596545964243289489247182860243",
				"1",
			},
			{
				"9255585742697107898396540824748617249796675837262778594710205251423547909391",
				"21252311345958756529843826782458097494903171861113489067518450694281638753940",
				"1",
			},
			{
				"20911406283497461651801969177668183817390375840777623031021125474041622540287",
				"10275270587102628971535520879400364640005486064321744709600223644102588379687",
				"1",
			},
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

func TestGetAuthority(t *testing.T) {
	f := SetupTest(t)

	authority := f.k.GetAuthority()
	require.NotEmpty(t, authority)
	require.Equal(t, f.govModAddr, authority)
}

func TestGetCodec(t *testing.T) {
	f := SetupTest(t)

	codec := f.k.GetCodec()
	require.NotNil(t, codec)
}

// ============================================================================
// Genesis Tests
// ============================================================================

func TestInitGenesis(t *testing.T) {
	t.Run("empty genesis state", func(t *testing.T) {
		f := SetupTest(t)

		gs := &types.GenesisState{
			Vkeys: []types.VKeyWithID{},
		}

		// Should not panic
		require.NotPanics(t, func() {
			f.k.InitGenesis(f.ctx, gs)
		})

		// Verify no vkeys exist
		vkeys, err := f.k.ListVKeys(f.ctx)
		require.NoError(t, err)
		require.Empty(t, vkeys)
	})

	t.Run("genesis with single vkey", func(t *testing.T) {
		f := SetupTest(t)

		vkeyBytes := createTestVKeyBytes("test_key")
		gs := &types.GenesisState{
			Vkeys: []types.VKeyWithID{
				{
					Id: 1,
					Vkey: types.VKey{
						Name:        "test_key",
						Description: "Test verification key",
						KeyBytes:    vkeyBytes,
					},
				},
			},
		}

		require.NotPanics(t, func() {
			f.k.InitGenesis(f.ctx, gs)
		})

		// Verify vkey exists
		vkey, err := f.k.GetVKeyByID(f.ctx, 1)
		require.NoError(t, err)
		require.Equal(t, "test_key", vkey.Name)
		require.Equal(t, "Test verification key", vkey.Description)

		// Verify name index works
		vkeyByName, err := f.k.GetVKeyByName(f.ctx, "test_key")
		require.NoError(t, err)
		require.Equal(t, vkey.Name, vkeyByName.Name)
	})

	t.Run("genesis with multiple vkeys", func(t *testing.T) {
		f := SetupTest(t)

		gs := &types.GenesisState{
			Vkeys: []types.VKeyWithID{
				{
					Id: 1,
					Vkey: types.VKey{
						Name:        "key1",
						Description: "First key",
						KeyBytes:    createTestVKeyBytes("key1"),
					},
				},
				{
					Id: 2,
					Vkey: types.VKey{
						Name:        "key2",
						Description: "Second key",
						KeyBytes:    createTestVKeyBytes("key2"),
					},
				},
				{
					Id: 5,
					Vkey: types.VKey{
						Name:        "key5",
						Description: "Fifth key (gap in IDs)",
						KeyBytes:    createTestVKeyBytes("key5"),
					},
				},
			},
		}

		require.NotPanics(t, func() {
			f.k.InitGenesis(f.ctx, gs)
		})

		// Verify all vkeys exist
		vkeys, err := f.k.ListVKeys(f.ctx)
		require.NoError(t, err)
		require.Len(t, vkeys, 3)

		// Verify each key by ID
		vkey1, err := f.k.GetVKeyByID(f.ctx, 1)
		require.NoError(t, err)
		require.Equal(t, "key1", vkey1.Name)

		vkey2, err := f.k.GetVKeyByID(f.ctx, 2)
		require.NoError(t, err)
		require.Equal(t, "key2", vkey2.Name)

		vkey5, err := f.k.GetVKeyByID(f.ctx, 5)
		require.NoError(t, err)
		require.Equal(t, "key5", vkey5.Name)

		// Verify sequence is set correctly (should be 6, one after highest ID)
		nextID, err := f.k.NextVKeyID.Peek(f.ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(6), nextID)
	})

	t.Run("genesis preserves sequence after highest ID", func(t *testing.T) {
		f := SetupTest(t)

		gs := &types.GenesisState{
			Vkeys: []types.VKeyWithID{
				{
					Id: 100,
					Vkey: types.VKey{
						Name:        "high_id_key",
						Description: "Key with high ID",
						KeyBytes:    createTestVKeyBytes("high_id_key"),
					},
				},
			},
		}

		f.k.InitGenesis(f.ctx, gs)

		// Verify sequence is set to 101
		nextID, err := f.k.NextVKeyID.Peek(f.ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(101), nextID)

		// Add a new key and verify it gets ID 101
		newVKeyBytes := createTestVKeyBytes("new_key")
		newID, err := f.k.AddVKey(f.ctx, f.govModAddr, "new_key", newVKeyBytes, "New key")
		require.NoError(t, err)
		require.Equal(t, uint64(101), newID)
	})
}

func TestExportGenesis(t *testing.T) {
	t.Run("export empty genesis", func(t *testing.T) {
		f := SetupTest(t)

		gs := f.k.ExportGenesis(f.ctx)
		require.NotNil(t, gs)
		require.Empty(t, gs.Vkeys)
	})

	t.Run("export genesis with vkeys", func(t *testing.T) {
		f := SetupTest(t)

		// Add some vkeys
		vkey1Bytes := createTestVKeyBytes("key1")
		id1, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "First key")
		require.NoError(t, err)

		vkey2Bytes := createTestVKeyBytes("key2")
		id2, err := f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Second key")
		require.NoError(t, err)

		// Export genesis
		gs := f.k.ExportGenesis(f.ctx)
		require.NotNil(t, gs)
		require.Len(t, gs.Vkeys, 2)

		// Verify exported vkeys contain correct data
		exportedIDs := make(map[uint64]types.VKey)
		for _, vkeyWithID := range gs.Vkeys {
			exportedIDs[vkeyWithID.Id] = vkeyWithID.Vkey
		}

		require.Contains(t, exportedIDs, id1)
		require.Equal(t, "key1", exportedIDs[id1].Name)
		require.Equal(t, "First key", exportedIDs[id1].Description)

		require.Contains(t, exportedIDs, id2)
		require.Equal(t, "key2", exportedIDs[id2].Name)
		require.Equal(t, "Second key", exportedIDs[id2].Description)
	})

	t.Run("export then import genesis roundtrip", func(t *testing.T) {
		f := SetupTest(t)

		// Add vkeys
		vkey1Bytes := createTestVKeyBytes("key1")
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "First key")
		require.NoError(t, err)

		vkey2Bytes := createTestVKeyBytes("key2")
		_, err = f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Second key")
		require.NoError(t, err)

		// Export genesis
		exportedGS := f.k.ExportGenesis(f.ctx)

		// Create new fixture and import
		f2 := SetupTest(t)
		f2.k.InitGenesis(f2.ctx, exportedGS)

		// Verify imported state matches
		vkeys, err := f2.k.ListVKeys(f2.ctx)
		require.NoError(t, err)
		require.Len(t, vkeys, 2)

		// Verify by name
		key1, err := f2.k.GetVKeyByName(f2.ctx, "key1")
		require.NoError(t, err)
		require.Equal(t, "First key", key1.Description)

		key2, err := f2.k.GetVKeyByName(f2.ctx, "key2")
		require.NoError(t, err)
		require.Equal(t, "Second key", key2.Description)
	})
}

// ============================================================================
// IterateVKeys Tests
// ============================================================================

func TestIterateVKeys(t *testing.T) {
	t.Run("iterate empty store", func(t *testing.T) {
		f := SetupTest(t)

		count := 0
		err := f.k.IterateVKeys(f.ctx, func(id uint64, vkey types.VKey) (bool, error) {
			count++
			return false, nil
		})
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("iterate all vkeys", func(t *testing.T) {
		f := SetupTest(t)

		// Add multiple vkeys
		for i := 0; i < 5; i++ {
			vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
			_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
			require.NoError(t, err)
		}

		// Iterate and collect
		var collectedVKeys []types.VKey
		var collectedIDs []uint64
		err := f.k.IterateVKeys(f.ctx, func(id uint64, vkey types.VKey) (bool, error) {
			collectedIDs = append(collectedIDs, id)
			collectedVKeys = append(collectedVKeys, vkey)
			return false, nil
		})
		require.NoError(t, err)
		require.Len(t, collectedVKeys, 5)
		require.Len(t, collectedIDs, 5)

		// Verify all keys were collected
		names := make(map[string]bool)
		for _, vkey := range collectedVKeys {
			names[vkey.Name] = true
		}
		for i := 0; i < 5; i++ {
			require.True(t, names[fmt.Sprintf("key%d", i)])
		}
	})

	t.Run("iterate with early stop", func(t *testing.T) {
		f := SetupTest(t)

		// Add multiple vkeys
		for i := 0; i < 10; i++ {
			vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
			_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
			require.NoError(t, err)
		}

		// Iterate but stop after 3
		count := 0
		err := f.k.IterateVKeys(f.ctx, func(id uint64, vkey types.VKey) (bool, error) {
			count++
			if count >= 3 {
				return true, nil // Stop iteration
			}
			return false, nil
		})
		require.NoError(t, err)
		require.Equal(t, 3, count)
	})

	t.Run("iterate with error in callback", func(t *testing.T) {
		f := SetupTest(t)

		// Add a vkey
		vkeyBytes := createTestVKeyBytes("key1")
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkeyBytes, "Key 1")
		require.NoError(t, err)

		// Iterate with error
		expectedErr := fmt.Errorf("test error")
		err = f.k.IterateVKeys(f.ctx, func(id uint64, vkey types.VKey) (bool, error) {
			return false, expectedErr
		})
		require.Error(t, err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("iterate collects IDs correctly", func(t *testing.T) {
		f := SetupTest(t)

		// Add vkeys and track their IDs
		expectedIDs := make(map[uint64]string)
		for i := 0; i < 3; i++ {
			name := fmt.Sprintf("key%d", i)
			vkeyBytes := createTestVKeyBytes(name)
			id, err := f.k.AddVKey(f.ctx, f.govModAddr, name, vkeyBytes, fmt.Sprintf("Key %d", i))
			require.NoError(t, err)
			expectedIDs[id] = name
		}

		// Iterate and verify IDs match names
		err := f.k.IterateVKeys(f.ctx, func(id uint64, vkey types.VKey) (bool, error) {
			expectedName, ok := expectedIDs[id]
			require.True(t, ok, "unexpected ID: %d", id)
			require.Equal(t, expectedName, vkey.Name)
			return false, nil
		})
		require.NoError(t, err)
	})
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
			name:        "successfully add with non-governance authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "user_added",
			vkeyBytes:   createTestVKeyBytes("user_added"),
			description: "Added by user account",
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
	require.Equal(t, 88, circomVKey.NPublic)
}

func TestGetCircomVKeyByID(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test key")
	require.NoError(t, err)

	t.Run("successfully get circom vkey by ID", func(t *testing.T) {
		circomVKey, err := f.k.GetCircomVKeyByID(f.ctx, id)
		require.NoError(t, err)
		require.NotNil(t, circomVKey)
		require.Equal(t, "groth16", circomVKey.Protocol)
		require.Equal(t, "bn128", circomVKey.Curve)
	})

	t.Run("fail to get non-existent circom vkey by ID", func(t *testing.T) {
		_, err := f.k.GetCircomVKeyByID(f.ctx, 9999)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
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
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Original description")
	require.NoError(t, err)

	// Add user-owned vkey
	userVkeyBytes := createTestVKeyBytes("user_auth")
	_, err = f.k.AddVKey(f.ctx, f.addrs[0].String(), "user_auth", userVkeyBytes, "User description")
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
			name:        "successfully update with uploader authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "user_auth",
			newBytes:    createTestVKeyBytes("user_auth"),
			description: "User updated description",
			expectError: false,
		},
		{
			name:        "fail with mismatched authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "email_auth",
			newBytes:    createTestVKeyBytes("email_auth"),
			description: "User updated description",
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
				updated, err := f.k.GetVKeyByName(f.ctx, tt.vkeyName)
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

	userKeyBytes := createTestVKeyBytes("user_key")
	_, err = f.k.AddVKey(f.ctx, f.addrs[0].String(), "user_key", userKeyBytes, "User key")
	require.NoError(t, err)

	tests := []struct {
		name        string
		authority   string
		vkeyName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully remove vkey",
			authority:   f.govModAddr,
			vkeyName:    "key1",
			expectError: false,
		},
		{
			name:        "successfully remove with uploader authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "user_key",
			expectError: false,
		},
		{
			name:        "fail with mismatched authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "key2",
			expectError: true,
			errorMsg:    "invalid authority",
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
	require.Equal(t, 88, circomVKey.NPublic)
	require.Len(t, circomVKey.VkAlpha1, 3)
	require.Len(t, circomVKey.IC, 89)
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
	require.Equal(t, 88, circomVKey.NPublic)
	require.Len(t, circomVKey.IC, 89) // 88 public inputs + 1
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
			errorMsg:    "invalid verification key",
		},
		{
			name:        "invalid json",
			data:        []byte(base64.StdEncoding.EncodeToString([]byte("not json"))),
			expectError: true,
			errorMsg:    "invalid verification key",
		},
		{
			name: "missing required fields",
			data: []byte(base64.StdEncoding.EncodeToString([]byte(`{
				"protocol": "groth16",
				"curve": "bn128"
			}`))),
			expectError: true,
			errorMsg:    "invalid verification key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateVKeyBytes(tt.data, 0)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
