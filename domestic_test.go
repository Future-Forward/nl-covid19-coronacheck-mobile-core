package mobilecore

import (
	"encoding/json"
	"github.com/go-errors/errors"
	"github.com/minvws/nl-covid19-coronacheck-idemix/issuer"
	"github.com/minvws/nl-covid19-coronacheck-idemix/issuer/localsigner"
	"github.com/privacybydesign/gabi"
	gabipool "github.com/privacybydesign/gabi/pool"
	"strconv"
	"testing"
	"time"
)

func TestInitialization(t *testing.T) {
	r1 := InitializeHolder("./testdata")
	if r1.Error != "" {
		t.Fatal("Could not initialize holder:", r1.Error)
	}

	// Initialize verifier with testdata
	r2 := InitializeVerifier("./testdata")
	if r2.Error != "" {
		t.Fatal("Could not initialize verifier:", r2.Error)
	}
}

func TestFlow(t *testing.T) {
	credentialAmount := 3
	credentialAttributes := buildCredentialsAttributes(credentialAmount)

	// Generate holdercore secret key
	r3 := GenerateHolderSk()
	if r3.Error != "" {
		t.Fatal("Could not generate holdercore secret key:", r3.Error)
	}

	// Create a signer and issuer for the tests
	ls, err := localsigner.NewFromString(testIssuerPkId, testIssuerPkXml, testIssuerSkXml, gabipool.NewRandomPool())
	if err != nil {
		t.Fatal("Could not create local signer:", err)
	}

	iss := issuer.New(ls)
	pim, err := iss.PrepareIssue(credentialAmount)
	if err != nil {
		t.Fatal("Could not prepare issue:", err)
	}

	// Issuance dance
	pimJson, err := json.Marshal(pim)
	if err != nil {
		t.Fatal("Could not JSON marshal prepare issue message:", err)
	}

	r4 := CreateCommitmentMessage(r3.Value, pimJson)
	if r4.Error != "" {
		t.Fatal("Could not create commitment message:", r4.Error)
	}

	icm := new(gabi.IssueCommitmentMessage)
	err = json.Unmarshal(r4.Value, icm)
	if err != nil {
		t.Fatal("Could not unmarshal issue commitment message:", err)
	}

	im := &issuer.IssueMessage{
		PrepareIssueMessage:    pim,
		IssueCommitmentMessage: icm,
		CredentialsAttributes:  credentialAttributes,
	}

	ccms, err := iss.Issue(im)
	if err != nil {
		t.Fatal("Could not issue create credential messages:", err)
	}

	ccmsJson, err := json.Marshal(ccms)
	if err != nil {
		t.Fatal("Could not marshal create credential messages:", err)
	}

	r5 := CreateCredentials(ccmsJson)
	if r5.Error != "" {
		t.Fatal("Could not create credential:", r5.Error)
	}

	// Check back attributes returns on creation
	var r5Values []*CreateCredentialResultValue
	err = json.Unmarshal(r5.Value, &r5Values)
	if err != nil {
		t.Fatal("Could not unmarshal create credential result values:", err)
	}

	if credentialAmount != len(r5Values) {
		t.Fatal("Invalid amount of create credential result values")
	}

	for i := 0; i < credentialAmount; i++ {
		val := r5Values[i]

		err := areAttributesEqualWithCredentialVersion(credentialAttributes[i], val.Attributes)
		if err != nil {
			t.Fatal("Attributes do not match attributes from create credentials")
		}

		credJson, err := json.Marshal(val.Credential)
		if err != nil {
			t.Fatal("Could not marshal credential:", err)
		}

		// Read
		r6 := ReadDomesticCredential(credJson)
		if r6.Error != "" {
			t.Fatal("Could not read credential:", r6.Error)
		}

		err = checkAttributesJson(credentialAttributes[i], r6.Value)
		if err != nil {
			t.Fatal(err)
		}

		// Disclose
		r7 := Disclose(r3.Value, credJson)
		if r7.Error != "" {
			t.Fatal("Could not disclose credential:", r6.Error)
		}

		// Verify. Only the first two credentials should validate due to validFrom in future
		r8 := Verify(r7.Value)
		if i < 2 {
			if r8.Status != VERIFICATION_SUCCESS || r8.Error != "" {
				t.Fatal("Could not verify credential:", r8.Error)
			}

			if *r8.Details != attributesToVerificationDetails(credentialAttributes[i]) {
				t.Fatal("Credential attributes do not correspond with verification details")
			}
		}

		if i > 2 && (r8.Status != VERIFICATION_FAILED_ERROR || r8.Error == "") {
			t.Fatal("Credential should not validate due to validFrom in future")
		}
	}
}

