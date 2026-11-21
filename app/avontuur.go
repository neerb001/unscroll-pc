package main

import (
	"cmp"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tekstavonturen: verhalen om zelf te lezen, met keuzes. Onderweg moet je dingen oplossen die je
// op school leert: klokkijken, sommen, breuken, een plek op de kaart, een robot programmeren.
// Bij een fout antwoord blijf je bij de vraag en helpt het maatje: eerst een duwtje, dan hoe je
// het kunt beredeneren, en daarna zegt het letter voor letter iets van het antwoord voor (nooit
// alles: intypen doe je zelf). Met ? vraag je zelf om een tip. Er zijn meer eindes, dus je kunt
// een avontuur vaker spelen.
//
// Een avontuur is een tekstbestand in app/avonturen:
//
//	titel: Het geheim van de klokkentoren
//	over: één zin voor in de lijst
//
//	# scène-naam
//	Alinea's tekst, gescheiden door een lege regel.
//	> Een keuze -> volgende-scène [+zaklamp] [-sleutel] ?zaklamp !sleutel
//	? Een vraag -> antwoord/ander-antwoord -> goed-scène
//	tip: een duwtje in de goede richting (minstens twee tips, van zacht naar sterk)
//	tip: hoe je het kunt beredeneren
//	anders: Geef het op -> andere-scène   (een uitweg met Tab)
//	kaart: nederland stad:Utrecht
//	robot: 6x4 start=0,3,oost doel=5,0 muur=2,1 2,2 -> goed-scène
//	== einde: hoe dit einde heet
//	| een tekening van tekens: regels die met | beginnen, spaties blijven staan
//	% ..rrrr..   een pixelplaatje in kleur: elke letter is een pixel (zie plaatjeKleuren)
//
// [+ding] krijg je, [-ding] raak je kwijt, ?ding alleen als je het hebt, !ding alleen als je
// het niet hebt. Regels die met // beginnen zijn commentaar (bijvoorbeeld de oplossing van een
// robotpuzzel, voor de tests).

//go:embed avonturen/*.txt
var avonturenBestanden embed.FS

type avKeuze struct {
	tekst, doel      string
	krijg, verlies   []string
	nodig, nietNodig []string
}

type avScene struct {
	id        string
	alineas   []string
	keuzes    []avKeuze
	vraag     string   // een antwoord intypen
	goed      []string // goede antwoorden
	naarGoed  string
	naarFout  string
	kaart     string   // "nederland stad:Utrecht"
	robot     string   // de robotpuzzel
	einde     string   // naam van het einde, als dit een einde is
	tips      []string // hints van het maatje bij een vraag of robotpuzzel, van zacht naar sterk
	anders    *avKeuze // bij een vraag: een uitweg met Tab (bijvoorbeeld "geef het op")
	oplossing string   // bij een robotpuzzel: een programma dat werkt (uit // oplossing)
}

type avontuur struct {
	id, titel, over string
	scenes          map[string]*avScene
	begin           string
	eindes          []string
}

func leesAvontuur(id, tekst string) (*avontuur, error) {
	a := &avontuur{id: id, scenes: map[string]*avScene{}}
	var s *avScene
	var alinea, tekening, plaatje []string
	oplossingen := map[string]string{}
	klaarAlinea := func() {
		if s != nil && len(alinea) > 0 {
			s.alineas = append(s.alineas, strings.Join(alinea, " "))
		}
		if s != nil && len(tekening) > 0 {
			s.alineas = append(s.alineas, tekeningTeken+strings.Join(tekening, "\n"))
		}
		if s != nil && len(plaatje) > 0 {
			s.alineas = append(s.alineas, plaatjeTeken+strings.Join(plaatje, "\n"))
		}
		alinea, tekening, plaatje = nil, nil, nil
	}
	for nr, regel := range strings.Split(strings.ReplaceAll(tekst, "\r", ""), "\n") {
		r := strings.TrimSpace(regel)
		if s != nil && strings.HasPrefix(strings.TrimLeft(regel, " \t"), "|") {
			if len(alinea) > 0 { // een tekening begint een nieuwe alinea
				klaarAlinea()
			}
			lijn := strings.TrimLeft(regel, " \t")[1:]
			tekening = append(tekening, strings.TrimRight(strings.TrimPrefix(lijn, " "), " "))
			continue
		}
		if s != nil && strings.HasPrefix(strings.TrimLeft(regel, " \t"), "%") {
			if len(alinea) > 0 || len(tekening) > 0 {
				klaarAlinea()
			}
			plaatje = append(plaatje, strings.TrimSpace(strings.TrimLeft(regel, " \t")[1:]))
			continue
		}
		if len(tekening) > 0 || len(plaatje) > 0 {
			klaarAlinea()
		}
		switch {
		case strings.HasPrefix(r, "//"):
			if m := oplossingRegel.FindStringSubmatch(r); m != nil {
				oplossingen[m[1]] = strings.TrimSpace(m[2])
			}
		case s == nil && strings.HasPrefix(r, "titel:"):
			a.titel = strings.TrimSpace(strings.TrimPrefix(r, "titel:"))
		case s == nil && strings.HasPrefix(r, "over:"):
			a.over = strings.TrimSpace(strings.TrimPrefix(r, "over:"))
		case strings.HasPrefix(r, "# "):
			klaarAlinea()
			s = &avScene{id: strings.TrimSpace(r[2:])}
			if _, dubbel := a.scenes[s.id]; dubbel {
				return nil, fmt.Errorf("regel %d: scène %q staat er twee keer in", nr+1, s.id)
			}
			a.scenes[s.id] = s
			if a.begin == "" {
				a.begin = s.id
			}
		case s == nil:
		case r == "":
			klaarAlinea()
		case strings.HasPrefix(r, "> "):
			klaarAlinea()
			k, err := leesKeuze(r[2:])
			if err != nil {
				return nil, fmt.Errorf("regel %d: %w", nr+1, err)
			}
			s.keuzes = append(s.keuzes, k)
		case strings.HasPrefix(r, "? "):
			klaarAlinea()
			delen := strings.Split(r[2:], "->")
			if len(delen) != 3 {
				return nil, fmt.Errorf("regel %d: een vraag is: ? vraag -> antwoord -> goed | fout", nr+1)
			}
			naar := strings.Split(delen[2], "|")
			if len(naar) > 2 {
				return nil, fmt.Errorf("regel %d: na het antwoord: de scène als het goed is", nr+1)
			}
			naar = append(naar, "")
			s.vraag = strings.TrimSpace(delen[0])
			for _, g := range strings.Split(delen[1], "/") {
				s.goed = append(s.goed, strings.TrimSpace(g))
			}
			s.naarGoed, s.naarFout = strings.TrimSpace(naar[0]), strings.TrimSpace(naar[1])
		case strings.HasPrefix(r, "tip:"):
			klaarAlinea()
			s.tips = append(s.tips, strings.TrimSpace(strings.TrimPrefix(r, "tip:")))
		case strings.HasPrefix(r, "anders:"):
			klaarAlinea()
			k, err := leesKeuze(strings.TrimSpace(strings.TrimPrefix(r, "anders:")))
			if err != nil {
				return nil, fmt.Errorf("regel %d: %w", nr+1, err)
			}
			s.anders = &k
		case strings.HasPrefix(r, "kaart:"):
			klaarAlinea()
			s.kaart = strings.TrimSpace(strings.TrimPrefix(r, "kaart:"))
		case strings.HasPrefix(r, "robot:"):
			klaarAlinea()
			spec, naar, ok := strings.Cut(strings.TrimPrefix(r, "robot:"), "->")
			delen := append(strings.Split(naar, "|"), "")
			if !ok || len(delen) > 3 {
				return nil, fmt.Errorf("regel %d: robot: rooster -> goed", nr+1)
			}
			s.robot = strings.TrimSpace(spec)
			s.naarGoed, s.naarFout = strings.TrimSpace(delen[0]), strings.TrimSpace(delen[1])
			if _, err := leesRooster(s.robot); err != nil {
				return nil, fmt.Errorf("regel %d: %w", nr+1, err)
			}
		case strings.HasPrefix(r, "== einde:"):
			klaarAlinea()
			s.einde = strings.TrimSpace(strings.TrimPrefix(r, "== einde:"))
			a.eindes = append(a.eindes, s.einde)
		default:
			alinea = append(alinea, r)
		}
	}
	klaarAlinea()
	for id, op := range oplossingen {
		if sc := a.scenes[id]; sc != nil {
			sc.oplossing = op
		}
	}
	if a.titel == "" || a.begin == "" {
		return nil, fmt.Errorf("titel of scènes ontbreken")
	}
	return a, nil
}

// oplossingRegel: // oplossing scène: programma (voor de tests, en voor de laatste hints)
var oplossingRegel = regexp.MustCompile(`^//\s*oplossing\s+(\S+):\s*(.+)$`)

// tekeningTeken: een alinea die zo begint is een tekening (niet afbreken, wel kleuren).
const tekeningTeken = "\x00tekening\n"

// plaatjeTeken: een alinea die zo begint is een pixelplaatje.
const plaatjeTeken = "\x00plaatje\n"

// plaatjeKleuren: de letters in een pixelplaatje. Een punt is doorzichtig.
var plaatjeKleuren = map[rune]string{
	'k': "#1b1b1b", 'w': "#f2f2f2", 's': "#8a8a8a", 'd': "#4a4a4a", 'r': "#d63a3a", 'o': "#f28c28",
	'y': "#f5d142", 'g': "#4caf50", 'l': "#8bd16a", 'n': "#8b5a2b", 'b': "#3a6fd6", 'c': "#5fc8e8",
	'p': "#e88ac0", 'v': "#8e5bd6", 'h': "#f0c8a0",
}

// plaatjeTekst: twee pixelregels per tekstregel, met halve blokjes in kleur.
func plaatjeTekst(regels []string) string {
	breed := 0
	for _, r := range regels {
		breed = max(breed, len([]rune(r)))
	}
	kleur := func(y, x int) string {
		if y >= len(regels) {
			return ""
		}
		r := []rune(regels[y])
		if x >= len(r) {
			return ""
		}
		return plaatjeKleuren[r[x]]
	}
	var uit []string
	for y := 0; y < len(regels); y += 2 {
		var r kleurRegel
		for x := 0; x < breed; x++ {
			boven, onder := kleur(y, x), kleur(y+1, x)
			switch {
			case boven == "" && onder == "":
				r.zet("", "", false, " ")
			case boven == "":
				r.zet(onder, "", false, "▄")
			case onder == "" || onder == boven:
				r.zet(boven, "", false, map[bool]string{true: "█", false: "▀"}[onder == boven])
			default:
				r.zet(boven, onder, false, "▀")
			}
		}
		uit = append(uit, r.klaar())
	}
	return strings.Join(uit, "\n")
}

func leesKeuze(r string) (avKeuze, error) {
	tekst, rest, ok := strings.Cut(r, "->")
	if !ok {
		return avKeuze{}, fmt.Errorf("een keuze is: > tekst -> scène")
	}
	k := avKeuze{tekst: strings.TrimSpace(tekst)}
	for _, w := range strings.Fields(rest) {
		switch {
		case strings.HasPrefix(w, "[+") && strings.HasSuffix(w, "]"):
			k.krijg = append(k.krijg, w[2:len(w)-1])
		case strings.HasPrefix(w, "[-") && strings.HasSuffix(w, "]"):
			k.verlies = append(k.verlies, w[2:len(w)-1])
		case strings.HasPrefix(w, "?"):
			k.nodig = append(k.nodig, w[1:])
		case strings.HasPrefix(w, "!"):
			k.nietNodig = append(k.nietNodig, w[1:])
		case k.doel == "":
			k.doel = w
		default:
			return k, fmt.Errorf("onbekend in een keuze: %q", w)
		}
	}
	if k.doel == "" {
		return k, fmt.Errorf("een keuze zonder scène")
	}
	return k, nil
}

func alleAvonturen() []*avontuur {
	namen, _ := avonturenBestanden.ReadDir("avonturen")
	var uit []*avontuur
	for _, n := range namen {
		data, err := avonturenBestanden.ReadFile(path.Join("avonturen", n.Name()))
		if err != nil {
			continue
		}
		if a, err := leesAvontuur(strings.TrimSuffix(n.Name(), ".txt"), string(data)); err == nil {
			uit = append(uit, a)
		}
	}
	sort.Slice(uit, func(i, j int) bool { return uit[i].id < uit[j].id })
	return uit
}

// antwoordGoed: hoofdletters, accenten, spaties en komma of punt maken niet uit.
func antwoordGoed(antwoord string, goed []string) bool {
	vorm := func(s string) string { // de punt blijft staan: 2,5 is 2.5, maar niet 25
		return strings.NewReplacer(",", ".", " ", "", "-", "", "'", "").Replace(zoekVorm(strings.TrimSpace(s)))
	}
	a := vorm(antwoord)
	for _, g := range goed {
		if a != "" && vorm(g) == a {
			return true
		}
	}
	return false
}

/* ---------------- de robot ---------------- */

type rooster struct {
	b, h         int
	x, y, kijk   int // kijk: 0 noord, 1 oost, 2 zuid, 3 west
	doelX, doelY int
	muur         map[[2]int]bool
}

func leesRooster(spec string) (rooster, error) {
	r := rooster{muur: map[[2]int]bool{}}
	velden := strings.Fields(spec)
	if len(velden) == 0 {
		return r, fmt.Errorf("robot: geen rooster")
	}
	if _, err := fmt.Sscanf(velden[0], "%dx%d", &r.b, &r.h); err != nil || r.b < 2 || r.h < 2 || r.b > 12 || r.h > 8 {
		return r, fmt.Errorf("robot: maat als 6x4 (hooguit 12x8)")
	}
	richting := map[string]int{"noord": 0, "oost": 1, "zuid": 2, "west": 3}
	inMuur := false
	for _, v := range velden[1:] {
		sleutel, waarde, isVeld := strings.Cut(v, "=")
		if !isVeld && inMuur {
			sleutel, waarde = "muur", v
		}
		punt := func(s string) (int, int, error) {
			var x, y int
			_, err := fmt.Sscanf(s, "%d,%d", &x, &y)
			if err != nil || x < 0 || y < 0 || x >= r.b || y >= r.h {
				return 0, 0, fmt.Errorf("robot: %q ligt niet in het rooster", s)
			}
			return x, y, nil
		}
		switch sleutel {
		case "start":
			delen := strings.Split(waarde, ",")
			if len(delen) != 3 {
				return r, fmt.Errorf("robot: start=x,y,richting")
			}
			x, y, err := punt(delen[0] + "," + delen[1])
			k, ok := richting[delen[2]]
			if err != nil || !ok {
				return r, fmt.Errorf("robot: start=x,y,richting (noord, oost, zuid, west)")
			}
			r.x, r.y, r.kijk = x, y, k
			inMuur = false
		case "doel":
			x, y, err := punt(waarde)
			if err != nil {
				return r, err
			}
			r.doelX, r.doelY = x, y
			inMuur = false
		case "muur":
			x, y, err := punt(waarde)
			if err != nil {
				return r, err
			}
			r.muur[[2]int{x, y}] = true
			inMuur = true
		default:
			return r, fmt.Errorf("robot: onbekend %q", v)
		}
	}
	return r, nil
}

// draaiRobot voert een programmaatje uit: "vooruit 2, rechts, herhaal 3: vooruit 1, links".
// Bij herhaal hoort alles na de dubbele punt tot het einde. Geeft de route en of het doel
// gehaald is, of een uitleg als het programma niet klopt.
func draaiRobot(r rooster, programma string) (route [][2]int, gehaald bool, fout string) {
	stappen, fout := leesProgramma(programma, 0)
	if fout != "" {
		return nil, false, fout
	}
	if len(stappen) > 200 {
		return nil, false, "zoveel stappen kan de robot niet onthouden"
	}
	dx := [4]int{0, 1, 0, -1}
	dy := [4]int{-1, 0, 1, 0}
	x, y, kijk := r.x, r.y, r.kijk
	route = append(route, [2]int{x, y})
	for _, s := range stappen {
		switch s {
		case "L":
			kijk = (kijk + 3) % 4
		case "R":
			kijk = (kijk + 1) % 4
		case "V":
			nx, ny := x+dx[kijk], y+dy[kijk]
			if nx < 0 || ny < 0 || nx >= r.b || ny >= r.h || r.muur[[2]int{nx, ny}] {
				return route, false, "bonk"
			}
			x, y = nx, ny
			route = append(route, [2]int{x, y})
		}
	}
	return route, x == r.doelX && y == r.doelY, ""
}

func leesProgramma(p string, diepte int) ([]string, string) {
	if diepte > 2 {
		return nil, "herhaal in herhaal in herhaal is te ingewikkeld"
	}
	p = strings.ToLower(strings.TrimSpace(p))
	var uit []string
	for p != "" {
		var deel string
		if strings.HasPrefix(p, "herhaal") {
			kop, rest, ok := strings.Cut(p, ":")
			if !ok {
				return nil, "na herhaal en een getal komt een dubbele punt, bijvoorbeeld: herhaal 3: vooruit 1"
			}
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(kop, "herhaal")))
			if err != nil || n < 1 || n > 20 {
				return nil, "na herhaal komt een getal van 1 tot 20"
			}
			binnen, f := leesProgramma(rest, diepte+1)
			if f != "" {
				return nil, f
			}
			for i := 0; i < n; i++ {
				uit = append(uit, binnen...)
			}
			return uit, ""
		}
		deel, p, _ = strings.Cut(p, ",")
		p = strings.TrimSpace(p)
		w := strings.Fields(deel)
		switch {
		case len(w) == 0:
		case w[0] == "links":
			uit = append(uit, "L")
		case w[0] == "rechts":
			uit = append(uit, "R")
		case w[0] == "vooruit":
			n := 1
			if len(w) > 1 {
				var err error
				if n, err = strconv.Atoi(w[1]); err != nil || n < 1 || n > 20 {
					return nil, "na vooruit komt een getal van 1 tot 20"
				}
			}
			for i := 0; i < n; i++ {
				uit = append(uit, "V")
			}
		default:
			return nil, fmt.Sprintf("de robot kent %q niet; wel: vooruit, links, rechts, herhaal", w[0])
		}
	}
	return uit, ""
}

