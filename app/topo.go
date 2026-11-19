package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Topografie: Europa (landen en hoofdsteden, groep 7) en Nederland (provincies en hun
// hoofdsteden, steden, rivieren, wateren en Waddeneilanden, groep 6 en 7). Een kaart getekend
// met braillepuntjes waarop één ding oplicht: een land of provincie ingekleurd, een stad als
// stip, een rivier als lijn. Eerst rondkijken, dan overhoren; de leermotor laat terugkomen wat
// lastig is.
//
// De kaarten komen uit Natural Earth (publiek domein), vereenvoudigd met tools/maak_topo.py.

//go:embed topo/*.json
var topoBestanden embed.FS

type topoStad struct {
	NL   string     `json:"nl"`
	EN   string     `json:"en"`
	Ook  []string   `json:"ook"`
	Punt [2]float64 `json:"punt"`
}

type topoDing struct {
	Code  string         `json:"code"`
	Groep string         `json:"groep"` // land, provincie, stad, water
	NL    string         `json:"nl"`
	EN    string         `json:"en"`
	Ook   []string       `json:"ook"`
	Vlak  [][][2]float64 `json:"vlak"`
	Punt  *[2]float64    `json:"punt"`
	Lijn  [][][2]float64 `json:"lijn"`
	Stad  *topoStad      `json:"stad"`
}

type topoKaart struct {
	Kader       [4]float64     `json:"kader"`
	Achtergrond [][][2]float64 `json:"achtergrond"`
	Lijnen      [][][2]float64 `json:"lijnen"`
	Items       []topoDing     `json:"items"`
}

var (
	topoMu      sync.Mutex
	topoKaarten = map[string]*topoKaart{}
)

func topoKaartVan(naam string) *topoKaart {
	topoMu.Lock()
	defer topoMu.Unlock()
	if k, ok := topoKaarten[naam]; ok {
		return k
	}
	k := &topoKaart{}
	if data, err := topoBestanden.ReadFile("topo/" + naam + ".json"); err == nil {
		_ = json.Unmarshal(data, k)
	}
	topoKaarten[naam] = k
	return k
}

// topoOefening: wat je oefent, op welke kaart, en of je de naam of de hoofdstad noemt.
type topoOefening struct {
	sleutel, kaart, groep string
	vraag                 string // "kijk", "naam" of "stad"
}

var topoOefeningen = []topoOefening{
	{"topo.eu.look", "europa", "land", "kijk"},
	{"topo.eu.countries", "europa", "land", "naam"},
	{"topo.eu.capitals", "europa", "land", "stad"},
	{"topo.nl.look", "nederland", "", "kijk"},
	{"topo.nl.provinces", "nederland", "provincie", "naam"},
	{"topo.nl.capitals", "nederland", "provincie", "stad"},
	{"topo.nl.cities", "nederland", "stad", "naam"},
	{"topo.nl.water", "nederland", "water", "naam"},
}

func (o topoOefening) set() string { return "topo:" + o.kaart + ":" + o.groep + ":" + o.vraag }

// dingen: alles wat bij deze oefening hoort ("" is alles op de kaart).
func (o topoOefening) dingen() []*topoDing {
	k := topoKaartVan(o.kaart)
	var uit []*topoDing
	for i := range k.Items {
		if o.groep == "" || k.Items[i].Groep == o.groep {
			uit = append(uit, &k.Items[i])
		}
	}
	return uit
}

func (d *topoDing) naam(taal string) string {
	if taal == "nl" {
		return d.NL
	}
	return d.EN
}

func (s *topoStad) naam(taal string) string {
	if taal == "nl" {
		return s.NL
	}
	return s.EN
}

// gewoon: zonder hoofdletters, accenten, streepjes en spaties, zodat "oekraine" en
// "Oekraïne" allebei goed zijn.
func gewoon(s string) string {
	return strings.NewReplacer("-", "", " ", "", "'", "", ".", "").Replace(zoekVorm(strings.TrimSpace(s)))
}

func topoGoed(antwoord string, opties ...string) bool {
	a := gewoon(antwoord)
	if a == "" {
		return false
	}
	for _, o := range opties {
		if gewoon(o) == a {
			return true
		}
	}
	return false
}

/* ---------------- tekenen ---------------- */

// topoMaat: hoeveel tekens breed een kaart van h regels hoog is (zodat hij niet uitgerekt wordt).
func topoMaat(kaart string, h int) int {
	k := topoKaartVan(kaart)
	cosL := math.Cos((k.Kader[1] + k.Kader[3]) / 2 * math.Pi / 180)
	verhouding := (k.Kader[2] - k.Kader[0]) * cosL / (k.Kader[3] - k.Kader[1])
	return int(math.Ceil(float64(h*4) * verhouding / 2))
}

var (
	topoCacheMu sync.Mutex
	topoCache   = map[string]string{}
)

// topoTekst tekent een kaart van b × h tekens met één ding dat oplicht (code, of niets), en
// bij stad ook een stip op de hoofdstad. Onthouden per tekening: tekenen is werk voor een netbook.
func topoTekst(kaart, code string, stad bool, b, h int, p palet) string {
	sleutel := fmt.Sprintf("%s|%s|%v|%d|%d|%s|%v", kaart, code, stad, b, h, p.acc, alleenLetters())
	topoCacheMu.Lock()
	if s, ok := topoCache[sleutel]; ok {
		topoCacheMu.Unlock()
		return s
	}
	topoCacheMu.Unlock()

	k := topoKaartVan(kaart)
	lonMin, latMin, lonMax, latMax := k.Kader[0], k.Kader[1], k.Kader[2], k.Kader[3]
	cosL := math.Cos((latMin + latMax) / 2 * math.Pi / 180) // een graad lengte is korter dan een graad breedte
	schaal := math.Min(float64(b*2)/((lonMax-lonMin)*cosL), float64(h*4)/(latMax-latMin))
	naar := func(pt [2]float64) (float64, float64) {
		return (pt[0] - lonMin) * cosL * schaal, (latMax - pt[1]) * schaal
	}
	rand, water, vul := nieuwBrailleVlak(b, h), nieuwBrailleVlak(b, h), nieuwBrailleVlak(b, h)
	omtrek := func(v *brailleVlak, r [][2]float64) {
		for j := 1; j < len(r); j++ {
			x0, y0 := naar(r[j-1])
			x1, y1 := naar(r[j])
			v.lijn(x0, y0, x1, y1)
		}
	}
	for _, r := range k.Achtergrond {
		omtrek(rand, r)
	}
	var doel *topoDing
	for i := range k.Items {
		d := &k.Items[i]
		if d.Code == code {
			doel = d
		}
		for _, r := range d.Vlak {
			omtrek(rand, r)
		}
	}
	for _, l := range k.Lijnen { // rivieren, lichtjes
		omtrek(water, l)
	}
	puntX, puntY := -1, -1
	if doel != nil {
		// een vlak inkleuren: elk puntje waarvan het midden erin ligt
		var ringen [][][2]float64
		for _, r := range doel.Vlak {
			var rr [][2]float64
			for _, pt := range r {
				x, y := naar(pt)
				rr = append(rr, [2]float64{x, y})
			}
			ringen = append(ringen, rr)
		}
		gevuld := 0
		if len(ringen) > 0 {
			for y := 0; y < h*4; y++ {
				for x := 0; x < b*2; x++ {
					if binnen(float64(x)+0.5, float64(y)+0.5, ringen) {
						vul.zet(x, y)
						gevuld++
					}
				}
			}
		}
		if len(doel.Vlak) > 0 && gevuld < 4 { // heel klein (Luxemburg, Schiermonnikoog): omtrek en een stip
			var sx, sy, n float64
			for _, r := range doel.Vlak {
				omtrek(vul, r)
				for _, pt := range r {
					sx, sy, n = sx+pt[0], sy+pt[1], n+1
				}
			}
			x, y := naar([2]float64{sx / n, sy / n})
			puntX, puntY = int(x/2), int(y/4)
		}
		for _, l := range doel.Lijn { // een rivier: dik
			for _, d := range [][2]float64{{0, 0}, {0.7, 0}, {0, 0.7}} {
				for j := 1; j < len(l); j++ {
					x0, y0 := naar(l[j-1])
					x1, y1 := naar(l[j])
					vul.lijn(x0+d[0], y0+d[1], x1+d[0], y1+d[1])
				}
			}
		}
		if doel.Punt != nil {
			x, y := naar(*doel.Punt)
			puntX, puntY = int(x/2), int(y/4)
		}
	}
	stadX, stadY := -1, -1
	if stad && doel != nil && doel.Stad != nil {
		x, y := naar(doel.Stad.Punt)
		stadX, stadY = int(x/2), int(y/4)
	}
	randT, waterT, vulT := rand.tekens(), water.tekens(), vul.tekens()
	stijlRand := lipgloss.NewStyle().Foreground(p.lijn)
	stijlWater := lipgloss.NewStyle().Foreground(p.dim)
	stijlDoel := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	stijlStad := lipgloss.NewStyle().Foreground(p.geel).Bold(true)
	var regels []string
	for y := 0; y < h; y++ {
		var s strings.Builder
		for x := 0; x < b; x++ {
			switch {
			case x == puntX && y == puntY:
				s.WriteString(stijlDoel.Render("●"))
			case x == stadX && y == stadY:
				s.WriteString(stijlStad.Render("●"))
			case vulT[y][x] != ' ':
				s.WriteString(stijlDoel.Render(string(vulT[y][x])))
			case waterT[y][x] != ' ':
				s.WriteString(stijlWater.Render(string(waterT[y][x])))
			case randT[y][x] != ' ':
				s.WriteString(stijlRand.Render(string(randT[y][x])))
			default:
				s.WriteString(" ")
			}
		}
		regels = append(regels, s.String())
	}
	uit := strings.Join(regels, "\n")
	topoCacheMu.Lock()
	topoCache[sleutel] = uit
	topoCacheMu.Unlock()
	return uit
}

// binnen: ligt het punt in een van de ringen? (even-oneven-regel)
func binnen(x, y float64, ringen [][][2]float64) bool {
	in := false
	for _, r := range ringen {
		for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
			if (r[i][1] > y) != (r[j][1] > y) && x < (r[j][0]-r[i][0])*(y-r[i][1])/(r[j][1]-r[i][1])+r[i][0] {
				in = !in
			}
		}
	}
	return in
}

