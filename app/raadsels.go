package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Raadsels: een eigen verzameling kinderraadsels (Nederlands en Engels) en rij-puzzels die de app
// zelf maakt ("welk getal komt erna?"), zodat ze nooit op raken. Eerst zelf raden, dan een hint,
// dan het antwoord. Geen punten.

type raadsel struct {
	Vraag    string
	Antwoord []string // het eerste is het antwoord dat we laten zien; de rest telt ook als goed
	Hint     string
	getallen bool
}

var raadselsNL = []raadsel{
	{"Wat heeft tanden maar kan niet bijten?", []string{"een kam", "kam", "rits", "zaag"}, "Je gebruikt het voor je haar.", false},
	{"Wat wordt natter naarmate het meer droogt?", []string{"een handdoek", "handdoek"}, "Je pakt het na het douchen.", false},
	{"Wat heeft een hals maar geen hoofd?", []string{"een fles", "fles"}, "Er zit vaak drinken in.", false},
	{"Wat gaat omhoog maar komt nooit naar beneden?", []string{"je leeftijd", "leeftijd"}, "Op je verjaardag wordt het meer.", false},
	{"Hoe meer je ervan weghaalt, hoe groter het wordt. Wat is het?", []string{"een gat", "gat", "kuil"}, "Denk aan graven in het zand.", false},
	{"Wat heeft een oog maar kan niet zien?", []string{"een naald", "naald"}, "Er gaat een draad doorheen.", false},
	{"Wat kun je vangen maar niet gooien?", []string{"een verkoudheid", "verkoudheid", "griep"}, "Je wordt er snotterig van.", false},
	{"Wat heeft vier poten maar kan niet lopen?", []string{"een tafel", "tafel", "stoel", "bed"}, "Je eet er aan.", false},
	{"Wat heeft wijzers maar geen handen om te zwaaien?", []string{"een klok", "klok", "horloge"}, "Het tikt.", false},
	{"Ik heb sleutels maar open geen deuren. Wat ben ik?", []string{"een piano", "piano", "toetsenbord"}, "Je kunt er muziek mee maken.", false},
	{"Wat breekt als je de naam zegt?", []string{"stilte", "de stilte"}, "Ssst!", false},
	{"Wat is van jou, maar gebruiken anderen veel vaker dan jij?", []string{"je naam", "naam"}, "Anderen roepen het als ze je zoeken.", false},
	{"Wat loopt maar heeft geen benen?", []string{"een rivier", "rivier", "water", "neus", "klok", "kraan"}, "Er zwemmen vissen in.", false},
	{"Wat staat in het midden van Parijs?", []string{"de r", "r"}, "Kijk naar de letters.", false},
	{"Welke maand heeft 28 dagen?", []string{"alle maanden", "alle", "allemaal", "elke maand"}, "Lees de vraag heel precies.", false},
	{"Wat gaat de hele wereld rond maar blijft in een hoekje?", []string{"een postzegel", "postzegel"}, "Het zit op een envelop.", false},
	{"Hoe meer er zijn, hoe minder je ziet. Wat is het?", []string{"het donker", "donker", "duisternis", "mist"}, "'s Nachts is er veel van.", false},
	{"Wat heeft een bed maar slaapt nooit?", []string{"een rivier", "rivier"}, "Het stroomt.", false},
	{"Wat heeft een duim en vier vingers maar leeft niet?", []string{"een handschoen", "handschoen", "want"}, "In de winter draag je het.", false},
	{"Wat kun je houden zonder het aan te raken?", []string{"een belofte", "belofte", "je adem", "adem", "geheim"}, "Je geeft je woord.", false},
	{"Wat valt maar doet nooit pijn?", []string{"de nacht", "nacht", "sneeuw", "regen", "blad", "bladeren"}, "Het wordt dan donker.", false},
	{"Wat gaat op en neer maar beweegt niet?", []string{"een trap", "trap"}, "Je loopt er naar boven mee.", false},
	{"Wat heeft een staart en een kop maar geen lijf?", []string{"een munt", "munt", "geldstuk"}, "Je kunt ermee tossen.", false},
	{"Wat moet je breken voordat je het kunt gebruiken?", []string{"een ei", "ei"}, "Voor een omelet.", false},
	{"Ik ben altijd voor je, maar je kunt me nooit zien. Wat ben ik?", []string{"de toekomst", "toekomst", "morgen"}, "Het komt nog.", false},
	{"Welk woord wordt altijd fout gespeld?", []string{"fout"}, "Lees de vraag nog eens.", false},
	{"Wat heeft veel ringen maar geen vingers?", []string{"een boom", "boom", "telefoon"}, "Tel ze en je weet hoe oud hij is.", false},
	{"Wat heeft één hoorn en geeft melk?", []string{"een melkwagen", "melkwagen", "melkboer"}, "Het is geen dier.", false},
	{"Wat heeft vleugels maar kan niet vliegen?", []string{"een windmolen", "windmolen", "molen", "gebouw", "ziekenhuis"}, "Typisch Nederlands.", false},
	{"Wat wordt kleiner als je het omdraait?", []string{"9", "negen", "het getal 9"}, "Het is een cijfer.", false},
	{"Wat zit er in een ei maar je ziet het niet?", []string{"de dooier", "dooier", "kuiken"}, "Het is geel.", false},
	{"Wat heeft een mond maar eet nooit?", []string{"een rivier", "rivier", "fles"}, "Het stroomt naar de zee.", false},
	{"Wat is licht als een veer, maar zelfs de sterkste persoon kan het niet lang vasthouden?", []string{"je adem", "adem"}, "Hou het maar eens in.", false},
	{"Wat heeft steden maar geen huizen, bossen maar geen bomen, en water maar geen vissen?", []string{"een landkaart", "kaart", "landkaart"}, "Je kijkt erop als je de weg zoekt.", false},
	{"Wat hoort bij jou, maar je ziet het alleen in een spiegel?", []string{"je gezicht", "gezicht", "je ogen"}, "Je wast het 's ochtends.", false},
	{"Wat kun je niet zien maar wel horen, en het antwoordt als je roept?", []string{"een echo", "echo"}, "In de bergen of een tunnel.", false},
	{"Wat staat stil maar gaat toch van Amsterdam naar Utrecht?", []string{"de weg", "weg", "snelweg", "spoor"}, "Auto's rijden eroverheen.", false},
	{"Ik heb een hoofd en een voet maar geen lijf. Wat ben ik?", []string{"een bed", "bed"}, "Je slaapt erin.", false},
	{"Wat heeft een rug maar geen buik?", []string{"een stoel", "stoel", "boek"}, "Je zit erop.", false},
	{"Wat kun je maken, maar niet zien?", []string{"geluid", "lawaai", "een geluid"}, "Je hoort het wel.", false},
}

