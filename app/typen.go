package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Typcursus: rustige lessen zonder klok, snelheid of score. De lessen komen uit GNU Typist
// (Debian-pakket gtypist; de Nederlandse juniorlessen van Marc Heyvaert voor KTouch), daarna
// oefenen met eigen zinnen. Een toetsenbord in beeld wijst de volgende toets en de vinger aan.

type typLes struct {
	Titel      string
	Oefeningen [][]string // per oefening de regels
}

// typBestanden: welke lessen per taal (te vervangen in tests).
var typMap = "/usr/share/gtypist"
var typBestanden = map[string]string{"nl": "ktnl_junior.typ", "en": "kten.typ"}

var lesNummer = regexp.MustCompile(`(?i)^\s*lesson\s+(\d+):\s*`)

// leesTypLessen leest een gtypist-bestand: B: begint een les, D: en S: een oefening, " :" gaat verder.
func leesTypLessen(pad string) ([]typLes, error) {
	f, err := os.Open(pad)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lessen []typLes
	var oef []string
	klaarOef := func() {
		if len(oef) > 0 && len(lessen) > 0 {
			l := &lessen[len(lessen)-1]
			l.Oefeningen = append(l.Oefeningen, oef)
		}
		oef = nil
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		r := sc.Text()
		switch {
		case strings.HasPrefix(r, "B:"):
			klaarOef()
			titel := lesNummer.ReplaceAllString(strings.TrimSpace(r[2:]), "")
			lessen = append(lessen, typLes{Titel: titel})
		case strings.HasPrefix(r, "D:") || strings.HasPrefix(r, "S:"):
			klaarOef()
			if t := strings.TrimRight(r[2:], " "); t != "" {
				oef = append(oef, t)
			}
		case strings.HasPrefix(r, " :") && oef != nil:
			if t := strings.TrimRight(r[2:], " "); t != "" {
				oef = append(oef, t)
			}
		default:
			if !strings.HasPrefix(r, " :") {
				klaarOef()
			}
		}
	}
	klaarOef()
	var uit []typLes
	for _, l := range lessen {
		if len(l.Oefeningen) > 0 {
			uit = append(uit, l)
		}
	}
	return uit, sc.Err()
}

var eigenZinnenNL = []string{
	"De egel slaapt de hele winter in een hoopje bladeren.",
	"Een octopus heeft drie harten en blauw bloed.",
	"Op zolder staat een doos vol oude stripboeken.",
	"De robot maakt elke ochtend pannenkoeken voor iedereen.",
	"Vanavond zie ik de maan boven de bomen staan.",
	"Mijn fiets heeft een bel, een lampje en een rode bagagedrager.",
	"In het bos hoor je een specht tegen een boom tikken.",
	"De ijsvogel duikt pijlsnel in het water.",
	"Een ruimteschip vliegt langs de ringen van Saturnus.",
	"Op school bouwen we een toren van twintig blokken.",
}

var eigenZinnenEN = []string{
	"The hedgehog sleeps all winter in a pile of leaves.",
	"An octopus has three hearts and blue blood.",
	"In the attic there is a box full of old comics.",
	"The robot makes pancakes for everyone every morning.",
	"Tonight I can see the moon above the trees.",
	"My bike has a bell, a light and a red rack.",
	"In the forest you can hear a woodpecker tapping a tree.",
	"The kingfisher dives into the water like an arrow.",
	"A spaceship flies past the rings of Saturn.",
	"At school we build a tower of twenty blocks.",
}

// eigenZinnen: zinnen uit zijn eigen verhalen, aangevuld met rustige ingebouwde zinnen.
func eigenZinnen(taal string, aantal int) []string {
	var zinnen []string
	for _, v := range eerdereVerhalen(20) {
		data, err := os.ReadFile(v.pad)
		if err != nil {
			continue
		}
		for _, z := range regexp.MustCompile(`[^.!?]+[.!?]`).FindAllString(string(data), -1) {
			z = strings.Join(strings.Fields(z), " ")
			if n := len([]rune(z)); n >= 20 && n <= 70 {
				zinnen = append(zinnen, z)
			}
		}
	}
	ingebouwd := eigenZinnenEN
	if taal == "nl" {
		ingebouwd = eigenZinnenNL
	}
	rand.Shuffle(len(zinnen), func(i, j int) { zinnen[i], zinnen[j] = zinnen[j], zinnen[i] })
	if len(zinnen) > aantal-2 {
		zinnen = zinnen[:aantal-2]
	}
	for _, i := range rand.Perm(len(ingebouwd)) {
		if len(zinnen) >= aantal {
			break
		}
		zinnen = append(zinnen, ingebouwd[i])
	}
	return zinnen
}

/* ---------------- voortgang ---------------- */

type typVoortgang struct {
	Klaar map[string]bool `json:"klaar"` // "nl-3" = les 3 in het Nederlands af
}

func typPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "typen.json") }

func laadTypVoortgang() typVoortgang {
	v := typVoortgang{Klaar: map[string]bool{}}
	if data, err := os.ReadFile(typPad()); err == nil {
		_ = json.Unmarshal(data, &v)
	}
	if v.Klaar == nil {
		v.Klaar = map[string]bool{}
	}
	return v
}

