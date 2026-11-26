package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Missie Bliep: een verhaal dat doorloopt zolang je oefent. Het ruimteschip van Bliep is
// neergestort; elk goed antwoord bij Leren (tafels, topografie, dictee, flitslezen, Engels,
// geschiedenis, en de sommen van Eerst oefenen) geeft energie, en met energie repareer je het
// schip, onderdeel voor onderdeel. Is het af, dan vliegt Bliep weg en begint een nieuw seizoen.
//
// Elke dag staat er een baas klaar: een rekenmix, vooral met de sommen die nog lastig zijn.
// Gevechten die je niet gebruikt, spaar je op (hooguit drie): wie een dag niet kon, heeft de
// volgende dag twee gevechten. Winnen geeft een beker en extra energie; verliezen kost het
// gevecht, en dezelfde baas wacht je de volgende keer weer op.
//
// De verhalen per seizoen staan in app/missies (zie seizoenen.go). Daarna gaan de verhalen
// opnieuw rond; ouders zien in de ouderstand wanneer er nieuwe seizoenen nodig zijn.
//
// Alleen schoolvakken leveren energie op. Lezen, luisteren, spelletjes en avonturen niet: dat
// doet een kind uit zichzelf.

type missieStand struct {
	Energie  int      `json:"energie"`   // alles bij elkaar, over alle seizoenen
	Gezien   int      `json:"gezien"`    // de energie toen het kind de missie het laatst bekeek
	Gewonnen []string `json:"gewonnen"`  // dagen (vroeger weken) waarop een baas verslagen is: de bekers
	Tegoed   int      `json:"tegoed"`    // gevechten die nog gespeeld mogen worden
	TegoedOp string   `json:"tegoed_op"` // de dag waarop het tegoed het laatst is bijgewerkt
	Baas     int      `json:"baas"`      // welke baas nu aan de beurt is (na winnen de volgende)
	Begin    string   `json:"begin"`     // de eerste dag met de missie: voor het tempo
}

// maxTegoed: zoveel gevechten kun je hooguit opsparen.
const maxTegoed = 3

// bijwerken: elke nieuwe dag een gevecht erbij, tot maxTegoed.
func (s *missieStand) bijwerken(nu time.Time) {
	vandaag := nu.Format("2006-01-02")
	if s.Begin == "" {
		s.Begin = vandaag
	}
	if s.TegoedOp == "" {
		s.Tegoed, s.TegoedOp = 1, vandaag
		return
	}
	if d := dagenTussen(s.TegoedOp, vandaag); d > 0 {
		s.Tegoed, s.TegoedOp = min(maxTegoed, s.Tegoed+d), vandaag
	}
}

func dagenTussen(van, tot string) int {
	a, err1 := time.ParseInLocation("2006-01-02", van, time.Local)
	b, err2 := time.ParseInLocation("2006-01-02", tot, time.Local)
	if err1 != nil || err2 != nil {
		return 1
	}
	return int(math.Round(b.Sub(a).Hours() / 24))
}

func missiePad() string { return filepath.Join(filepath.Dir(instellingenPad()), "missie.json") }

func laadMissie() missieStand {
	var s missieStand
	if data, err := os.ReadFile(missiePad()); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	return s
}

func (s missieStand) bewaar() {
	_ = os.MkdirAll(filepath.Dir(missiePad()), 0o755)
	data, _ := json.Marshal(s)
	_ = veiligSchrijven(missiePad(), data, 0o644)
}

// missieGoed: een goed antwoord bij een schoolvak. Eén energie erbij.
func missieGoed() {
	s := laadMissie()
	s.Energie++
	s.bewaar()
}

/* ---------------- het schip ---------------- */

// De onderdelen in de volgorde waarin je ze repareert, met hoeveel energie (binnen het seizoen)
// er dan in totaal nodig is. De cijfers in het plaatje horen bij de onderdelen.
type missieDeel struct {
	teken rune
	kleur rune // kleur uit plaatjeKleuren als het gerepareerd is
	nodig int
}