var raadselsEN = []raadsel{
	{"What has teeth but can't bite?", []string{"a comb", "comb", "zip", "saw"}, "You use it on your hair.", false},
	{"What gets wetter the more it dries?", []string{"a towel", "towel"}, "You use it after a shower.", false},
	{"What has a neck but no head?", []string{"a bottle", "bottle"}, "There's often a drink in it.", false},
	{"What goes up but never comes down?", []string{"your age", "age"}, "It grows on your birthday.", false},
	{"The more you take away, the bigger it gets. What is it?", []string{"a hole", "hole"}, "Think of digging in the sand.", false},
	{"What has an eye but cannot see?", []string{"a needle", "needle", "storm"}, "A thread goes through it.", false},
	{"What can you catch but not throw?", []string{"a cold", "cold"}, "It makes your nose run.", false},
	{"What has four legs but can't walk?", []string{"a table", "table", "chair", "bed"}, "You eat at it.", false},
	{"What has hands but can't clap?", []string{"a clock", "clock", "watch"}, "It ticks.", false},
	{"I have keys but open no locks. What am I?", []string{"a piano", "piano", "keyboard"}, "You can make music with it.", false},
	{"What breaks when you say its name?", []string{"silence"}, "Shhh!", false},
	{"What belongs to you, but others use it more than you do?", []string{"your name", "name"}, "People call it when they look for you.", false},
	{"What runs but has no legs?", []string{"a river", "river", "water", "nose", "tap"}, "Fish swim in it.", false},
	{"What travels around the world but stays in a corner?", []string{"a stamp", "stamp"}, "It goes on an envelope.", false},
	{"The more of them there are, the less you see. What are they?", []string{"darkness", "the dark", "dark", "fog"}, "There's a lot of it at night.", false},
	{"What has a thumb and four fingers but isn't alive?", []string{"a glove", "glove", "mitten"}, "You wear it in winter.", false},
	{"What can you keep after giving it to someone?", []string{"your word", "a promise", "promise", "word"}, "You make one and must not break it.", false},
	{"What goes up and down but doesn't move?", []string{"stairs", "a staircase", "staircase"}, "You climb them.", false},
	{"What has a head and a tail but no body?", []string{"a coin", "coin"}, "You can flip it.", false},
	{"What has to be broken before you can use it?", []string{"an egg", "egg"}, "For an omelette.", false},
	{"What word is always spelled wrong?", []string{"wrong"}, "Read the question again.", false},
	{"What has many rings but no fingers?", []string{"a tree", "tree", "phone", "telephone"}, "Count them to know how old it is.", false},
	{"What has cities but no houses, forests but no trees, and water but no fish?", []string{"a map", "map"}, "You look at it to find the way.", false},
	{"What can you hear but not see, and it answers when you call?", []string{"an echo", "echo"}, "In the mountains or a tunnel.", false},
	{"I have a head and a foot but no body. What am I?", []string{"a bed", "bed"}, "You sleep in it.", false},
	{"What is light as a feather, but even the strongest person can't hold it for long?", []string{"your breath", "breath"}, "Try holding it.", false},
	{"Which month has 28 days?", []string{"all of them", "all", "every month", "all months"}, "Read the question carefully.", false},
	{"What has one horn and gives milk?", []string{"a milk truck", "milk truck", "milk lorry"}, "It's not an animal.", false},
	{"What is always in front of you but can't be seen?", []string{"the future", "future"}, "It hasn't happened yet.", false},
	{"What can fill a room but takes up no space?", []string{"light", "air", "sound", "laughter"}, "Switch it on.", false},
	{"What has a bottom at the top?", []string{"your legs", "legs", "a leg"}, "You stand on them.", false},
	{"What kind of band never plays music?", []string{"a rubber band", "rubber band", "elastic band"}, "It's stretchy.", false},
	{"What has a mouth but never eats?", []string{"a river", "river", "bottle"}, "It flows into the sea.", false},
	{"What do you throw out when you want to use it, and take in when you're done?", []string{"an anchor", "anchor"}, "Boats use it.", false},
	{"What goes through towns and over hills but never moves?", []string{"a road", "road", "path"}, "Cars drive on it.", false},
	{"What has a back but no stomach?", []string{"a chair", "chair", "book"}, "You sit on it.", false},
	{"Where does today come before yesterday?", []string{"in the dictionary", "dictionary"}, "Think of alphabetical order.", false},
	{"What gets smaller every time it takes a bath?", []string{"soap", "a bar of soap"}, "It makes bubbles.", false},
	{"What has words but never speaks?", []string{"a book", "book"}, "You read it.", false},
	{"What can you make but never see?", []string{"a sound", "sound", "noise"}, "You can hear it.", false},
}