func TestUnrecognizedCred(t *testing.T) {
	someQR := []byte(`1K9P/3FD!C.%2H5N4$**$IVY+3$`)

	r1 := Verify(someQR)
	if r1.Status != VERIFICATION_FAILED_UNRECOGNIZED_PREFIX {
		t.Fatal("QR could should have status unrecognized")
	}
}

func TestDeniedProof(t *testing.T) {
	deniedQr := []byte(`NL2:3QYLJN7UNC EJZ2I/1AJ/NLOSBX8O/N7*SQ376YP86E:U6ZO:5K UXRBMCARHV4ETZT1 -3-%CKD6T4MLMGS%:-S+U8EMJV0+3JINFCK4RYSQR8G/-JC M-L-VXS14XXNF-.-E 1:7H1X2GINPH0%YRP+/B.GSP4RHEDXRYTKO/VL1BC39X6MDM6ES+HPKH9VUS$XMYKU.%PJCCQT74HRY8$Q73I2P-77D$8NN.TK9PX4+/:8KR39HC*ZG86QJ9QKKJ.MAFYCPPV3BI*IYARY**J%WQAXX/-5KARDLZ9LXXIL%KKYF.X.JVM0$W0P4ATK66EQUI/R/7Z2TOHE4H:163J:7D5S X*ULK1VVYRC-4.PTE$ZJOWCGJR$UQAACJ1QAIXJ/SA1M1W NJBQQP56YY2V7YEP8E6:UD5AXMAXV*.DG.MW QQ/3QPEHR-YQIA0:85T+W4YC087A$MQ825173D/LBA0GL:88/S4Q8GNP2KCAD0LW$R1UXUH0PVCO4--1:+JH $PG6ZPORBHG7O4HU63-.1JZ9DUDACOHYZ869Y*X7TG+PYRRCY08*Q9DCAT0T:+HJ9B0ZS.NJIEHFTVEO8Y+L6$*H :TTSGRAS M9U8SLC+33/C3*86HAOW6PIPP13 HF+38%8EA$+7O+6+URI09%LDJX872V8 VM8O:Z1H0FIUVKGBBP4OL$Z0E8$KJAIUC6%8Y2BHX+95P.JP8CM1SUYF *J3UB3.KW81PJ$-8288C17NTCYFMI0%KB1GO07ZKB$R:DYFO+Q9:GA+9EE40Q3GQ53I-*:+*2U  LS 9O9Y2V7TZTO.FB5NSNMCOZX65WU%CYEJH$%9585A126-T5XB3%UVX3ULIQ5XJ7H.C%QG4RG$P L:C6WC*6N5X646 SJATTH85P5QJE/K3PU80CVYN+VZ9+8C8IPT+%T1YF50+4.NJPI12KM0..1ZDK/$NEZBVW-2TM+%V9$N00YUQSO 57M1JE-M2TT%%9RXHDWH/BFX3ZV/V6.S5F6BPJGZPJ4V$*K6ACOS6P:XFS-T-RW21H52 65VUSS*P505ILMZN%9DNRO42NKQKACHG/1GNLS V0OUUJL*9PZ/06SS9/GCFKIFW39KJ/ IVB10WE%T1C9MS2J4BR/9E$F5O5 DYPNVV9Y I0H4MRC5AO36UNWBRMCUPGCYZH+FIELM-R4:7L3L/L3F5WL4.-73494V.X67$*H$S9X51EX:6VRUMEBT224 +FW/0*4.V75:W:HXLB*Z6DGK27L.ZY-S4LZOP.HIWG:/PM9L6PVN.$PZDS$6$0KAL8OT/TM`)
	r1 := Verify(deniedQr)
	if r1.Status != VERIFICATION_FAILED_ERROR {
		t.Fatal("QR could should have status error")
	}
}

