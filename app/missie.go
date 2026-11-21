package main

import (
	"encoding/json"
	"fmt"
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
// Eén keer per week is er een baasgevecht: een rekenmix, vooral met de sommen die nog lastig
// zijn. Winnen geeft een flinke bonus en een beker voor die week. Verliezen kan: morgen mag je
// het opnieuw proberen.
//
// Alleen schoolvakken leveren energie op. Lezen, luisteren, spelletjes en avonturen niet: dat
// doet een kind uit zichzelf.

type missieStand struct {
	Energie  int      `json:"energie"`  // alles bij elkaar, over alle seizoenen
	Gezien   int      `json:"gezien"`   // de energie toen het kind de missie het laatst bekeek
	Gewonnen []string `json:"gewonnen"` // weken (2026-W47) waarin de baas verslagen is
	Verloren string   `json:"verloren"` // datum van het laatste verloren gevecht
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
const bonusBaas = 25

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
	plaatje []string
}

var bazen = []baas{
	{[]string{ // robot
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
	{[]string{ // draak
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
	{[]string{ // piraat
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
	{[]string{ // mistheks
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
}

const (
	baasLevens   = 12 // zoveel goede antwoorden en de baas is verslagen
	baasSchilden = 3  // zoveel fouten mag je maken
)

func weekVan(t time.Time) string {
	j, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", j, w)
}

// welkeBaas: elke week een andere.
func welkeBaas(t time.Time) int {
	_, w := t.ISOWeek()
	return w % len(bazen)
}

func (s missieStand) baasVerslagen(t time.Time) bool {
	w := weekVan(t)
	for _, g := range s.Gewonnen {
		if g == w {
			return true
		}
	}
	return false
}

// baasKan: mag het kind nu vechten? Eén keer winnen per week; na verliezen de volgende dag weer.
func (s missieStand) baasKan(t time.Time) bool {
	return !s.baasVerslagen(t) && s.Verloren != t.Format("2006-01-02")
}

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

// baasSom: een rekenmix uit de tafels van 2 tot en met 10, vaker wat nog niet zit, en een op de
// drie is een deelsom.
func baasSom(v tafelVoortgang, vorige tafelSom) tafelSom {
	for {
		var kandidaten []tafelSom
		var gewichten []int
		totaal := 0
		delen := rand.IntN(3) == 0
		for t := 2; t <= 10; t++ {
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
		if !s.baasKan(m.nu) {
			if s.baasVerslagen(m.nu) {
				m.zeg(m.t("mission.bosswon"))
			} else {
				m.zeg(m.t("mission.bosstomorrow"))
			}
			return nil, true
		}
		g := &baasGevecht{wie: welkeBaas(m.nu), schilden: baasSchilden}
		g.som = baasSom(laadTafels(), tafelSom{})
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
			s.Gewonnen = append(s.Gewonnen, weekVan(m.nu))
			if len(s.Gewonnen) > 60 {
				s.Gewonnen = s.Gewonnen[len(s.Gewonnen)-60:]
			}
			s.Energie += bonusBaas
			s.bewaar()
			m.zegLang(m.t("mission.won", bonusBaas))
		case g.schilden <= 0:
			g.klaar = true
			s := laadMissie()
			s.Verloren = m.nu.Format("2006-01-02")
			s.bewaar()
		default:
			g.som = baasSom(tv, g.som)
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
	seizoen, klaar, binnen := seizoenEnDelen(s.Energie)
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	tekstBreed := max(breed-34, 24)
	tekst := lipgloss.NewStyle().Width(tekstBreed)

	var r []string
	r = append(r, acc.Render(m.t("mission.season", seizoen)), "")
	switch {
	case ms.weg:
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(tekst.Render(m.t(fmt.Sprintf("mission.story.%d", len(missieDelen)-1)))), "")
		if klaar == 0 {
			r = append(r, tekst.Render(m.t("mission.intro2")), "")
		}
	case len(ms.nieuw) > 0:
		laatste := ms.nieuw[len(ms.nieuw)-1]
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(m.t("mission.fixed", m.t(fmt.Sprintf("mission.part.%d", laatste)))),
			tekst.Render(m.t(fmt.Sprintf("mission.story.%d", laatste))), "")
	case klaar == 0 && seizoen == 1:
		r = append(r, tekst.Render(m.t("mission.intro")), "")
	case klaar == 0:
		r = append(r, tekst.Render(m.t("mission.intro2")), "")
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
	r = append(r, dim.Render(tekst.Render(m.t("mission.how"))), "")
	switch {
	case s.baasVerslagen(m.nu):
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Render(m.t("mission.bosswon")))
	case !s.baasKan(m.nu):
		r = append(r, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("mission.bosstomorrow")))
	default:
		r = append(r, lipgloss.NewStyle().Foreground(p.geel).Bold(true).Render(m.t("mission.bossready", m.t(fmt.Sprintf("mission.boss.%d", welkeBaas(m.nu))))))
	}
	if n := len(s.Gewonnen); n > 0 {
		r = append(r, dim.Render(m.t("mission.cups", n)))
	}
	inhoud := lipgloss.JoinHorizontal(lipgloss.Top, "  "+strings.ReplaceAll(plaatjeTekst(schipPlaatje(klaar)), "\n", "\n  ")+"    ", strings.Join(r, "\n"))
	return kop + "\n\n" + paneel(p, m.t("mission.title"), inhoud, breed, true) + "\n" + dim.Render(m.t("mission.keys"))
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