// maakRij maakt een getallenrij met een regel; het kind raadt het volgende getal.
func maakRij(rnd *rand.Rand, taal string) raadsel {
	var rij []int
	var hint string
	nl := taal == "nl"
	t := func(a, b string) string {
		if nl {
			return a
		}
		return b
	}
	switch rnd.Intn(7) {
	case 0: // optellen
		start, stap := rnd.Intn(20)+1, rnd.Intn(9)+2
		for i := 0; i < 6; i++ {
			rij = append(rij, start+i*stap)
		}
		hint = t("Kijk hoeveel er steeds bij komt.", "Look at how much is added each time.")
	case 1: // aftrekken
		stap := rnd.Intn(7) + 2
		start := stap*6 + rnd.Intn(30)
		for i := 0; i < 6; i++ {
			rij = append(rij, start-i*stap)
		}
		hint = t("Kijk hoeveel er steeds af gaat.", "Look at how much is taken away each time.")
	case 2: // verdubbelen of keer drie
		f := rnd.Intn(2) + 2
		x := rnd.Intn(4) + 1
		for i := 0; i < 6; i++ {
			rij = append(rij, x)
			x *= f
		}
		hint = t("Het gaat steeds keer hetzelfde getal.", "It's multiplied by the same number each time.")
	case 3: // kwadraten
		b := rnd.Intn(3) + 1
		for i := b; i < b+6; i++ {
			rij = append(rij, i*i)
		}
		hint = t("1×1, 2×2, 3×3 …", "1×1, 2×2, 3×3 …")
	case 4: // groeiende stappen
		x := rnd.Intn(10) + 1
		for i := 1; i <= 6; i++ {
			rij = append(rij, x)
			x += i
		}
		hint = t("De stap wordt steeds één groter.", "The step gets one bigger each time.")
	case 5: // Fibonacci-achtig
		a, b := rnd.Intn(3)+1, rnd.Intn(3)+1
		for i := 0; i < 6; i++ {
			rij = append(rij, a)
			a, b = b, a+b
		}
		hint = t("Tel de twee getallen ervoor bij elkaar op.", "Add the two numbers before it.")
	default: // om en om
		x, p, q := rnd.Intn(10)+1, rnd.Intn(5)+2, rnd.Intn(4)+1
		for i := 0; i < 6; i++ {
			rij = append(rij, x)
			if i%2 == 0 {
				x += p
			} else {
				x -= q
			}
		}
		hint = t("Om de beurt gebeurt er iets anders.", "Something different happens every other step.")
	}
	var delen []string
	for _, n := range rij[:5] {
		delen = append(delen, strconv.Itoa(n))
	}
	vraag := t("Welk getal komt erna?  ", "Which number comes next?  ") + strings.Join(delen, ", ") + ", …"
	return raadsel{Vraag: vraag, Antwoord: []string{strconv.Itoa(rij[5])}, Hint: hint, getallen: true}
}