func buildCredentialsAttributes(credentialAmount int) []map[string]string {
	cas := make([]map[string]string, 0, credentialAmount)

	for i := 0; i < credentialAmount; i++ {
		validFrom := time.Now().Truncate(time.Hour).AddDate(0, 0, i-1).UTC().Unix()

		ca := map[string]string{
			"isSpecimen":       "0",
			"isPaperProof":     "0",
			"validFrom":        strconv.FormatInt(validFrom, 10),
			"validForHours":    "40",
			"firstNameInitial": "A",
			"lastNameInitial":  "R",
			"birthDay":         "20",
			"birthMonth":       "10",
		}

		cas = append(cas, ca)
	}

	return cas
}

func attributesToVerificationDetails(attributes map[string]string) VerificationDetails {
	return VerificationDetails{
		CredentialVersion: "2",
		IsSpecimen:        attributes["isSpecimen"],
		IssuerCountryCode: "NL",

		FirstNameInitial: attributes["firstNameInitial"],
		LastNameInitial:  attributes["lastNameInitial"],
		BirthDay:         attributes["birthDay"],
		BirthMonth:       attributes["birthMonth"],
	}
}

func checkAttributesJson(attributes map[string]string, attributesJson []byte) error {
	var decodedAttributes map[string]string
	err := json.Unmarshal(attributesJson, &decodedAttributes)
	if err != nil {
		return errors.New("Error unmarshalling attributes json")
	}

	return areAttributesEqualWithCredentialVersion(attributes, decodedAttributes)
}

func areAttributesEqualWithCredentialVersion(attributes map[string]string, decodedAttributes map[string]string) error {
	if len(attributes)+1 != len(decodedAttributes) {
		return errors.New("Decoded attributes amount (with credential version) mismatch")
	}

	versionAttr, ok := decodedAttributes["credentialVersion"]
	if !ok || versionAttr != "2" {
		return errors.Errorf("Decoded credential version attribute isn't as expected")
	}

	for k, v := range attributes {
		vv, ok := decodedAttributes[k]
		if !ok || v != vv {
			return errors.New("Decoded attributes mismatch")
		}
	}

	return nil
}