var missieDelen = []missieDeel{
	{'1', 'w', 15},  // de romp
	{'2', 'c', 40},  // het raam
	{'3', 'b', 70},  // de linkervleugel
	{'4', 'b', 100}, // de rechtervleugel
	{'5', 's', 140}, // de motor
	{'7', 'y', 190}, // de antenne
	{'8', 'g', 250}, // de lichten
	{'6', 'o', 320}, // brandstof: de vlam, en weg!
}

// seizoenLengte: zoveel energie kost een heel schip.
var seizoenLengte = missieDelen[len(missieDelen)-1].nodig

// bonusBaas: zoveel energie geeft een verslagen baas.
const bonusBaas = 15

// het schip, linkerhelft; de rechterhelft is het spiegelbeeld (vleugel 3 wordt 4)
var schipHelft = []string{
	".........7",
	".........7",
	"........11",
	".......111",
	"......1122",
	"......1222",
	"......1122",
	"......1111",
	"......1811",
	".....31111",
	"....331111",
	"...3331811",
	"..33331111",
	"..33..1111",
	"......5555",
	".......555",
	".......66.",
	"......666.",
	".......6..",
}

// seizoenEnDelen: in welk seizoen je zit (vanaf 1) en hoeveel onderdelen al klaar zijn.
func seizoenEnDelen(energie int) (seizoen, klaar, binnen int) {
	seizoen = energie/seizoenLengte + 1
	binnen = energie % seizoenLengte
	for _, d := range missieDelen {
		if binnen >= d.nodig {
			klaar++
		}
	}
	return
}

// schipPlaatje: de onderdelen die klaar zijn in kleur, de rest donkergrijs.
func schipPlaatje(klaar int) []string {
	kleur := map[rune]rune{}
	for i, d := range missieDelen {
		kleur[d.teken] = 'd'
		if i < klaar {
			kleur[d.teken] = d.kleur
		}
	}
	var uit []string
	for _, h := range schipHelft {
		r := []rune(h)
		rechts := make([]rune, len(r))
		for i, c := range r {
			if c == '3' {
				c = '4'
			}
			rechts[len(r)-1-i] = c
		}
		var b strings.Builder
		for _, c := range append(r, rechts...) {
			if k, ok := kleur[c]; ok {
				b.WriteRune(k)
			} else {
				b.WriteRune(c)
			}
		}
		uit = append(uit, b.String())
	}
	return uit
}

/* ---------------- de bazen ---------------- */

type baas struct {
	soort   int
	plaatje []string
}