// roosterTekst tekent het rooster met de robot, de muren, het doel en de gelopen route.
func roosterTekst(r rooster, route [][2]int, p palet) string {
	op := map[[2]int]bool{}
	for _, pt := range route {
		op[pt] = true
	}
	eind := [2]int{r.x, r.y}
	if len(route) > 0 {
		eind = route[len(route)-1]
	}
	pijl := [4]string{"▲", "▶", "▼", "◀"}
	if alleenLetters() {
		pijl = [4]string{"^", ">", "v", "<"}
	}
	stijl := func(kleur lipgloss.TerminalColor, s string) string {
		return lipgloss.NewStyle().Foreground(kleur).Bold(true).Render(s)
	}
	var regels []string
	for y := 0; y < r.h; y++ {
		var b strings.Builder
		for x := 0; x < r.b; x++ {
			pt := [2]int{x, y}
			switch {
			case len(route) == 0 && pt == [2]int{r.x, r.y}:
				b.WriteString(" " + stijl(p.acc, pijl[r.kijk]) + " ")
			case len(route) > 0 && pt == eind:
				b.WriteString(" " + stijl(p.acc, "●") + " ")
			case r.muur[pt]:
				b.WriteString(stijl(p.dim, "███"))
			case pt == [2]int{r.doelX, r.doelY}:
				b.WriteString(" " + stijl(p.geel, "★") + " ")
			case op[pt]:
				b.WriteString(" " + stijl(p.groen, "·") + " ")
			default:
				b.WriteString(lipgloss.NewStyle().Foreground(p.lijn).Render(" □ "))
			}
		}
		regels = append(regels, b.String())
	}
	return strings.Join(regels, "\n")
}