// schoon: kleine letters, zonder accenten, lidwoorden en leestekens, voor het vergelijken.
func schoon(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, strings.ToLower(s))
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			return r
		}
		return -1
	}, s)
	var woorden []string
	for _, w := range strings.Fields(s) {
		switch w {
		case "een", "de", "het", "a", "an", "the", "je", "your":
			continue
		}
		woorden = append(woorden, w)
	}
	return strings.Join(woorden, " ")
}

func (r raadsel) klopt(antwoord string) bool {
	a := schoon(antwoord)
	if a == "" {
		return false
	}
	for _, goed := range r.Antwoord {
		g := schoon(goed)
		if a == g || (len(g) >= 3 && strings.Contains(" "+a+" ", " "+g+" ")) {
			return true
		}
	}
	return false
}

/* ---------------- scherm ---------------- */

type raadselStaat struct {
	huidig raadsel
	invoer textinput.Model
	stand  int // 0 raden, 1 hint getoond, 2 antwoord getoond, 3 goed
	gezien []int
	poging string
}

func raadselPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "raadsels.json") }

func nieuweRaadsels() *raadselStaat {
	r := &raadselStaat{}
	r.invoer = textinput.New()
	r.invoer.CharLimit = 60
	r.invoer.Prompt = "› "
	if data, err := os.ReadFile(raadselPad()); err == nil {
		_ = json.Unmarshal(data, &r.gezien)
	}
	return r
}