var bazen = []baas{
	{baasMix, []string{ // robot
		".....ssssss.....",
		"....ssssssss....",
		"....srrssrrs....",
		"....ssssssss....",
		"....sskkkkss....",
		".....ssssss.....",
		"..ssssssssssss..",
		".sssbbssssbbsss.",
		".ss.ssssssss.ss.",
		".ss.ssssssss.ss.",
		"....ss....ss....",
		"...ddd....ddd...",
	}},
	{baasMix, []string{ // draak
		"..g..........g..",
		"..gg........gg..",
		"...gggggggggg...",
		"..ggyggggggyggg.",
		"..gggggggggggg..",
		"..gggwgwgwgggg..",
		"...gggggggggg...",
		".r..gggggggg..r.",
		"rr.gggggggggg.rr",
		".rggggllllggggr.",
		"...gggllllggg...",
		"...gg......gg...",
	}},
	{baasMix, []string{ // piraat
		".....kkkkkk.....",
		"....kkkkkkkk....",
		"...kkkwkkwkkk...",
		".....hhhhhh.....",
		"....hkkhhhhh....",
		"....hhhhhhhh....",
		"....hhrrrrhh....",
		".....hhhhhh.....",
		"...rrrrrrrrrr...",
		"..rrwrrrrrrwrr..",
		"..rr.rrrrrr.rr..",
		"....nn....nn....",
	}},
	{baasMix, []string{ // mistheks
		".......vv.......",
		"......vvvv......",
		".....vvvvvv.....",
		"...vvvvvvvvvv...",
		".....hhhhhh.....",
		".....hghhgh.....",
		".....hhhhhh.....",
		"......hrrh......",
		"....vvvvvvvv....",
		"...vvvvvvvvvv...",
		"..cvvvvvvvvvvc..",
		".ccc.vvvvvv.ccc.",
	}},
	{baasDelen, []string{ // generaal
		".....gggggg.....",
		"....gggggggg....",
		"...gggggggggg...",
		"....kkkkkkkk....",
		"....hhhhhhhh....",
		"....hkhhhhkh....",
		"....hhhhhhhh....",
		".....hrrrrh.....",
		"...gggyggyggg...",
		"..ggggyggygggg..",
		"..gg.gggggg.gg..",
		"....kk....kk....",
	}},
	{baasKeer, []string{ // tafeltrol
		"...n........n...",
		"...nn......nn...",
		"....llllllll....",
		"...llllllllll...",
		"...llwkllwklll..",
		"...llllllllll...",
		"...lllwwwwlll...",
		"....llllllll....",
		"..nnnnnnnnnnnn..",
		".lnnnnnnnnnnnnl.",
		".l..nnnnnnnn..l.",
		"....ll....ll....",
	}},
	{baasPittig, []string{ // robo-rex
		"........ssssss..",
		".......ssrsssss.",
		".......ssssssss.",
		".......sssswwww.",
		"......ssssss....",
		"..s..sssssss....",
		"..ssssssssss....",
		"...ssssssssss...",
		"....sssssss.s...",
		"....ss...ss.....",
		"....ss...ss.....",
		"...ddd..ddd.....",
	}},
	{baasMix, []string{ // sneeuwman
		".....dddddd.....",
		".....dddddd.....",
		"....dddddddd....",
		".....wwwwww.....",
		"....wkwwwwkw....",
		"....wwwoowww....",
		".....wwwwww.....",
		"...rrrrrrrrrr...",
		"...wwwwwwwwww...",
		"..wwwwwkwwwwww..",
		"..wwwwwkwwwwww..",
		"...wwwwwwwwww...",
	}},
}

const (
	baasLevens   = 12 // zoveel goede antwoorden en de baas is verslagen
	baasSchilden = 3  // zoveel fouten mag je maken
)

// welkeBaas: de baas die nu aan de beurt is.
func (s missieStand) welkeBaas() int { return ((s.Baas % len(bazen)) + len(bazen)) % len(bazen) }

type baasGevecht struct {
	wie      int
	som      tafelSom
	antwoord string
	raak     int // goede antwoorden
	schilden int
	terug    string
	goed     bool
	klaar    bool // gewonnen of verloren: het verhaal laten zien
	gewonnen bool
}

// Soorten bazen: welke sommen ze geven.
const (
	baasMix    = iota // tafels van 2 tot en met 10, een op de drie is een deelsom
	baasKeer          // alleen keersommen
	baasDelen         // alleen deelsommen
	baasPittig        // de lastige tafels: 6, 7, 8 en 9
)

// baasSom: een som voor dit soort baas, vaker wat nog niet zit.
func baasSom(v tafelVoortgang, vorige tafelSom, soort int) tafelSom {
	van, tot := 2, 10
	if soort == baasPittig {
		van, tot = 6, 9
	}
	for {
		var kandidaten []tafelSom
		var gewichten []int
		totaal := 0
		delen := rand.IntN(3) == 0
		switch soort {
		case baasKeer:
			delen = false
		case baasDelen:
			delen = true
		}
		for t := van; t <= tot; t++ {
			for a := 2; a <= 10; a++ {
				s := tafelSom{a: a, b: t, delen: delen}
				g := 4 - min(v.Reeks[s.sleutel()], 3)
				kandidaten, gewichten = append(kandidaten, s), append(gewichten, g)
				totaal += g
			}
		}
		lot := rand.IntN(totaal)
		for i, g := range gewichten {
			if lot < g {
				if kandidaten[i] != vorige {
					return kandidaten[i]
				}
				break
			}
			lot -= g
		}
	}
}