/* ---------------- bewaren ---------------- */

type avGeheugen struct {
	Plek   map[string]string   `json:"plek"`   // per avontuur: de scène waar je bent
	Spul   map[string][]string `json:"spul"`   // per avontuur: wat je bij je hebt
	Eindes map[string][]string `json:"eindes"` // per avontuur: welke eindes je gevonden hebt
}

func avPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "avonturen.json") }

func laadAvGeheugen() avGeheugen {
	g := avGeheugen{}
	if data, err := os.ReadFile(avPad()); err == nil {
		_ = json.Unmarshal(data, &g)
	}
	if g.Plek == nil {
		g.Plek = map[string]string{}
	}
	if g.Spul == nil {
		g.Spul = map[string][]string{}
	}
	if g.Eindes == nil {
		g.Eindes = map[string][]string{}
	}
	return g
}

func (g avGeheugen) bewaar() {
	_ = os.MkdirAll(filepath.Dir(avPad()), 0o755)
	data, _ := json.Marshal(g)
	_ = veiligSchrijven(avPad(), data, 0o644)
}

/* ---------------- spelen ---------------- */

type avStaat struct {
	av      *avontuur
	scene   string
	spul    []string
	invoer  textinput.Model
	terug   string // na een fout antwoord of een botsende robot: wat er misging
	route   [][2]int
	boven   int  // gescrold in de tekst
	opnieuw bool // n is één keer ingedrukt: nog een keer om opnieuw te beginnen
	// het maatje dat helpt bij een vraag
	tip    string // wat het maatje nu zegt
	hint   int    // hoeveel tips al gegeven
	masker int    // hoeveel letters (of opdrachten) van het antwoord al voorgezegd
	vorige string // het vorige foute antwoord: hetzelfde nog eens telt niet als nieuwe poging
}