/* ---------------- oefenen ---------------- */

type topoStaat struct {
	bezig      bool // false: kiezen wat je oefent
	keuze      int  // in topoOefeningen
	kijk       int  // bij rondkijken: welk ding
	ronde      []*topoDing
	nr         int
	invoer     textinput.Model
	uitslag    string
	goed       bool
	klaar      bool // nagekeken: Enter gaat verder
	gehad      map[string]bool
	lastig     []string
	rondeKlaar bool
}

func nieuweTopo() *topoStaat {
	t := &topoStaat{}
	t.invoer = textinput.New()
	t.invoer.CharLimit = 40
	t.invoer.Prompt = "➜ "
	return t
}

func (t *topoStaat) oefening() topoOefening { return topoOefeningen[t.keuze] }

func (m *model) topoStart() tea.Cmd {
	t := m.lr.topo
	t.bezig, t.kijk = true, 0
	o := t.oefening()
	if o.vraag == "kijk" {
		return nil
	}
	per := map[string]*topoDing{}
	var codes []string
	for _, d := range o.dingen() {
		per[d.Code] = d
		codes = append(codes, d.Code)
	}
	t.ronde = nil
	for _, c := range laadLeren().kies(o.set(), codes, 10, time.Now()) {
		t.ronde = append(t.ronde, per[c])
	}
	t.nr, t.gehad, t.lastig, t.rondeKlaar = 0, map[string]bool{}, nil, false
	t.uitslag, t.klaar = "", false
	t.invoer.SetValue("")
	return t.invoer.Focus()
}