/* ---------------- het scherm ---------------- */

type missieStaat struct {
	gevecht *baasGevecht
	nieuw   []int // onderdelen die net klaar zijn: even laten zien
	weg     bool  // het schip was af en is weggevlogen: een nieuw seizoen
	klaarNu int   // hoeveel onderdelen klaar waren toen het scherm openging
}

// missieOpen: bij het openen kijken wat er sinds de vorige keer gerepareerd is.
func (m *model) missieOpen() {
	ms := m.lr.missie
	s := laadMissie()
	oudSeizoen, oudKlaar, _ := seizoenEnDelen(s.Gezien)
	seizoen, klaar, _ := seizoenEnDelen(s.Energie)
	ms.nieuw, ms.weg = nil, seizoen > oudSeizoen
	if ms.weg {
		oudKlaar = 0
	}
	for i := oudKlaar; i < klaar; i++ {
		ms.nieuw = append(ms.nieuw, i)
	}
	ms.klaarNu = klaar
	ms.gevecht = nil
	s.Gezien = s.Energie
	s.bijwerken(m.nu)
	s.bewaar()
}

func (m *model) toetsMissie(msg tea.KeyMsg) (tea.Cmd, bool) {
	ms := m.lr.missie
	if g := ms.gevecht; g != nil {
		return m.toetsBaas(msg, g)
	}
	switch msg.String() {
	case "esc":
		m.lr.scherm = lerenMenu
	case "b", "enter":
		s := laadMissie()
		s.bijwerken(m.nu)
		if s.Tegoed <= 0 {
			m.zeg(m.t("mission.bossnone"))
			return nil, true
		}
		g := &baasGevecht{wie: s.welkeBaas(), schilden: baasSchilden}
		g.som = baasSom(laadTafels(), tafelSom{}, bazen[g.wie].soort)
		ms.gevecht = g
	default:
		return nil, false
	}
	return nil, true
}

func (m *model) toetsBaas(msg tea.KeyMsg, g *baasGevecht) (tea.Cmd, bool) {
	k := msg.String()
	if g.klaar {
		if k == "esc" || k == "enter" {
			m.missieOpen()
		}
		return nil, true
	}
	switch {
	case k == "esc":
		m.lr.missie.gevecht = nil // weglopen mag; het telt niet als verloren
	case k == "backspace":
		if g.antwoord != "" {
			g.antwoord = g.antwoord[:len(g.antwoord)-1]
		}
	case msg.Type == tea.KeyRunes && strings.Trim(k, "0123456789") == "":
		g.antwoord = (g.antwoord + k)[:min(len(g.antwoord+k), 3)]
	case k == "enter":
		if g.antwoord == "" {
			return nil, true
		}
		n, _ := strconv.Atoi(g.antwoord)
		tv := laadTafels()
		if n == g.som.uitkomst() {
			g.raak++
			g.goed, g.terug = true, m.t("mission.hit")
			tv.Reeks[g.som.sleutel()]++
		} else {
			g.schilden--
			g.goed, g.terug = false, m.t("mission.miss", g.som.tekst(), g.som.uitkomst())
			tv.Reeks[g.som.sleutel()] = 0
		}
		tv.bewaar()
		g.antwoord = ""
		switch {
		case g.raak >= baasLevens:
			g.klaar, g.gewonnen = true, true
			s := laadMissie()
			s.bijwerken(m.nu)
			s.Gewonnen = append(s.Gewonnen, m.nu.Format("2006-01-02"))
			s.Tegoed = max(s.Tegoed-1, 0)
			s.Baas++
			s.Energie += bonusBaas
			s.bewaar()
			m.zegLang(m.t("mission.won", bonusBaas))
		case g.schilden <= 0:
			g.klaar = true
			s := laadMissie()
			s.bijwerken(m.nu)
			s.Tegoed = max(s.Tegoed-1, 0) // dezelfde baas wacht de volgende keer weer
			s.bewaar()
		default:
			g.som = baasSom(tv, g.som, bazen[g.wie].soort)
		}
	default:
		return nil, false
	}
	return nil, true
}