func (v typVoortgang) bewaar() {
	_ = os.MkdirAll(filepath.Dir(typPad()), 0o755)
	data, _ := json.Marshal(v)
	_ = os.WriteFile(typPad(), data, 0o644)
}

/* ---------------- staat en toetsen ---------------- */

type typStaat struct {
	lessen   []typLes
	fout     string
	sel      int // in de lijst: 0..len(lessen)-1 lessen, len(lessen) = eigen zinnen
	bezig    bool
	les      int // -1: eigen zinnen
	oef, rij int
	regels   []string
	getypt   []rune
	lastig   map[rune]int
	klaar    bool
}

func (m *model) typOpen() {
	t := m.sh.ty
	taal := m.inst.Taal
	if taal != "nl" {
		taal = "en"
	}
	lessen, err := leesTypLessen(filepath.Join(typMap, typBestanden[taal]))
	t.lessen, t.fout, t.bezig = lessen, "", false
	if err != nil {
		t.fout = m.t("type.notinstalled")
	}
}

func (m *model) typBegin(les int) {
	t := m.sh.ty
	t.les, t.oef, t.rij, t.getypt, t.klaar, t.bezig = les, 0, 0, nil, false, true
	t.lastig = map[rune]int{}
	if les < 0 {
		t.regels = eigenZinnen(m.inst.Taal, 6)
	} else {
		t.regels = t.lessen[les].Oefeningen[0]
	}
}

func (m *model) toetsTypen(msg tea.KeyMsg) (tea.Cmd, bool) {
	t := m.sh.ty
	k := msg.String()
	if !t.bezig {
		switch k {
		case "up", "k":
			t.sel = max(t.sel-1, 0)
		case "down", "j":
			t.sel = min(t.sel+1, len(t.lessen))
		case "enter":
			if t.sel == len(t.lessen) {
				m.typBegin(-1)
			} else if len(t.lessen) > 0 {
				m.typBegin(t.sel)
			}
		case "esc":
			m.sh.scherm = schrijfMenu
		default:
			return nil, false
		}
		return nil, true
	}
	if t.klaar {
		switch k {
		case "enter", " ":
			// volgende oefening, of de volgende les
			if t.les >= 0 && t.oef+1 < len(t.lessen[t.les].Oefeningen) {
				t.oef++
				t.regels, t.rij, t.getypt, t.klaar = t.lessen[t.les].Oefeningen[t.oef], 0, nil, false
				t.lastig = map[rune]int{}
			} else if t.les >= 0 && t.les+1 < len(t.lessen) {
				m.typBegin(t.les + 1)
				t.sel = t.les
			} else {
				m.typBegin(t.les)
			}
		case "esc":
			t.bezig = false
		}
		return nil, true
	}
	doel := []rune(t.regels[t.rij])
	switch {
	case k == "esc":
		t.bezig = false
	case k == "backspace":
		if len(t.getypt) > 0 {
			t.getypt = t.getypt[:len(t.getypt)-1]
		}
	case k == "enter":
		if len(t.getypt) >= len(doel) {
			m.typVolgendeRegel()
		}
	case msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace:
		for _, r := range msg.Runes {
			if len(t.getypt) >= len(doel) {
				break
			}
			if r != doel[len(t.getypt)] {
				t.lastig[doel[len(t.getypt)]]++
			}
			t.getypt = append(t.getypt, r)
		}
		if len(t.getypt) == len(doel) && string(t.getypt) == string(doel) {
			m.typVolgendeRegel() // klopt helemaal: meteen door
		}
	}
	return nil, true
}

func (m *model) typVolgendeRegel() {
	t := m.sh.ty
	t.rij++
	t.getypt = nil
	if t.rij >= len(t.regels) {
		t.klaar = true
		if t.les >= 0 && t.oef == len(t.lessen[t.les].Oefeningen)-1 {
			v := laadTypVoortgang()
			v.Klaar[fmt.Sprintf("%s-%d", m.inst.Taal, t.les)] = true
			v.bewaar()
		}
	}
}

/* ---------------- weergave ---------------- */

var toetsRijen = []string{"1234567890", "qwertyuiop", "asdfghjkl;", "zxcvbnm,./"}

// vingers: welke vinger hoort bij welke toets (QWERTY, tienvingersysteem).
func vinger(r rune) string {
	r = []rune(strings.ToLower(string(r)))[0]
	switch {
	case r == ' ':
		return "thumb"
	case strings.ContainsRune("1qaz`", r):
		return "lpinky"
	case strings.ContainsRune("2wsx", r):
		return "lring"
	case strings.ContainsRune("3edc", r):
		return "lmiddle"
	case strings.ContainsRune("45rtfgvb", r):
		return "lindex"
	case strings.ContainsRune("67yuhjnm", r):
		return "rindex"
	case strings.ContainsRune("8ik,", r):
		return "rmiddle"
	case strings.ContainsRune("9ol.", r):
		return "rring"
	}
	return "rpinky"
}