func (m *model) topoNakijken() {
	t := m.lr.topo
	o := t.oefening()
	d := t.ronde[t.nr]
	taal := m.inst.Taal
	var goed bool
	var juist string
	if o.vraag == "stad" {
		goed = topoGoed(t.invoer.Value(), append([]string{d.Stad.NL, d.Stad.EN}, d.Stad.Ook...)...)
		juist = d.Stad.naam(taal)
	} else {
		goed = topoGoed(t.invoer.Value(), append([]string{d.NL, d.EN}, d.Ook...)...)
		juist = d.naam(taal)
	}
	if !t.gehad[d.Code] {
		t.gehad[d.Code] = true
		ls := laadLeren()
		ls.zet(o.set(), d.Code, goed, time.Now())
		ls.bewaar()
	}
	t.goed, t.klaar = goed, true
	if goed {
		t.uitslag = m.t("topo.right", juist)
		return
	}
	t.uitslag = m.t("topo.wrong", juist)
	t.lastig = append(t.lastig, juist)
	if len(t.ronde) < 14 {
		t.ronde = append(t.ronde, d)
	}
}

func (m *model) toetsTopo(msg tea.KeyMsg) (tea.Cmd, bool) {
	t := m.lr.topo
	k := msg.String()
	if !t.bezig {
		switch k {
		case "up", "k":
			t.keuze = max(t.keuze-1, 0)
		case "down", "j":
			t.keuze = min(t.keuze+1, len(topoOefeningen)-1)
		case "enter":
			return m.topoStart(), true
		case "esc":
			m.lr.scherm = lerenMenu
		default:
			return nil, false
		}
		return nil, true
	}
	o := t.oefening()
	if o.vraag == "kijk" {
		n := len(o.dingen())
		switch k {
		case "right", "down", "l", "j", " ":
			t.kijk = (t.kijk + 1) % n
		case "left", "up", "h", "k":
			t.kijk = (t.kijk + n - 1) % n
		case "esc":
			t.bezig = false
		default:
			return nil, false
		}
		return nil, true
	}
	if t.rondeKlaar {
		switch k {
		case "enter", " ":
			return m.topoStart(), true
		case "esc":
			t.invoer.Blur()
			t.bezig = false
		}
		return nil, true
	}
	switch k {
	case "esc":
		t.invoer.Blur()
		t.bezig = false
		return nil, true
	case "enter":
		switch {
		case t.klaar:
			t.nr++
			t.klaar, t.uitslag = false, ""
			t.invoer.SetValue("")
			if t.nr >= len(t.ronde) {
				t.rondeKlaar = true
			}
		case strings.TrimSpace(t.invoer.Value()) != "":
			m.topoNakijken()
		}
		return nil, true
	}
	if t.klaar {
		return nil, true
	}
	var cmd tea.Cmd
	t.invoer, cmd = t.invoer.Update(msg)
	return cmd, true
}

