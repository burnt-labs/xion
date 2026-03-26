// keeper/query_server_test.go
package keeper_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/burnt-labs/barretenberg-go/barretenberg"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/zk/types"
)

// Define the proof data here to be used in the test
var proofData = []byte(`{
        "pi_a": [
            "5583158245518012202854967966688803983422579480975771799159435109682404412144",
            "19132509617989255559927911185942768582713778613503304661723852230698387114840",
            "1"
        ],
        "pi_b": [
            [
                "16209151427684011206863591092531391562117041646748639896310737311173246509260",
                "17729357182912272387117349263688449009610186531485947940640482832772517448927"
            ],
            [
                "5695516600618485685754260649529465903248888152110855008128547397403792546988",
                "656772577582924627058107331850692187484072991458347712020152128940322124285"
            ],
            [
                "1",
                "0"
            ]
        ],
        "pi_c": [
            "17453897224382172288517505191435866511305436208311355514241444398256793953872",
            "9163422778422181829456976190497942172380575625369266408413936273192580460236",
            "1"
        ],
        "protocol": "groth16"
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
	"2247300589898352470465977834393385125952162613548803856347250930590848545974",
	"0",
	"191581113848055322477272311147821680130451026496941019613909483584263833445",
	"149108628584424258332964971884436592255105616775526759101383287099246929273",
	"20356082004311139738363494460884070443445370694676839",
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
	"0",
	"0",
	"9079378704521501721378444251561135763203091338587747860525949554473799137061",
	"1",
	"145464208130933216679374873468710647147",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
	"180980592328871182281563474567090989367752380861661653173671556731952063826",
	"172098319462167245787450256346327983900299348343399089472063709653278487145",
	"112992901528165214978731108",
	"0",
	"0",
	"0",
	"0",
	"0",
	"0",
}

// Define the vkey here to be used in the test
var vkeyJSON = []byte(`{
  "protocol": "groth16",
  "curve": "bn128",
  "nPublic": 88,
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
    "17392078998052632343262307277610023502719756906213977655620834371545535983980",
    "21382662876262974785505842184655498074895683066652790606494907472138038564263"
   ],
   [
    "19436381806731607522425069059625496157577116178088423901419643501207252678788",
    "3671213371011402617870213400215333594525403110203169640387998414321679236071"
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
    "15950214535785561503685901885170416089208679241613946489241078754121531345368",
    "6413566741484414396601460146086570277052919216702699673845046940202545452712",
    "1"
   ],
   [
    "16863092586897262433781345351732838892413475324733242808383514398789918257821",
    "2562926972726649639219893647901465073299431928632439273712843001810941814953",
    "1"
   ],
   [
    "13068025862463096060363455359016956346236100286757263148377084234631824702773",
    "7737164861220347728330147009122685407385475738569408313096457178276957900800",
    "1"
   ],
   [
    "4880502924625142882637218816341390548115741496372444113090118291337749751910",
    "20641211449938587878773621163821960030891368876242808289954092058040062388213",
    "1"
   ],
   [
    "3862436174048768061564857650966663195191482254865068674374262483451806389520",
    "12402267693937063020550836252660463530985468695168451085487700610121047810951",
    "1"
   ],
   [
    "13259997926828616484017101986753339350308361973527940017383814849072285699811",
    "10368672356259218839812719891499002458185249451833161710220630391964837445352",
    "1"
   ],
   [
    "6397992820508516842241029094898516186725279286540672554116403160914176358443",
    "11195708633986858615714896781442268079030446466740337173653814606879461857562",
    "1"
   ],
   [
    "16711396487645523624974699453994048850819233871032982904645504869858729525093",
    "16738482245786492820552168054381022310168489767902867788923385844539016279008",
    "1"
   ],
   [
    "18549786394029224114545421187435575957887646018397541546865376163643142068707",
    "20052055145772687396769805410655970206023291130121819803408317971585826442303",
    "1"
   ],
   [
    "9488244878405253066326398911806349476402820283916955909555088408451825216566",
    "14755712876564758833957923354105492023651788053211248807130155088310530601959",
    "1"
   ],
   [
    "14373538572767172871472740985216352990517175912727836245348580109076322689434",
    "16551701169108805146510679977707499310951563344723188459880788910250556192396",
    "1"
   ],
   [
    "21375041240635456273133847411338529361372641157826050078444226405949823351210",
    "3990701785499350931201013954804244253544512156381231628411657195236539105943",
    "1"
   ],
   [
    "5392664266481004015100790636645876890061146810321246928426924028746789521386",
    "10749854649104785399017162103843486127753039621788468862658123655363901077181",
    "1"
   ],
   [
    "9200155369151155561666333315183846584563973278547039380517493788730113043689",
    "3681808583763093472679418746401624806120862118346819377349076446950467066385",
    "1"
   ],
   [
    "17530250430917406008626763464663663784525511871465031036020474163808056762909",
    "20509651622379631442933086061646947716120671868445047203776095938670039566844",
    "1"
   ],
   [
    "19508957406013174741442608162562832887991931528284778091889421085900689127207",
    "10208609206252879202046625789094819803998748253603741410278686202757540798516",
    "1"
   ],
   [
    "21686217944728010507586740480873321410941819443803188923658980698884670361261",
    "20943922276768364284451796397444156387069934574573272808623624956547122936735",
    "1"
   ],
   [
    "20232461811568906700497150211824030936112074376197218907096759717503065821357",
    "15956506013354668088289770379353538815817784455167435761572956382456579635066",
    "1"
   ],
   [
    "4045620713404972029797155330214849888029242650388151636874142954100049234692",
    "20523594583438851187849737856155340013317630404249704360690401323665466211571",
    "1"
   ],
   [
    "14045008891890861821934552191744592053190701283630431621962332417749403017410",
    "12436059612224214699934430907774865184442829101183443713965971581594652198921",
    "1"
   ],
   [
    "3653940122287964145237348199295673459992974603920669982495236791950070527626",
    "2577487249519353755769349964737706597907561017441061577131545294016853399202",
    "1"
   ],
   [
    "9768531125480921709549417643253756436957790241271268561831249334077975552931",
    "18721409322580688507497216887702841485607828572976338424263073805186097780618",
    "1"
   ],
   [
    "10385995135688946227255483643120865737554264560200280417726642622746663283640",
    "2500538262979246644639113090101849985701865785408589115890617441332273007320",
    "1"
   ],
   [
    "11237527922926484074571415915644558055350198320930328841917242816906729921984",
    "9989064688222575788989699201390210701598360614811246368037826159811238215339",
    "1"
   ],
   [
    "13602240994466841542593822177421187494135375480904595334407188100374738023279",
    "7979329358377217491130521234533807294864596933062464497366381252964663511911",
    "1"
   ],
   [
    "1711092060580308217314239791744806143435492838955938811896790605109199849707",
    "6124000530364770163873726114714522226327235978456519872871971819440325839389",
    "1"
   ],
   [
    "7799884938403008152118851385637692021433934443217070047257552408445155909530",
    "17995166091708851447356099736619972016980612736869467487878993961727582346305",
    "1"
   ],
   [
    "6964723158213822055903020395775887935359512418864546589372028788008830715214",
    "10603142758860447408212637181621006651148702133594999090457906007787862741022",
    "1"
   ],
   [
    "21585042147643659531478438797392857942466764344251065997734451941586050937147",
    "19206750508334847412700376800643503145601743211616957432602097915492480948284",
    "1"
   ],
   [
    "19410826129283908627224600179433457093859454726250815431452936037403518129037",
    "21285690837378097546559937683636907348481424137844375530431400728616860751434",
    "1"
   ],
   [
    "17632290282488082803101420074418252288083457791857528886767407390256669487402",
    "17601828943652226539740622761647617667562548009148250169032981750111383756903",
    "1"
   ],
   [
    "3300800043068359501168906383251622915927319213819504738248596838440868491800",
    "9077020854714517489632578827611903104824643714690794811715918824746980034062",
    "1"
   ],
   [
    "11539791592063540373565051781605532819743889181048325280516095215599278415284",
    "10770569126750218640631965776891557069905077747951579385656724402057932910688",
    "1"
   ],
   [
    "13724223271717932146542793560988193267634645382104140426101652902821727110537",
    "4856436887855555488108070477685326609666056846400527879446335244723429476079",
    "1"
   ],
   [
    "6241507949034324079800115382174737834566930959514884141318927681662216523943",
    "14393199679188906744606306198503537959676457383341152874934499775533009409606",
    "1"
   ],
   [
    "13436319270221703070116084576472516502991447365941865176279504837843783543206",
    "12706323356935158720751079589081494910388874407711411527357229396762505544170",
    "1"
   ],
   [
    "13598297511625991421031161846448598192312155440397910285175739915635261376057",
    "4426713382064984964487484322434624939479314861306325388834933222277300696381",
    "1"
   ],
   [
    "4886370502050706893960406806786429893151412627823242468793057403822891056317",
    "5238856709761889474143528825087827219557312572431139070979300944579717464738",
    "1"
   ],
   [
    "3134243413481044152526156202217921428466884823704454191944686448110742402342",
    "2599539807343292984997694185916996481169822480743916477712082096077653382969",
    "1"
   ],
   [
    "6658643794409316793705291453949445823644555496249384507873716853332934668986",
    "13897297086987489580876479814537425419100995423569151661399608783311218870034",
    "1"
   ],
   [
    "7639216203502438771749587700304160576480992048622128105295591906259506421984",
    "17901568302074394971546943159117851358859364213472838734249317970515017712332",
    "1"
   ],
   [
    "3820430048383362344675685907365588952781700622139282165698846294285041050009",
    "14350371325690709315412811076968893225853472441680688955435272779527196071480",
    "1"
   ],
   [
    "19741056833517215091019681519161084617255186700240895847113956533982894728044",
    "20284194974784615257288195801961634371687421809029601211109561213653381765343",
    "1"
   ],
   [
    "7017169041508795614997911671897895339113193275820386036139280849575223455037",
    "11982810763519820597490797383127300808367511979686315546850445487989939786311",
    "1"
   ],
   [
    "12414214262317235387766149426150079811656594287116042146646069084014600463564",
    "4467060607047064647048017348630058323081461935891076860278872208792869989541",
    "1"
   ],
   [
    "14854719023038286333789214372998949285345259897348664731891459878596867915238",
    "21701895600760900652355401503873236516153623881639962228982145813293800165201",
    "1"
   ],
   [
    "21467378938829605952320861456896682628734522207637254136870777587403196542707",
    "3323265524796620136486855393056291075067202206846266874550487930548803655889",
    "1"
   ],
   [
    "1160354600852200592162660540201026590538244527515479441048140727478123678204",
    "10012930608119980354319250519856965362060251050066875588508607282136339410705",
    "1"
   ],
   [
    "3583661491681735379690040574159705160493728586284550050052447927015896894459",
    "6003897836691435561922619171486538258987082170798801117795921005035674695745",
    "1"
   ],
   [
    "11095556631362669461564420426853498980513453089428341728093742633312770162365",
    "6794271493907316768019285494104898486495396104699623580464075592380779089476",
    "1"
   ],
   [
    "21666696947872480725445170872389123738667879006506166192433247345022644440734",
    "17493522665437407217479491711496237384680993522912159838803918309801075697925",
    "1"
   ],
   [
    "5485673981926661082743526956233627869185248827866081723720465569764609233721",
    "4045083751858422052100717317149711478118055660858644263502264172717098386032",
    "1"
   ],
   [
    "12155266854873373714579469422317219181806026601525137556258438418734246988116",
    "11797233548915192896028636876234557620060680507586158982666928251642539305423",
    "1"
   ],
   [
    "16583520184362063182677074368201636605199859497032982479247000667311864104375",
    "10506907758813719094737636583503385800753433021331248728651677214258004088559",
    "1"
   ],
   [
    "3712508399833140998060963899280362581681022870963525242397161522990742154646",
    "16280723390252623514700101989253312790264168456963172502708592074310968549591",
    "1"
   ],
   [
    "17868698875581175628137099605490246473243401999252406999511512247460002901021",
    "1649168920263494223861701097863895560224608849426978375751850303377673819172",
    "1"
   ],
   [
    "6810657939168348880087204018066475625605806147563193070841516632644765868480",
    "17861791586059032352097327375385385624473273394109074701894471379978074040784",
    "1"
   ],
   [
    "4620016759046441678773492701523809923461409760417558746453181041208821583418",
    "4612521404064736144689514026357777275115244068514713805129035664178213979554",
    "1"
   ],
   [
    "18771002108786903247959537255031401956476977840015721421148261463190095851113",
    "498886275661823072287718704049949258400991209968727350252554523778105782215",
    "1"
   ],
   [
    "15326013501770747921174775482478715930743514297654100923596902576575671816849",
    "2883308288996168085983065367681663502068338699538440275816275978410323328406",
    "1"
   ],
   [
    "21581684913076799209490557818483260110296790332918771795348816669651847000679",
    "19255873940850804451783260286343088032937552756774189816713586265529831908504",
    "1"
   ],
   [
    "9422261690912571173986303028522296342606395637942219379666652619903201019131",
    "13157297488865573001303139061647034843599036423019952497061432740799741656842",
    "1"
   ],
   [
    "1514157775351832733851600116508176403078970867542532240445777211872550685122",
    "10098819081512815519980061805077722208902467184247461515351240331734854518381",
    "1"
   ],
   [
    "11782815250364193986475389178853147568516472982130840457112491737559788955101",
    "3335902146830715656021749998239646474827407353582699596885331049435108953201",
    "1"
   ],
   [
    "19715141303756368931876618652294034157902362015546362095456319804205226826133",
    "13940583470143867039418224929910049911006893023789574099286250232307258688711",
    "1"
   ],
   [
    "8716744272985801264882445926416318716002742018795335350713831221043480580032",
    "5981267136782955250495545666218546733118820751848196041477454671263163558897",
    "1"
   ],
   [
    "21628952821580586497929833355825274220202057128991040228930331708145040660855",
    "3469516323364817710854129570092268160069632978883031941255303021154065607922",
    "1"
   ],
   [
    "17501861154724193589590506891062483213646232948920678675545153811813856128573",
    "20254238639286400751749247937175107725010995373011052127004348867729068873966",
    "1"
   ],
   [
    "8442995010652504441784521577785851339669684005049314495578925421517035610011",
    "15627528014249992938080566822784668676708451534871137221203003405263205622292",
    "1"
   ],
   [
    "19414400530135975385492231648570111940243147307766181195188375717332269327959",
    "1499364253014807765434847446251511366276069422266546393509394742672423498642",
    "1"
   ],
   [
    "9161864863925961293845383820344598372665960579564678665694408576515214580204",
    "13406476698756762066901676887188803910954348623494044661390967444811064032830",
    "1"
   ],
   [
    "7008531358837212090350935476929706988333478235531943394907033017977192860610",
    "7774424843983567179012169757171980747968105070613972208975551310700117771542",
    "1"
   ],
   [
    "14578588366366136567572216886935469954382916858356947811454442710467695870366",
    "17231497063137231274856192560145970477655535850141461878515723965739614607035",
    "1"
   ],
   [
    "4319881357156202808731141185602438275838823213614732720325977789014374291149",
    "14351127588947523773520193838214020706347297025902615585223544665583857923393",
    "1"
   ],
   [
    "18192254486173626791927949654743431767586920040690298573321190032488684795658",
    "6072114391506802460909542442713589973786175258256944676453994774586144868971",
    "1"
   ],
   [
    "3798425761531572380130880687418099092838859498323200122859781833653604156767",
    "7084161216630305433056954952704057959691436560540571492579268107886758282590",
    "1"
   ],
   [
    "12221482081437000836203647300597080175031889687813472200014154223123836620199",
    "20968736189291315653908935342644552914098668187863638497385415943421228733058",
    "1"
   ],
   [
    "617400237964230416228804192601882940521456716046054781850543477724806091311",
    "2904398655777212823206942900445971875646587821051609522215605193631325136260",
    "1"
   ],
   [
    "2630283627321778633778673545048506457244227283376747634599863527799915022425",
    "940513164069974613974034762133339181649727449337769637272141800475219688707",
    "1"
   ],
   [
    "18163405159728587410831224788431815807681099737058317492533922873291080064040",
    "9031215561364014341835611435935312647306825311699176403239210668640380631758",
    "1"
   ],
   [
    "1737647792914177177630507239053383779243024285042606473063461958504284885437",
    "8578877333890370354788498261638161376531459669187328779551688892176587772921",
    "1"
   ],
   [
    "12354002330358693886969982374034740922082764644395640762532220758019139198399",
    "19575157564538221508252074540975523997474775427079762157651825427443387741334",
    "1"
   ],
   [
    "18032566659806712997656749275422220696173189796125826963827833268809202086322",
    "16946040601798695321043126656806137519263966567822907008292341347796760038300",
    "1"
   ],
   [
    "21427739805154086213649306614743044371728499524515544239186979041239237442117",
    "8923985632057383012782601268197398502317544291061078307272989411673950351474",
    "1"
   ],
   [
    "7081324789641261935825569660606618609091430934854329039157458662925381113109",
    "21565060637886002596108024486962876241946928759218624744800835274778545826453",
    "1"
   ],
   [
    "9474074458682123852946953048882083153831215474062305951197139598855191369887",
    "17076595853287865670607387197085589417258936334498202048370392739025133484947",
    "1"
   ],
   [
    "5281705668899767051116648812837006764258612940205033154396824669172615411374",
    "21610060531524118652658079655974381794228027531596545964243289489247182860243",
    "1"
   ],
   [
    "9255585742697107898396540824748617249796675837262778594710205251423547909391",
    "21252311345958756529843826782458097494903171861113489067518450694281638753940",
    "1"
   ],
   [
    "20911406283497461651801969177668183817390375840777623031021125474041622540287",
    "10275270587102628971535520879400364640005486064321744709600223644102588379687",
    "1"
   ]
  ]
 }`)

var vkeyData = vkeyJSON

// groth16PublicInputsTotalBytes matches ProofVerify: sum of UTF-8 lengths of each public input string.
func groth16PublicInputsTotalBytes(inputs []string) uint64 {
	var n uint64
	for _, s := range inputs {
		n += uint64(len(s))
	}
	return n
}

func TestQueryProofVerify(t *testing.T) {
	f := SetupTest(t)

	// Add valid vkey to the keeper for successful tests
	validVKeyID, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth_circuit", vkeyData, "Email authentication circuit", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
	invalidVKeyID, err := f.k.AddVKey(f.ctx, f.govModAddr, "invalid_circuit", invalidVKey, "Invalid circuit for testing", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		proofBz      []byte
		publicInputs []string
		vkeyName     string
		vkeyID       uint64
		shouldError  bool
		errorMsg     string
		// expectedErr, when non-nil, is checked with require.ErrorIs in addition to the
		// errorMsg substring check. Storing this in the struct avoids fragile name-based
		// switches in the test runner.
		expectedErr error
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
			name:         "proof exceeds max size",
			proofBz:      make([]byte, int(types.DefaultMaxGroth16ProofSizeBytes)+1),
			publicInputs: publicInputs,
			vkeyName:     "email_auth_circuit",
			shouldError:  true,
			errorMsg:     "proof size",
		},
		{
			name:         "public inputs exceed max size",
			proofBz:      proofData,
			publicInputs: []string{strings.Repeat("1", int(types.DefaultMaxGroth16PublicInputSizeBytes)+1)},
			vkeyName:     "email_auth_circuit",
			shouldError:  true,
			errorMsg:     "public inputs size",
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
				switch tc.name {
				case "proof exceeds max size":
					require.ErrorIs(t, err, types.ErrProofTooLarge)
				case "public inputs exceed max size":
					require.ErrorIs(t, err, types.ErrPublicInputsTooLarge)
				}
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

// TestQueryProofVerify_ParamMaxSizeEnforced checks that ProofVerify reads limits from module params:
// tightening max below the real payload must fail; restoring limits allows verification to succeed.
func TestQueryProofVerify_ParamMaxSizeEnforced(t *testing.T) {
	f := SetupTest(t)
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "size_limit_circuit", vkeyData, "circuit for size limit tests", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	proofLen := uint64(len(proofData))
	pubTotal := groth16PublicInputsTotalBytes(publicInputs)
	require.Greater(t, proofLen, uint64(1), "proof fixture must be longer than 1 byte")
	require.Greater(t, pubTotal, uint64(1), "public inputs fixture must have positive total size")

	base := types.DefaultParams()

	t.Run("proof rejected when max_groth16_proof_size_bytes below payload", func(t *testing.T) {
		p := base
		p.MaxGroth16ProofSizeBytes = proofLen - 1
		require.NoError(t, f.k.SetParams(f.ctx, p))

		_, err := f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{
			Proof:        proofData,
			PublicInputs: publicInputs,
			VkeyName:     "size_limit_circuit",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrProofTooLarge)
	})

	t.Run("public inputs rejected when max_groth16_public_input_size_bytes below payload", func(t *testing.T) {
		p := base
		p.MaxGroth16ProofSizeBytes = proofLen
		p.MaxGroth16PublicInputSizeBytes = pubTotal - 1
		require.NoError(t, f.k.SetParams(f.ctx, p))

		_, err := f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{
			Proof:        proofData,
			PublicInputs: publicInputs,
			VkeyName:     "size_limit_circuit",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrPublicInputsTooLarge)
	})

	t.Run("verification succeeds when limits cover payload", func(t *testing.T) {
		require.NoError(t, f.k.SetParams(f.ctx, base))

		res, err := f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{
			Proof:        proofData,
			PublicInputs: publicInputs,
			VkeyName:     "size_limit_circuit",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, res.Verified)
	})

	t.Run("proof exactly at max size limit is allowed through size gate", func(t *testing.T) {
		p := base
		p.MaxGroth16ProofSizeBytes = proofLen
		p.MaxGroth16PublicInputSizeBytes = pubTotal
		require.NoError(t, f.k.SetParams(f.ctx, p))

		res, err := f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{
			Proof:        proofData,
			PublicInputs: publicInputs,
			VkeyName:     "size_limit_circuit",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, res.Verified)
	})
}

// TestQueryProofVerifyWithStoredVKey tests the complete flow of storing a vkey and using it for verification
func TestQueryProofVerifyWithStoredVKey(t *testing.T) {
	f := SetupTest(t)

	// 1. Add vkey to keeper
	vkeyID, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyData, "Email authentication circuit", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
	require.Equal(t, 88, circomVKey.NPublic)

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
	vkey1ID, err := f.k.AddVKey(f.ctx, f.govModAddr, "circuit_1", vkeyData, "Circuit 1", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	vkey2ID, err := f.k.AddVKey(f.ctx, f.govModAddr, "circuit_2", vkeyData, "Circuit 2", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	vkey3ID, err := f.k.AddVKey(f.ctx, f.govModAddr, "circuit_3", vkeyData, "Circuit 3", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test verification key", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
	expectedID, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i), types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i), types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
	expectedID, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test key", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
		_, err = f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkeyBytes, "Key 1", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
			_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("multi_key%d", i), vkeyBytes, fmt.Sprintf("Multi Key %d", i), types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, "to_remove", vkeyBytes, "To Remove", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
		actualID, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("predict_key%d", i), vkeyBytes, fmt.Sprintf("Predict Key %d", i), types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
	newID, err := f.k.AddVKey(f.ctx, f.govModAddr, "new_key_after_genesis", vkeyBytes, "New key after genesis", types.ProofSystem_PROOF_SYSTEM_GROTH16)
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
		actualID, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("seq_key%d", i), vkeyBytes, fmt.Sprintf("Seq Key %d", i), types.ProofSystem_PROOF_SYSTEM_GROTH16)
		require.NoError(t, err)
		require.Equal(t, expectedID, actualID)
	}

	// Final next ID should be startID + numVKeys
	finalResp, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
	require.NoError(t, err)
	require.Equal(t, startID+uint64(numVKeys), finalResp.NextId)
}

func TestQueryParams(t *testing.T) {
	f := SetupTest(t)

	customParams := types.Params{
		MaxVkeySizeBytes: 1024,
		UploadChunkSize:  32,
		UploadChunkGas:   500,

		MaxGroth16ProofSizeBytes:         types.DefaultMaxGroth16ProofSizeBytes,
		MaxGroth16PublicInputSizeBytes:   types.DefaultMaxGroth16PublicInputSizeBytes,
		MaxUltraHonkProofSizeBytes:       types.DefaultMaxUltraHonkProofSizeBytes,
		MaxUltraHonkPublicInputSizeBytes: types.DefaultMaxUltraHonkPublicInputSizeBytes,
	}
	err := f.k.Params.Set(f.ctx, customParams)
	require.NoError(t, err)

	resp, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, customParams, resp.Params)
}

// ============================================================================
// UltraHonk (Barretenberg) Query Tests
// ============================================================================

// loadBarretenbergTestdata loads vk, proof, and public_inputs from x/zk/barretenberg/testdata/statics.
// Skips the test if any file is missing (e.g. when testdata is not generated).
func loadBarretenbergTestdata(t *testing.T) (vkBytes, proofBytes, publicInputsBytes []byte) {
	t.Helper()
	// Try paths: from package dir "testdata/barretenberg", or from repo root "x/zk/keeper/testdata/barretenberg"
	candidates := []string{
		filepath.Join("testdata", "barretenberg"),
		filepath.Join("x", "zk", "keeper", "testdata", "barretenberg"),
	}
	var base string
	for _, cand := range candidates {
		path := filepath.Join(cand, "vk")
		if _, err := os.Stat(path); err == nil {
			base = cand
			break
		}
	}
	if base == "" {
		t.Skipf("barretenberg testdata not found (run from repo root or x/zk/keeper, or set up testdata)")
	}
	for _, name := range []string{"vk", "proof", "public_inputs"} {
		path := filepath.Join(base, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("barretenberg testdata %s not found: %v", name, err)
		}
		switch name {
		case "vk":
			vkBytes = data
		case "proof":
			proofBytes = data
		case "public_inputs":
			publicInputsBytes = data
		}
	}
	return vkBytes, proofBytes, publicInputsBytes
}

// TestQueryProofVerifyUltraHonk_Success verifies an UltraHonk proof using barretenberg testdata.
// Requires the real barretenberg library (not stub) and testdata regenerated with bb@4.0.4.
func TestQueryProofVerifyUltraHonk_Success(t *testing.T) {
	if strings.HasPrefix(barretenberg.Version(), "stub") {
		t.Skip("stub library does not perform real verification; build real library and regenerate testdata with bb@4.0.4")
	}
	vkBytes, proofBytes, publicInputsBytes := loadBarretenbergTestdata(t)
	f := SetupTest(t)

	vkeyID, err := f.k.AddVKey(f.ctx, f.govModAddr, "ultrahonk_circuit", vkBytes, "UltraHonk test vkey", types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK)
	require.NoError(t, err)

	reqByName := &types.QueryVerifyUltraHonkRequest{
		Proof:        proofBytes,
		PublicInputs: publicInputsBytes,
		VkeyName:     "ultrahonk_circuit",
	}
	resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, reqByName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Verified, "proof should verify by vkey name")

	reqByID := &types.QueryVerifyUltraHonkRequest{
		Proof:        proofBytes,
		PublicInputs: publicInputsBytes,
		VkeyId:       vkeyID,
	}
	resp2, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, reqByID)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.True(t, resp2.Verified, "proof should verify by vkey ID")
}

// TestQueryProofVerifyUltraHonk_InvalidRequest tests error cases for ProofVerifyUltraHonk.
func TestQueryProofVerifyUltraHonk_InvalidRequest(t *testing.T) {
	vkBytes, proofBytes, publicInputsBytes := loadBarretenbergTestdata(t)
	f := SetupTest(t)

	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "ultrahonk_circuit", vkBytes, "UltraHonk test vkey", types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK)
	require.NoError(t, err)

	t.Run("nil request", func(t *testing.T) {
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty request")
		require.Nil(t, resp)
	})

	t.Run("empty proof", func(t *testing.T) {
		req := &types.QueryVerifyUltraHonkRequest{
			Proof:        nil,
			PublicInputs: publicInputsBytes,
			VkeyName:     "ultrahonk_circuit",
		}
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "proof cannot be empty")
		require.Nil(t, resp)
	})

	t.Run("neither vkey_name nor vkey_id", func(t *testing.T) {
		req := &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
		}
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "either vkey_name or vkey_id must be provided")
		require.Nil(t, resp)
	})

	t.Run("public_inputs not multiple of 32", func(t *testing.T) {
		req := &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: []byte{1, 2, 3}, // not 32-byte aligned
			VkeyName:     "ultrahonk_circuit",
		}
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "multiple of")
		require.Nil(t, resp)
	})

	t.Run("proof exceeds max size", func(t *testing.T) {
		req := &types.QueryVerifyUltraHonkRequest{
			Proof:        make([]byte, int(types.DefaultMaxUltraHonkProofSizeBytes)+1),
			PublicInputs: publicInputsBytes,
			VkeyName:     "ultrahonk_circuit",
		}
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrProofTooLarge)
		require.Nil(t, resp)
	})

	t.Run("public inputs exceed max size", func(t *testing.T) {
		req := &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: make([]byte, int(types.DefaultMaxUltraHonkPublicInputSizeBytes)+1),
			VkeyName:     "ultrahonk_circuit",
		}
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrPublicInputsTooLarge)
		require.Nil(t, resp)
	})

	t.Run("vkey not found by name", func(t *testing.T) {
		req := &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyName:     "nonexistent",
		}
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
		require.Nil(t, resp)
	})

	t.Run("vkey not found by ID", func(t *testing.T) {
		req := &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyId:       9999,
		}
		resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
		require.Nil(t, resp)
	})
}

// TestQueryProofVerifyUltraHonk_ParamMaxSizeEnforced checks that ProofVerifyUltraHonk reads limits from module params.
func TestQueryProofVerifyUltraHonk_ParamMaxSizeEnforced(t *testing.T) {
	vkBytes, proofBytes, publicInputsBytes := loadBarretenbergTestdata(t)
	f := SetupTest(t)

	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "ultrahonk_size_circuit", vkBytes, "UltraHonk size limit tests", types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK)
	require.NoError(t, err)

	proofLen := uint64(len(proofBytes))
	pubLen := uint64(len(publicInputsBytes))
	require.Greater(t, proofLen, uint64(1))
	require.Greater(t, pubLen, uint64(1))

	base := types.DefaultParams()

	t.Run("proof rejected when max_ultrahonk_proof_size_bytes below payload", func(t *testing.T) {
		p := base
		p.MaxUltraHonkProofSizeBytes = proofLen - 1
		require.NoError(t, f.k.SetParams(f.ctx, p))

		_, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyName:     "ultrahonk_size_circuit",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrProofTooLarge)
	})

	t.Run("public inputs rejected when max_ultrahonk_public_input_size_bytes below payload", func(t *testing.T) {
		p := base
		p.MaxUltraHonkProofSizeBytes = proofLen
		p.MaxUltraHonkPublicInputSizeBytes = pubLen - 1
		require.NoError(t, f.k.SetParams(f.ctx, p))

		_, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyName:     "ultrahonk_size_circuit",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrPublicInputsTooLarge)
	})

	t.Run("verification succeeds when limits cover payload", func(t *testing.T) {
		if strings.HasPrefix(barretenberg.Version(), "stub") {
			t.Skip("stub library does not perform real verification")
		}
		require.NoError(t, f.k.SetParams(f.ctx, base))

		res, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyName:     "ultrahonk_size_circuit",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, res.Verified)
	})

	t.Run("payload exactly at max limits passes size gate and verifies", func(t *testing.T) {
		if strings.HasPrefix(barretenberg.Version(), "stub") {
			t.Skip("stub library does not perform real verification")
		}
		p := base
		p.MaxUltraHonkProofSizeBytes = proofLen
		p.MaxUltraHonkPublicInputSizeBytes = pubLen
		require.NoError(t, f.k.SetParams(f.ctx, p))

		res, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyName:     "ultrahonk_size_circuit",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, res.Verified)
	})
}

// TestQueryProofVerifyUltraHonk_Groth16VKeyRejected ensures a Groth16 vkey cannot be used for UltraHonk verification.
func TestQueryProofVerifyUltraHonk_Groth16VKeyRejected(t *testing.T) {
	_, proofBytes, publicInputsBytes := loadBarretenbergTestdata(t)
	f := SetupTest(t)

	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "groth16_circuit", vkeyData, "Groth16 vkey", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	req := &types.QueryVerifyUltraHonkRequest{
		Proof:        proofBytes,
		PublicInputs: publicInputsBytes,
		VkeyName:     "groth16_circuit",
	}
	resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not an UltraHonk key")
	require.Nil(t, resp)
}

// TestQueryProofVerifyUltraHonk_WrongInputsReturnsFalse verifies that wrong public inputs yield Verified=false (not an error).
// Requires the real barretenberg library (stub always returns success without cryptographic verification).
func TestQueryProofVerifyUltraHonk_WrongInputsReturnsFalse(t *testing.T) {
	if strings.HasPrefix(barretenberg.Version(), "stub") {
		t.Skip("stub library always returns success; build real library to test wrong-inputs rejection")
	}
	vkBytes, proofBytes, _ := loadBarretenbergTestdata(t)
	f := SetupTest(t)

	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "ultrahonk_circuit", vkBytes, "UltraHonk test vkey", types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK)
	require.NoError(t, err)

	// Wrong public inputs: correct count (must match vkey's expected public input count)
	// but with incorrect values. The testdata vkey expects 2 public inputs (64 bytes total).
	// We provide 2 field elements all set to 0xff so the count is correct but values are wrong.
	wrongInputs := make([]byte, 2*barretenberg.FieldElementSize)
	for i := range wrongInputs {
		wrongInputs[i] = 0xff
	}

	req := &types.QueryVerifyUltraHonkRequest{
		Proof:        proofBytes,
		PublicInputs: wrongInputs,
		VkeyName:     "ultrahonk_circuit",
	}
	resp, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.Verified, "proof with wrong inputs should not verify")
}