func (s *avStaat) heeft(ding string) bool {
	for _, d := range s.spul {
		if d == ding {
			return true
		}
	}
	return false
}

// zichtbareKeuzes: de keuzes die nu kunnen (met wat je bij je hebt).
func (s *avStaat) zichtbareKeuzes() []avKeuze {
	var uit []avKeuze
	for _, k := range s.av.scenes[s.scene].keuzes {
		ok := true
		for _, n := range k.nodig {
			ok = ok && s.heeft(n)
		}
		for _, n := range k.nietNodig {
			ok = ok && !s.heeft(n)
		}
		if ok {
			uit = append(uit, k)
		}
	}
	return uit
}

func avontuurItems() []item {
	g := laadAvGeheugen()
	var uit []item
	for _, a := range alleAvonturen() {
		info := fmt.Sprintf("%d/%d", len(g.Eindes[a.id]), len(a.eindes))
		uit = append(uit, item{id: "avontuur:" + a.id, titel: a.titel, info: info, klaar: len(g.Eindes[a.id]) == len(a.eindes), soort: soortEboek})
	}
	return uit
}

func (m *model) openAvontuur(id string) tea.Cmd {
	for _, a := range alleAvonturen() {
		if "avontuur:"+a.id != id {
			continue
		}
		g := laadAvGeheugen()
		s := &avStaat{av: a, scene: a.begin}
		if plek, ok := g.Plek[a.id]; ok && a.scenes[plek] != nil {
			s.scene, s.spul = plek, g.Spul[a.id]
		}
		s.invoer = textinput.New()
		s.invoer.CharLimit = 80
		s.invoer.Prompt = "➜ "
		m.ls.avontuur = s
		return m.avNaarScene(s.scene)
	}
	return nil
}