// topoVraag: de vraag bij een ding, passend bij wat het is.
func (m model) topoVraag(o topoOefening, d *topoDing) string {
	taal := m.inst.Taal
	if o.vraag == "stad" {
		return m.t("topo.whichcapital", d.naam(taal))
	}
	switch {
	case d.Groep == "land":
		return m.t("topo.whichcountry")
	case d.Groep == "provincie":
		return m.t("topo.whichprovince")
	case d.Groep == "stad":
		return m.t("topo.whichcity")
	case len(d.Lijn) > 0:
		return m.t("topo.whichriver")
	case strings.HasPrefix(d.Code, "eiland:"):
		return m.t("topo.whichisland")
	}
	return m.t("topo.whichwater")
}

func (m model) viewTopo(p palet, kop string, breed int) string {
	t := m.lr.topo
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	taal := m.inst.Taal
	if !t.bezig {
		l := laadLeren()
		var regels []string
		vorige := ""
		for i, o := range topoOefeningen {
			if o.kaart != vorige {
				if vorige != "" {
					regels = append(regels, "")
				}
				regels = append(regels, lipgloss.NewStyle().Bold(true).Render(m.t("topo.map."+o.kaart)))
				vorige = o.kaart
			}
			naam := fmt.Sprintf("%-26s", m.t(o.sleutel))
			extra := dim.Render("  " + m.t("topo.look.d"))
			if o.vraag != "kijk" {
				var codes []string
				for _, d := range o.dingen() {
					codes = append(codes, d.Code)
				}
				extra = dim.Render(fmt.Sprintf("  %d/%d", l.kent(o.set(), codes), len(codes)))
			}
			if i == t.keuze {
				regels = append(regels, acc.Render("▸ "+naam)+extra)
			} else {
				regels = append(regels, "  "+naam+extra)
			}
		}
		regels = append(regels, "", dim.Render(m.t("dictee.legend")))
		return kop + "\n\n" + paneel(p, m.t("learn.topo"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.pickkeys"))
	}
	o := t.oefening()
	// kaart links, tekst rechts
	h := max(m.hoog-11, 8)
	b := min(topoMaat(o.kaart, h), breed-36)
	breedRechts := max(breed-b-10, 20)
	tekst := lipgloss.NewStyle().Width(breedRechts)
	var ding *topoDing
	var rechts []string
	switch {
	case o.vraag == "kijk":
		alle := o.dingen()
		ding = alle[t.kijk%len(alle)]
		rechts = append(rechts, acc.Render(ding.naam(taal)))
		if ding.Stad != nil {
			rechts = append(rechts, "", m.t("topo.capital")+" "+lipgloss.NewStyle().Foreground(p.geel).Render(ding.Stad.naam(taal)))
		}
		rechts = append(rechts, "", dim.Render(fmt.Sprintf("%d/%d", t.kijk+1, len(alle))))
	case t.rondeKlaar:
		rechts = append(rechts, acc.Render(m.t("dictee.done")), "")
		if len(t.lastig) == 0 {
			rechts = append(rechts, m.t("dictee.allright"))
		} else {
			rechts = append(rechts, m.t("dictee.tricky"), tekst.Render(strings.Join(t.lastig, ", ")))
		}
	default:
		ding = t.ronde[t.nr]
		rechts = append(rechts, dim.Render(m.t("dictee.count", t.nr+1, len(t.ronde))), "",
			lipgloss.NewStyle().Bold(true).Width(breedRechts).Render(m.topoVraag(o, ding)), "", t.invoer.View(), "")
		if t.uitslag != "" {
			kleur := p.groen
			if !t.goed {
				kleur = p.geel
			}
			rechts = append(rechts, tekst.Foreground(kleur).Render(t.uitslag))
		}
	}
	code := ""
	if ding != nil {
		code = ding.Code
	}
	kaart := topoTekst(o.kaart, code, o.vraag != "naam", b, h, p)
	inhoud := lipgloss.JoinHorizontal(lipgloss.Top, kaart, "   ", strings.Join(rechts, "\n"))
	toetsen := m.t("topo.keys")
	switch {
	case o.vraag == "kijk":
		toetsen = m.t("topo.lookkeys")
	case t.rondeKlaar:
		toetsen = m.t("dictee.donekeys")
	case t.klaar:
		toetsen = m.t("topo.nextkeys")
	}
	return kop + "\n\n" + paneel(p, m.t("learn.topo"), inhoud, breed, true) + "\n" + dim.Render(toetsen)
}