func (m model) viewTypen(p palet, breed int) string {
	t := m.sh.ty
	dim := lipgloss.NewStyle().Foreground(p.dim)
	if t.fout != "" {
		return lipgloss.NewStyle().Foreground(p.geel).Width(breed - 6).Render(t.fout)
	}
	if !t.bezig {
		v := laadTypVoortgang()
		var r []string
		hoogte := max(m.hoog-12, 5)
		begin := min(max(t.sel-hoogte/2, 0), max(len(t.lessen)+1-hoogte, 0))
		for i := begin; i <= len(t.lessen) && i < begin+hoogte; i++ {
			naam := m.t("type.ownsentences")
			if i < len(t.lessen) {
				vink := "  "
				if v.Klaar[fmt.Sprintf("%s-%d", m.inst.Taal, i)] {
					vink = "✓ "
				}
				naam = vink + m.t("type.lesson", i+1) + ": " + t.lessen[i].Titel
			}
			if i == t.sel {
				r = append(r, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸ "+naam+" "))
			} else {
				r = append(r, "  "+naam)
			}
		}
		return strings.Join(r, "\n") + "\n\n" + dim.Render(m.t("type.source"))
	}
	var r []string
	titel := m.t("type.ownsentences")
	if t.les >= 0 {
		titel = m.t("type.lesson", t.les+1) + ": " + t.lessen[t.les].Titel + dim.Render(fmt.Sprintf("  ·  %s %d/%d", m.t("type.exercise"), t.oef+1, len(t.lessen[t.les].Oefeningen)))
	}
	r = append(r, lipgloss.NewStyle().Bold(true).Render(titel), "")
	if t.klaar {
		r = append(r, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(m.t("type.done")))
		if len(t.lastig) > 0 {
			type tl struct {
				r rune
				n int
			}
			var l []tl
			for k, n := range t.lastig {
				l = append(l, tl{k, n})
			}
			sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
			var ts []string
			for i, x := range l {
				if i == 5 {
					break
				}
				naam := string(x.r)
				if x.r == ' ' {
					naam = m.t("type.space")
				}
				ts = append(ts, naam)
			}
			r = append(r, "", m.t("type.tricky")+" "+lipgloss.NewStyle().Foreground(p.geel).Render(strings.Join(ts, "  ")))
		} else {
			r = append(r, "", m.t("type.noerrors"))
		}
		return strings.Join(r, "\n")
	}
	// de regels: wat al getypt is (goed groen, fout rood), de rest grijs
	for i, regel := range t.regels {
		doel := []rune(regel)
		if i < t.rij {
			r = append(r, dim.Render("  "+regel))
			continue
		}
		if i > t.rij {
			r = append(r, dim.Render("  "+regel))
			continue
		}
		var b strings.Builder
		for j, c := range doel {
			switch {
			case j < len(t.getypt) && t.getypt[j] == c:
				b.WriteString(lipgloss.NewStyle().Foreground(p.groen).Render(string(c)))
			case j < len(t.getypt):
				teken := string(c)
				if c == ' ' {
					teken = "·"
				}
				b.WriteString(lipgloss.NewStyle().Foreground(p.bg).Background(p.rood).Render(teken))
			case j == len(t.getypt):
				b.WriteString(lipgloss.NewStyle().Foreground(p.bg).Background(p.acc).Render(string(c)))
			default:
				b.WriteString(lipgloss.NewStyle().Foreground(p.fg).Render(string(c)))
			}
		}
		r = append(r, "▸ "+b.String())
	}
	// toetsenbord met de volgende toets
	doel := []rune(t.regels[t.rij])
	var volgende rune = '\n'
	if len(t.getypt) < len(doel) {
		volgende = doel[len(t.getypt)]
	}
	r = append(r, "")
	for i, rij := range toetsRijen {
		var b strings.Builder
		b.WriteString(strings.Repeat(" ", i+2))
		for _, c := range rij {
			stijl := lipgloss.NewStyle().Foreground(p.dim)
			if strings.ContainsRune("fj", c) {
				stijl = stijl.Foreground(p.fg).Underline(true) // de voelbare streepjes
			}
			if c == []rune(strings.ToLower(string(volgende)))[0] {
				stijl = lipgloss.NewStyle().Background(p.acc).Foreground(p.bg).Bold(true)
			}
			b.WriteString(stijl.Render(" " + string(c) + " "))
		}
		r = append(r, b.String())
	}
	spatie := dim.Render(strings.Repeat(" ", 10) + "[          " + m.t("type.space") + "          ]")
	if volgende == ' ' {
		spatie = strings.Repeat(" ", 10) + lipgloss.NewStyle().Background(p.acc).Foreground(p.bg).Bold(true).Render("[          "+m.t("type.space")+"          ]")
	}
	r = append(r, spatie, "")
	switch {
	case volgende == '\n':
		r = append(r, dim.Render(m.t("type.pressenter")))
	case volgende >= 'A' && volgende <= 'Z':
		r = append(r, dim.Render(m.t("type.finger."+vinger(volgende))+" · "+m.t("type.shift")))
	default:
		r = append(r, dim.Render(m.t("type.finger."+vinger(volgende))))
	}
	return strings.Join(r, "\n")
}