// avNaarScene gaat naar een scène, bewaart de plek, en onthoudt een einde.
func (m *model) avNaarScene(id string) tea.Cmd {
	s := m.ls.avontuur
	sc := s.av.scenes[id]
	if sc == nil {
		return nil
	}
	s.scene, s.boven, s.route = id, 0, nil
	s.tip, s.hint, s.masker, s.vorige = "", 0, 0, ""
	s.invoer.SetValue("")
	g := laadAvGeheugen()
	g.Plek[s.av.id], g.Spul[s.av.id] = id, s.spul
	if sc.einde != "" {
		gevonden := false
		for _, e := range g.Eindes[s.av.id] {
			gevonden = gevonden || e == sc.einde
		}
		if !gevonden {
			g.Eindes[s.av.id] = append(g.Eindes[s.av.id], sc.einde)
		}
		delete(g.Plek, s.av.id) // de volgende keer begin je opnieuw
		delete(g.Spul, s.av.id)
	}
	g.bewaar()
	if sc.vraag != "" || sc.robot != "" {
		return s.invoer.Focus()
	}
	s.invoer.Blur()
	return nil
}

func (m *model) toetsAvontuur(msg tea.KeyMsg) tea.Cmd {
	s := m.ls.avontuur
	sc := s.av.scenes[s.scene]
	k := msg.String()
	pagina := max(m.hoog-14, 4)
	switch k {
	case "esc":
		m.ls.avontuur = nil
		m.ls.avonturen = m.avonturenLijstVan(m.ls.avonturen.sel)
		return nil
	case "pgup", "pgdown", "up", "down":
		scrollToets(k, &s.boven, pagina)
		return nil
	}
	invoerScene := sc.vraag != "" || sc.robot != ""
	if k == "n" && !invoerScene && sc.einde == "" {
		if !s.opnieuw {
			s.opnieuw = true
			s.terug = m.t("adv.restartask")
			return nil
		}
		s.opnieuw, s.spul, s.terug = false, nil, ""
		return m.avNaarScene(s.av.begin)
	}
	s.opnieuw = false
	if sc.einde != "" {
		switch k {
		case "1", "enter": // opnieuw
			s.spul, s.terug = nil, ""
			return m.avNaarScene(s.av.begin)
		case "2":
			m.ls.avontuur = nil
			m.ls.avonturen = m.avonturenLijstVan(m.ls.avonturen.sel)
		}
		return nil
	}
	if sc.vraag != "" || sc.robot != "" {
		switch k {
		case "?":
			m.avTip(false)
			return nil
		case "tab":
			if sc.anders != nil {
				s.terug = ""
				return m.avNaarScene(sc.anders.doel)
			}
			return nil
		}
		if k != "enter" {
			var cmd tea.Cmd
			s.invoer, cmd = s.invoer.Update(msg)
			return cmd
		}
		antwoord := strings.TrimSpace(s.invoer.Value())
		if antwoord == "" {
			return nil
		}
		if sc.robot != "" {
			r, _ := leesRooster(sc.robot)
			route, gehaald, fout := draaiRobot(r, antwoord)
			switch {
			case fout != "" && fout != "bonk":
				s.terug = fout
				return nil
			case gehaald:
				s.terug = ""
				return m.avNaarScene(sc.naarGoed)
			}
			s.terug, s.route = m.t("adv.robotmiss"), route
			if fout == "bonk" {
				s.terug = m.t("adv.robotbonk")
			}
			m.avFout(antwoord)
			return nil
		}
		if antwoordGoed(antwoord, sc.goed) {
			s.terug = ""
			return m.avNaarScene(sc.naarGoed)
		}
		s.terug = m.t("adv.wrong", antwoord)
		m.avFout(antwoord)
		return nil
	}
	if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
		keuzes := s.zichtbareKeuzes()
		n := int(k[0] - '1')
		if n < len(keuzes) {
			kz := keuzes[n]
			for _, d := range kz.verlies {
				var rest []string
				for _, x := range s.spul {
					if x != d {
						rest = append(rest, x)
					}
				}
				s.spul = rest
			}
			for _, d := range kz.krijg {
				if !s.heeft(d) {
					s.spul = append(s.spul, d)
				}
			}
			s.terug = ""
			return m.avNaarScene(kz.doel)
		}
	}
	return nil
}