var testIssuerPkId = "testPk"
var testIssuerPkXml = `
<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<IssuerPublicKey xmlns="http://www.zurich.ibm.com/security/idemix">
   <Counter>0</Counter>
   <ExpiryDate>1643236269</ExpiryDate>
   <Elements>
      <n>20802935239735680649300836501595337523084875943858453072086532937087727484840258830186357666123341014323452958939683048679012474106796902098040267362870336731445929141036769105771621117492171859018462481404564743331200385781418271855967978254854778444385976985738694156316217095962588899765723273806770494909607541699952017849399309408007417979307186976452574612563487533124622189564440377177716288820962267275834180700278177843625771149134638952929043702960735256683313111391020400787961832214020770797739480756955988872633915049934259399056153089221930482156932799568214338448280950389504049086156004387545860041317</n>
      <Z>2065986586410238833762790310229414413235521258173254608333023393780978724887408481475325833146398661024881222042644441268838642333288989158779288953444177253298144420011805254841343186740255878481477733222511792887167409578287344530955261855198293852960984807526538096936313587714953339800852027633591488235985546469908758364845655506616048420506163727403292195730917677299882665732109505958996763689058364847379785727353272356651859529627360372630402337299189664105963260174827642174416165745480011624486855140065769813554873319959454585264015750525953122800006881123418514606074889786620496213429682546226715415850</Z>
      <S>9747020032497748531479972005651016172463609770082275462639756960862111120525276794721713428051680043624048901069084701927087112553945623369830101017052212366223341706894004442101497313134579073191353775431025513711767334295781049848177143816836351135693918501641428088656760979443363699181434056174440144303468940886784070846733039167978510816289033605907491173130943969510875169553737212231137139551366358436034142595185159440155427490655734188739921425475230307706873244642564186711876360549429477144304401313212629175634297400078412730856807526937872052054517906700089421208808198580942298613175274161167641810784</S>
      <Bases num="12">
         <Base_0>13780374641350384168334666952318376830294415916461973659542611943557311255447084810738946281845873551691606013805216940709302694843306661938242444632119716901182259750527264304726741939163123381612754627062832214496907517546686799808347715040087670931695354988786500014933654952358444081787929493498190784566045013732753097245198167100883780628038145935944724911328019125782500000961380482070211489587117273909317122344541355345010747199710975648887159650770245518663760990732802684172012102473170131306051095286846412272851190224644617778788234319511849492408307984594217301320149494120461712807203267779856107813185</Base_0>
         <Base_1>12252493278683441465069675194610188860110186593467631081745672282179552919130035203799576271368287924024645684199636872010117236035117790836473907494509943235807346757734052467539726777120137951737852196409822676016956191351623401943869047792240406553324700231918521455462508601150206973634152421214435926852220596369587191807896569838175466792783141601666347308637964966016747713880504107838233059881464038488451213216193931092164632235641540648936792862109078421705746277233952748787049551047624875632864544224780788852960793160923079579280345905059680538466183099054546961295553520170546903119275861213169961731000</Base_1>
         <Base_2>6898513089038472019379564706937124184985664328743216257752468561354943271589034780255774214080319730396899067005744693309732895004234294122841887908417716995550296502872454291354023618474215406690386543108957524328020530699527683722251665333248161715927154964536775971186241679176392501458783118472919898979134844504779660844398415201985381180246399633824088105536824121878578196778503388884388039443670143926346479165137410171227230489535470054111306655907529435558600326254324814473816548260335466941181714395910738224863839698646060251451125558781892087773201851691822750004391255799049904837017805509353151417911</Base_2>
         <Base_3>10599241348748790652376003363993720813333807569709341850842399017219390669322123873999183273339723812125805650441491167350690646352472573290300111569416494459294280212120435554739385529723891008206418450734704772530979859558824509376351848866800707220718379786620864941144057417892006000871416470426267323522234692793300018210732956516181750593839599747986884370387843539357467779505599914678797126135282466097357169811241699558256219613825646612439193765110027178193869012933690295346298286655446997362376346575034050723117451757938486586946810250569498331487209897974388808781969032787881115962484807219346458541597</Base_3>
         <Base_4>16708659761794886917035753107680077042363522744725118370303280454928971673269127851087140018763055353306690371167746495591575740615270475714888375623075605782815373029359238661138484811550543606374384501717410042952231593042111667840589631729072384566470839872142053268760324407158503547957245542929597506236386839203013081164210987281479679444124604443087297322638545412655014435616321631589728757062886598782302835081678245414866452684940156313993187653922604275686416824329173629442685477786196413536054700488813237385823243414230524652978047173590777624537811915694973329342292188365513929905442831614558957528652</Base_4>
         <Base_5>4823463391342164334356136847095564474781843505620078484195205829164331093982191040489179715043099343785312403311240978350707591530700276872429843497470936797252937980798479267997879872026356595342837932485338729866357505338625759004590174827111789563476582510216117197410169608756356003257688930256479578608396170815533128191276159750918220770826989168144659930010442332527942511608982468195450489744709608696745606068095013398872813428634708711973483701666498399407516853290024162966452708990009199444714225136055132088990831665405904387814207729372832018936876424787797158692517549744353828213781431080165215911241</Base_5>
         <Base_6>6328150748868557732613790811282282094861034367786734439816929600659279159141810462215991942097302211340863080595384973639638668398134073764910360211194418034049963376415581290580889893176717938191994372193004205747129751481251267336222298609044347743897155335153003175266000848157601164201521698872905138409865519092146000103068241706932794576426354878824902028481519111559459199803667524071914511600647319807312907236078312636286379345032691257973340019471369661087652997617620495608804951370501160871382353241470534171016057968785457572217302771568508208049949259766086492412661193908711058688576690727458732066908</Base_6>
         <Base_7>4222085167942431842483536134220085914742294020096854942083757134527853241023925126622859695275311616755494625544968328434639808579108862090958958414586028103057483925639413126190076817639720262743011639046891613909214582383028776898612854090560839634687458275509818266876738088721177037935901803750037492230710966899304006896493018566177918659419405589119274997889075588507689381555827664200321429515833734834751370514816731830207369085417676370160893610662197600061917925267737638936847060689649617827551760094868072661150336598034871427729524850885176125183718532605751304691771626550854629612199143713417732336973</Base_7>
         <Base_8>6689205093753950025745241772455508208127760053680763053116833365018109217555130810117149179849347865564586144693550396079034160996586946238453082083734990374020267312831317981744910274709418105655293641234616423754205583495595209237374018279776742327581348148066345422721004138045091996581320658321362460089081461999503853922618430086787334653775150434282710930060637238621855056544525430985011195749758180120954141477340541457197994631545897156675549419837945939768911551165580835720411903603842165891335291133053718609514656729043351508889388517529214375317619382630571241448637915053379796594659728997946506428949</Base_8>
         <Base_9>10039599304555800211371398257090732908069570902344825856514802908498803295486206571769032313221635904962077573540416347349732220565407334168522535053600094714425150661930456704962412130163660498156384542141067585449338454746252336121183452375145229533015915265557400221545592108550191648761211591080553131968285259689937497856479858333796897653479509560300770287305565789211002529887200369274929662787508250311723811213514630210624617469200542038106250053152174476331196573091981583158426231337854831514599338227597237227889208057851072852558075517106519037828078985281345432625359701706362165411581545014803241802004</Base_9>
         <Base_10>1860989457236634857271280809951672240532367839516129241887157150714378222328306288973899456833745186064003217053753260885418935952587676492960681170103177437077220236514278382690356688219315016292140740032816909644791085050588806744326542254528463632036454391128085224830921696826154807929395713221779059311085981047486438594986429502001797081897536876882869510625514334424093373778589264021445233747056806107145747919439801542632738732898748316921450986492398794804584603385294201737581161650542403927819046672477962043670610253792590410755504945572366957627358135330489717185895305858061137704297932071388332781561</Base_10>
         <Base_11>9654515452119593342734339508446651081113630116131703232832954756090761577371088219265241882519716018427049126506674237193979219301150250721179514616792153688494950057027857657512812044457900727794787377990662939160420293278542875377863698346489182310203482230871013672923790718143178714656709964092128580906438934580895628049252032854588710380097050945997531060755277738153588489085816804404356281136854398669030490126899297685723245896085640066130209086355127376700916984993742891903014916196158992185706607765472153699807068906281737283678482161763141621411183483604860621537927307723261746603486643745312563173680</Base_11>
      </Bases>
   </Elements>
   <Features>
      <Epoch length="432000"></Epoch>
   </Features>
</IssuerPublicKey>
`