func (m model) viewMissie(p palet, kop string, breed int) string {
	ms := m.lr.missie
	if g := ms.gevecht; g != nil {
		return m.viewBaas(p, kop, breed, g)
	}
	s := laadMissie()
	s.bijwerken(m.nu)
	seizoen, klaar, binnen := seizoenEnDelen(s.Energie)
	verhaal := seizoenVoor(seizoen)
	taal := m.inst.Taal
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	tekstBreed := max(breed-34, 24)
	tekst := lipgloss.NewStyle().Width(tekstBreed)

	var r []string
	r = append(r, acc.Render(tekst.Render(m.t("mission.season", seizoen, verhaal.zeg("titel", taal)))), "")
	switch {
	case ms.weg:
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(tekst.Render(seizoenVoor(seizoen-1).deel(len(missieDelen)-1, taal))), "")
		if klaar == 0 {
			r = append(r, tekst.Render(verhaal.zeg("intro", taal)), "")
		}
	case len(ms.nieuw) > 0:
		laatste := ms.nieuw[len(ms.nieuw)-1]
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(m.t("mission.fixed", m.t(fmt.Sprintf("mission.part.%d", laatste)))),
			tekst.Render(verhaal.deel(laatste, taal)), "")
	case klaar == 0:
		r = append(r, tekst.Render(verhaal.zeg("intro", taal)), "")
	}
	if klaar < len(missieDelen) {
		d := missieDelen[klaar]
		vorig := 0
		if klaar > 0 {
			vorig = missieDelen[klaar-1].nodig
		}
		r = append(r, m.t("mission.next", m.t(fmt.Sprintf("mission.part.%d", klaar))),
			voortgangsbalk(p, binnen-vorig, d.nodig-vorig, min(24, tekstBreed)),
			dim.Render(m.t("mission.energy", binnen-vorig, d.nodig-vorig)), "")
	}
	if !ms.weg && len(ms.nieuw) == 0 && !m.ouderModus { // niet te hoog worden op een klein scherm
		r = append(r, dim.Render(tekst.Render(m.t("mission.how"))), "")
	}
	if s.Tegoed > 0 {
		r = append(r, lipgloss.NewStyle().Foreground(p.geel).Bold(true).Render(m.t("mission.bossready", m.t(fmt.Sprintf("mission.boss.%d", s.welkeBaas())))))
		if s.Tegoed > 1 {
			r = append(r, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("mission.fights", s.Tegoed)))
		}
	} else {
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Render(tekst.Render(m.t("mission.bossnone"))))
	}
	if n := len(s.Gewonnen); n > 0 {
		r = append(r, dim.Render(m.t("mission.cups", n)))
	}
	if m.ouderModus {
		r = append(r, "", lipgloss.NewStyle().Foreground(p.geel).Render(tekst.Render(m.missieVoorOuders(s))))
	}
	inhoud := lipgloss.JoinHorizontal(lipgloss.Top, "  "+strings.ReplaceAll(plaatjeTekst(schipPlaatje(klaar)), "\n", "\n  ")+"    ", strings.Join(r, "\n"))
	return kop + "\n\n" + paneel(p, m.t("mission.title"), inhoud, breed, true) + "\n" + dim.Render(m.t("mission.keys"))
}