func (m model) viewAvontuur(p palet) string {
	s := m.ls.avontuur
	sc := s.av.scenes[s.scene]
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	breed := max(min(m.breed-10, 90), 30)
	tekst := lipgloss.NewStyle().Width(breed)

	var boven []string // het verhaal
	for _, a := range sc.alineas {
		if t, ok := strings.CutPrefix(a, plaatjeTeken); ok {
			boven = append(boven, plaatjeTekst(strings.Split(t, "\n")), "")
			continue
		}
		if t, ok := strings.CutPrefix(a, tekeningTeken); ok {
			boven = append(boven, lipgloss.NewStyle().Foreground(p.acc).Render(t), "")
			continue
		}
		boven = append(boven, tekst.Render(a), "")
	}
	if sc.kaart != "" {
		kaart, code, _ := strings.Cut(sc.kaart, " ")
		if k := topoKaartVan(kaart); len(k.Items) > 0 {
			b, h := min(breed, 70), 12
			kader := k.Kader
			for i := range k.Items {
				if k.Items[i].Code == strings.TrimSpace(code) {
					kader = topoZoom(kaart, &k.Items[i], b, h)
				}
			}
			boven = append(boven, topoTekst(kaart, strings.TrimSpace(code), false, false, b, h, p, m.inst.Taal, kader), "")
		}
	}
	if sc.robot != "" {
		r, _ := leesRooster(sc.robot)
		boven = append(boven, roosterTekst(r, s.route, p), "",
			dim.Render(m.t("adv.robothelp")), "")
	}
	verhaal := strings.TrimRight(strings.Join(boven, "\n"), "\n")

	// onder: wat je kunt doen
	var onder []string
	if s.terug != "" {
		onder = append(onder, lipgloss.NewStyle().Foreground(p.geel).Width(breed).Render(s.terug), "")
	}
	toetsen := m.t("adv.keys")
	switch {
	case sc.einde != "":
		g := laadAvGeheugen()
		onder = append(onder, acc.Render(m.t("adv.end", sc.einde)),
			dim.Render(m.t("adv.endings", len(g.Eindes[s.av.id]), len(s.av.eindes))), "",
			"  1  "+m.t("adv.again"), "  2  "+m.t("adv.back"))
		toetsen = m.t("adv.endkeys")
	case sc.vraag != "" || sc.robot != "":
		if s.tip != "" {
			gezicht := lipgloss.NewStyle().Foreground(p.acc2).Bold(true).Render(gezichten[m.inst.Gezicht] + " " + cmp.Or(m.inst.Maatje, "Bliep") + ":")
			onder = append(onder, gezicht+" "+lipgloss.NewStyle().Width(max(breed-lipgloss.Width(gezicht)-1, 20)).Render(s.tip), "")
		}
		if sc.vraag != "" {
			onder = append(onder, lipgloss.NewStyle().Bold(true).Width(breed).Render(sc.vraag))
		}
		onder = append(onder, s.invoer.View())
		toetsen = m.t("adv.answerkeys")
		if sc.anders != nil {
			toetsen += " · " + m.t("adv.other", sc.anders.tekst)
		}
	default:
		for i, k := range s.zichtbareKeuzes() {
			onder = append(onder, acc.Render(fmt.Sprintf("  %d  ", i+1))+k.tekst)
		}
	}
	if len(s.spul) > 0 {
		onder = append(onder, "", dim.Render(m.t("adv.bag", strings.Join(s.spul, ", "))))
	}
	hoogte := max(m.hoog-8-len(onder), 4)
	kop := lipgloss.NewStyle().Bold(true).Render(s.av.titel)
	inhoud := venster(verhaal, &s.boven, hoogte, dim.Render(m.t("scroll.more"))) + "\n\n" + strings.Join(onder, "\n")
	return lipgloss.PlaceHorizontal(m.breed-4, lipgloss.Center, kop+"\n\n"+lipgloss.NewStyle().Width(breed).Render(inhoud)) +
		"\n" + lipgloss.PlaceHorizontal(m.breed-4, lipgloss.Center, dim.Render(toetsen))
}