// This private key is only included for testing purposes
var testIssuerSkXml = `
<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<IssuerPrivateKey xmlns="http://www.zurich.ibm.com/security/idemix">
   <Counter>0</Counter>
   <ExpiryDate>1643236269</ExpiryDate>
   <Elements>
      <p>177012637043708070369346595490119089414679389021408180013464395209951360192645020817015695649151215288649409216960126694483488017472042403440397644004880041970079733165623755694692485795555807794960762001732329088330241901428036942022822173368524665368789552123938056470852714410318661337215585482555378108319</p>
      <q>117522316977849479463233663559700720601695480584241478428956832421994025621178370784123625940483784737436163627460510347400695272671758678973491443111298748173727127464926161787335387475004832338466293684779246497629948437027518725079655044423378513208787268087929801631212412349398775870123476278627993624443</q>
      <pPrime>88506318521854035184673297745059544707339694510704090006732197604975680096322510408507847824575607644324704608480063347241744008736021201720198822002440020985039866582811877847346242897777903897480381000866164544165120950714018471011411086684262332684394776061969028235426357205159330668607792741277689054159</pPrime>
      <qPrime>58761158488924739731616831779850360300847740292120739214478416210997012810589185392061812970241892368718081813730255173700347636335879339486745721555649374086863563732463080893667693737502416169233146842389623248814974218513759362539827522211689256604393634043964900815606206174699387935061738139313996812221</qPrime>
   </Elements>
</IssuerPrivateKey>
`