// volgende: om en om een raadsel en een getallenrij; raadsels die hij al zag komen pas later terug.
func (m *model) volgendRaadsel() tea.Cmd {
	r := m.dk.rd
	lijst := raadselsEN
	if m.inst.Taal == "nl" {
		lijst = raadselsNL
	}
	r.stand, r.poging = 0, ""
	r.invoer.SetValue("")
	if len(r.gezien)%2 == 1 {
		r.huidig = maakRij(rand.New(rand.NewSource(rand.Int63())), m.inst.Taal)
		r.gezien = append(r.gezien, -1)
	} else {
		var kandidaten []int
		recent := map[int]bool{}
		for i := max(len(r.gezien)-2*len(lijst)+4, 0); i < len(r.gezien); i++ {
			recent[r.gezien[i]] = true
		}
		for i := range lijst {
			if !recent[i] {
				kandidaten = append(kandidaten, i)
			}
		}
		if len(kandidaten) == 0 {
			kandidaten = []int{rand.Intn(len(lijst))}
		}
		i := kandidaten[rand.Intn(len(kandidaten))]
		r.huidig = lijst[i]
		r.gezien = append(r.gezien, i)
	}
	if len(r.gezien) > 400 {
		r.gezien = r.gezien[len(r.gezien)-400:]
	}
	if data, err := json.Marshal(r.gezien); err == nil {
		_ = os.MkdirAll(filepath.Dir(raadselPad()), 0o755)
		_ = os.WriteFile(raadselPad(), data, 0o644)
	}
	return r.invoer.Focus()
}

func (m *model) toetsRaadsel(msg tea.KeyMsg) (tea.Cmd, bool) {
	r := m.dk.rd
	switch msg.String() {
	case "esc":
		r.invoer.Blur()
		m.dk.scherm = denkMenu
	case "enter":
		if r.stand >= 2 {
			return m.volgendRaadsel(), true
		}
		poging := strings.TrimSpace(r.invoer.Value())
		if poging == "" {
			return nil, true
		}
		r.poging = poging
		if r.huidig.klopt(poging) {
			r.stand = 3
		}
		r.invoer.SetValue("")
	case "?":
		if r.stand == 0 {
			r.stand = 1
		}
	case "tab":
		if r.stand < 2 {
			r.stand = 2
		} else {
			return m.volgendRaadsel(), true
		}
	default:
		var cmd tea.Cmd
		r.invoer, cmd = r.invoer.Update(msg)
		return cmd, true
	}
	return nil, true
}

func (m model) viewRaadsel(p palet) string {
	r := m.dk.rd
	dim := lipgloss.NewStyle().Foreground(p.dim)
	breed := min(m.breed-8, 90)
	regels := []string{lipgloss.NewStyle().Bold(true).Width(breed - 6).Render(r.huidig.Vraag), ""}
	switch r.stand {
	case 3:
		regels = append(regels, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(m.t("riddle.right", r.huidig.Antwoord[0])))
	case 2:
		regels = append(regels, lipgloss.NewStyle().Foreground(p.acc).Render(m.t("riddle.answer", r.huidig.Antwoord[0])))
	default:
		if r.poging != "" {
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("riddle.notyet", r.poging)))
		}
		if r.stand == 1 {
			regels = append(regels, dim.Render(m.t("riddle.hint")+" ")+r.huidig.Hint)
		}
		regels = append(regels, "", r.invoer.View())
	}
	toetsen := m.t("riddle.keys")
	if r.stand >= 2 {
		toetsen = m.t("riddle.nextkeys")
	}
	titel := m.t("riddle.title")
	if r.huidig.getallen {
		titel = m.t("riddle.sequence")
	}
	return lipgloss.NewStyle().Bold(true).Render(m.t("menu.puzzles")) + "\n\n" + paneel(p, titel, strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(toetsen)
}