/* ---------------- het maatje helpt ---------------- */

// avFout: een fout antwoord. Je blijft bij de vraag en het maatje geeft de volgende tip. Een
// nieuwe poging is nodig: hetzelfde foute antwoord nog eens telt niet.
func (m *model) avFout(antwoord string) {
	s := m.ls.avontuur
	s.invoer.SetValue("")
	if zoekVorm(antwoord) == s.vorige {
		return
	}
	s.vorige = zoekVorm(antwoord)
	m.avTip(true)
}

// avTip: de volgende hint. Eerst de tips uit het verhaal, van zacht naar sterk (met ? kun je
// er zelf om vragen). Zijn die op, dan zegt het maatje na elke nieuwe foute poging één letter
// (of opdracht) meer van het antwoord voor, maar nooit alles: intypen doe je zelf.
func (m *model) avTip(doorPoging bool) {
	s := m.ls.avontuur
	sc := s.av.scenes[s.scene]
	tips := sc.tips
	if fout := s.av.scenes[sc.naarFout]; fout != nil { // oud formaat: de fout-scène als uitleg
		for _, a := range fout.alineas {
			if !strings.HasPrefix(a, plaatjeTeken) && !strings.HasPrefix(a, tekeningTeken) {
				tips = append(tips, a)
			}
		}
	}
	if s.hint < len(tips) {
		s.tip = tips[s.hint]
		s.hint++
		return
	}
	if !doorPoging {
		s.tip = m.t("adv.tiptry")
		if len(tips) > 0 {
			s.tip = tips[len(tips)-1] + " " + s.tip
		}
		return
	}
	s.masker++
	if sc.robot != "" {
		if voor := maskerProgramma(sc.oplossing, s.masker); voor != "" {
			s.tip = m.t("adv.tiprobot", voor)
		}
		return
	}
	s.tip = m.t("adv.tipmask", maskerAntwoord(sc.goed[0], s.masker))
}