// missieVoorOuders: hoe lang de geschreven verhalen nog meegaan, bij het tempo tot nu toe.
func (m model) missieVoorOuders(s missieStand) string {
	geschreven := len(alleSeizoenen())
	seizoen, _, _ := seizoenEnDelen(s.Energie)
	dagen := max(dagenTussen(s.Begin, m.nu.Format("2006-01-02"))+1, 1)
	tempo := float64(s.Energie) / float64(dagen)
	if seizoen > geschreven {
		return m.t("mission.parentrepeat", seizoen, geschreven)
	}
	over := geschreven*seizoenLengte - s.Energie
	if tempo < 1 || dagen < 3 {
		return m.t("mission.parentnew", seizoen, geschreven)
	}
	return m.t("mission.parent", seizoen, geschreven, int(math.Round(tempo)), int(math.Ceil(float64(over)/tempo)))
}

func (m model) viewBaas(p palet, kop string, breed int, g *baasGevecht) string {
	dim := lipgloss.NewStyle().Foreground(p.dim)
	naam := m.t(fmt.Sprintf("mission.boss.%d", g.wie))
	var r []string
	r = append(r, lipgloss.NewStyle().Foreground(p.rood).Bold(true).Render(naam), "")
	levens := lipgloss.NewStyle().Foreground(p.rood).Render(strings.Repeat("♥", baasLevens-g.raak)) +
		lipgloss.NewStyle().Foreground(p.lijn).Render(strings.Repeat("♡", g.raak))
	schild := lipgloss.NewStyle().Foreground(p.acc).Render(strings.Repeat("◆", max(g.schilden, 0))) +
		lipgloss.NewStyle().Foreground(p.lijn).Render(strings.Repeat("◇", baasSchilden-max(g.schilden, 0)))
	if alleenLetters() {
		levens = lipgloss.NewStyle().Foreground(p.rood).Render(strings.Repeat("#", baasLevens-g.raak)) + strings.Repeat(".", g.raak)
		schild = lipgloss.NewStyle().Foreground(p.acc).Render(strings.Repeat("O", max(g.schilden, 0))) + strings.Repeat(".", baasSchilden-max(g.schilden, 0))
	}
	r = append(r, m.t("mission.bosslife")+"  "+levens, m.t("mission.shields")+"  "+schild, "")
	tekst := lipgloss.NewStyle().Width(max(breed-30, 24))
	switch {
	case g.klaar && g.gewonnen:
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(tekst.Render(m.t("mission.wonstory", naam, bonusBaas))))
	case g.klaar:
		r = append(r, lipgloss.NewStyle().Foreground(p.geel).Render(tekst.Render(m.t("mission.loststory", naam))))
	default:
		if g.raak == 0 && g.terug == "" {
			r = append(r, tekst.Render(m.t(fmt.Sprintf("mission.taunt.%d", g.wie))), "")
		}
		r = append(r, lipgloss.NewStyle().Bold(true).Render(g.som.tekst()+" = ")+lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(g.antwoord+"▌"), "")
		if g.terug != "" {
			kleur := p.groen
			if !g.goed {
				kleur = p.geel
			}
			r = append(r, lipgloss.NewStyle().Foreground(kleur).Render(g.terug))
		}
	}
	toetsen := m.t("mission.bosskeys")
	if g.klaar {
		toetsen = m.t("mission.endkeys")
	}
	inhoud := lipgloss.JoinHorizontal(lipgloss.Top, "  "+strings.ReplaceAll(plaatjeTekst(bazen[g.wie].plaatje), "\n", "\n  ")+"    ", strings.Join(r, "\n"))
	return kop + "\n\n" + paneel(p, m.t("mission.bosstitle"), inhoud, breed, true) + "\n" + dim.Render(toetsen)
}