// maskerAntwoord: de eerste n letters of cijfers, de rest als streepjes. Nooit het hele woord:
// de laatste letter blijft altijd verborgen. Spaties en leestekens staan er gewoon.
func maskerAntwoord(antwoord string, n int) string {
	r := []rune(antwoord)
	letters := 0
	for _, c := range r {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			letters++
		}
	}
	n = min(n, letters-1)
	var b []string
	gezien := 0
	for _, c := range r {
		switch {
		case unicode.IsLetter(c) || unicode.IsDigit(c):
			if gezien < n {
				b = append(b, string(c))
			} else {
				b = append(b, "_")
			}
			gezien++
		case c == ' ':
			b = append(b, " ")
		default:
			b = append(b, string(c))
		}
	}
	return strings.Join(b, " ")
}

// maskerProgramma: de eerste n opdrachten van de oplossing, en ... voor de rest.
func maskerProgramma(oplossing string, n int) string {
	var delen []string
	for _, d := range strings.Split(oplossing, ",") {
		if d = strings.TrimSpace(d); d != "" {
			delen = append(delen, d)
		}
	}
	if len(delen) == 0 {
		return ""
	}
	n = max(1, min(n, len(delen)-1))
	if len(delen) == 1 {
		n = 0
	}
	return strings.Join(append(delen[:n:n], "..."), ", ")
}

func (m *model) avonturenLijst() lijst { return lijst{items: avontuurItems()} }

// avonturenLijstVan: opnieuw opbouwen (de eindes kunnen veranderd zijn), op dezelfde plek.
func (m *model) avonturenLijstVan(sel int) lijst {
	l := m.avonturenLijst()
	l.sel = max(0, min(sel, len(l.items)-1))
	return l
}
