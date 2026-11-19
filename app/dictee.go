package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dictee: je hoort een woord in een zinnetje en typt het. Fout? Dan zie je welke letters anders
// zijn, met de regel erbij, en typ je het goed over. Wat lastig is komt later terug (de
// leermotor). Geen punten, geen klok.
//
// Uit het hoofd opschrijven blijft beter hangen dan overschrijven; meteen zien wat er anders
// is en waarom, en het daarna goed typen, helpt om het de volgende keer wel te weten.

type dicteeWoord struct {
	woord string // met | tussen de klankgroepen waar dat bij de regel hoort (bo|men)
	zin   string // ___ is de plek van het woord
}

func (w dicteeWoord) schoon() string { return strings.ReplaceAll(w.woord, "|", "") }

type dicteeSoort struct {
	sleutel string
	woorden []dicteeWoord
}

// De categorieën zoals op school: elke soort fout heeft een eigen regel.
var dicteeSoorten = []dicteeSoort{
	{"klank", []dicteeWoord{
		{"bo|men", "In het bos staan hoge ___."},
		{"boom", "In de tuin staat een grote ___."},
		{"ra|men", "Doe de ___ maar open, het is warm."},
		{"raam", "Ik kijk door het ___ naar buiten."},
		{"ste|nen", "Hij gooit ___ in het water."},
		{"steen", "Er zit een ___ in mijn schoen."},
		{"mu|ren", "De ___ van mijn kamer zijn blauw."},
		{"muur", "De kat zit boven op de ___."},
		{"bo|ten", "In de haven liggen veel ___."},
		{"boot", "We gaan met de ___ naar het eiland."},
		{"gro|te", "Dat is een ___ hond."},
		{"groot", "Mijn broer is al heel ___."},
		{"ro|de", "Ik draag vandaag mijn ___ jas."},
		{"rood", "Het stoplicht staat op ___."},
		{"be|nen", "Een spin heeft acht ___."},
		{"been", "Hij heeft zijn ___ gebroken."},
		{"vuur", "Het ___ in de open haard is lekker warm."},
		{"zo|mer", "In de ___ gaan we zwemmen."},
		{"spe|len", "Na school gaan we buiten ___."},
		{"ha|len", "Wil jij even brood ___?"},
	}},
	{"bd", []dicteeWoord{
		{"bed", "Om acht uur ga ik naar ___."},
		{"bad", "Ik zit lekker in ___."},
		{"dak", "De kat loopt over het ___."},
		{"bak", "Doe de appels maar in de ___."},
		{"deur", "Doe de ___ dicht, het tocht."},
		{"duim", "Ik heb met de hamer op mijn ___ geslagen."},
		{"buik", "Ik heb pijn in mijn ___."},
		{"dorp", "Opa woont in een klein ___."},
		{"boek", "Ik lees een spannend ___."},
		{"doos", "De schoenen zitten nog in de ___."},
		{"boos", "Waarom ben je zo ___?"},
		{"brood", "Ik eet ___ met kaas."},
		{"dier", "Een olifant is een groot ___."},
		{"bal", "Schop de ___ in het doel."},
		{"baard", "Opa heeft een lange grijze ___."},
		{"draak", "De ___ spuwt vuur."},
	}},
	{"eiij", []dicteeWoord{
		{"trein", "We gaan met de ___ naar oma."},
		{"ijsje", "Mag ik een ___ met aardbei?"},
		{"klein", "Mijn zusje is nog heel ___."},
		{"tijd", "Kom, het is ___ om te gaan."},
		{"reis", "We maken een lange ___ naar Spanje."},
		{"fijn", "Wat een ___ dag!"},
		{"geit", "Op de boerderij woont een ___."},
		{"plein", "We spelen op het ___ voor de school."},
		{"blij", "Ik ben ___ met mijn cadeau."},
		{"ei", "Ik eet een gekookt ___."},
		{"lijn", "Teken een rechte ___."},
		{"eiland", "Op het ___ wonen maar weinig mensen."},
		{"pijl", "De ___ wijst naar links."},
		{"wij", "Morgen gaan ___ naar de dierentuin."},
	}},
	{"auou", []dicteeWoord{
		{"koud", "Het is ___ buiten, doe je jas aan."},
		{"blauw", "De lucht is ___."},
		{"oud", "Mijn opa is al heel ___."},
		{"saus", "Wil je ___ op je patat?"},
		{"hout", "De tafel is van ___."},
		{"auto", "Mama rijdt in de ___."},
		{"goud", "De ring is van ___."},
		{"pauze", "In de ___ spelen we buiten."},
		{"vrouw", "Die ___ is onze nieuwe juf."},
		{"touw", "We springen over het ___."},
		{"gauw", "Kom maar ___ terug!"},
		{"fout", "Dat antwoord is ___."},
		{"pauw", "De ___ laat zijn mooie veren zien."},
	}},
	{"dt", []dicteeWoord{
		{"hond", "Onze ___ heet Max."},
		{"hand", "Geef me je ___."},
		{"paard", "Ik rijd op een ___."},
		{"kat", "De ___ ligt in de zon."},
		{"tand", "Er zit een gaatje in mijn ___."},
		{"wind", "De ___ waait hard vandaag."},
		{"kind", "Ieder ___ krijgt een snoepje."},
		{"land", "Nederland is een klein ___."},
		{"hoed", "De tovenaar heeft een hoge ___."},
		{"rond", "De aarde is ___."},
		{"wit", "Sneeuw is ___."},
		{"bed", "Mijn ___ staat onder het raam."},
		{"brood", "Wil je nog een stukje ___?"},
		{"boot", "De ___ vaart naar de overkant."},
	}},
	{"iglijk", []dicteeWoord{
		{"lastig", "Deze som is best ___."},
		{"gezellig", "Het is altijd ___ bij oma."},
		{"vrolijk", "De clown is heel ___."},
		{"makkelijk", "Deze puzzel is ___."},
		{"moeilijk", "Breuken vind ik ___."},
		{"handig", "Een zakmes is heel ___."},
		{"eerlijk", "Je moet altijd ___ zijn."},
		{"gevaarlijk", "Over het spoor lopen is ___."},
		{"bezig", "Ik ben nog met mijn huiswerk ___."},
		{"duidelijk", "Is de uitleg ___?"},
		{"vriendelijk", "De buurvrouw is erg ___."},
		{"nodig", "Ik heb een pen ___."},
		{"zonnig", "Het is vandaag ___ weer."},
	}},
}

// dicteeAlles: geen categorie, alles door elkaar.
const dicteeAlles = "alles"

func dicteeSet(soort string) string { return "dictee:" + soort }

func dicteeWoorden(soort string) []dicteeWoord {
	var uit []dicteeWoord
	for _, s := range dicteeSoorten {
		if soort == dicteeAlles || s.sleutel == soort {
			uit = append(uit, s.woorden...)
		}
	}
	return uit
}

// soortVan: de categorie van een woord (voor de regel bij een fout).
func soortVan(w dicteeWoord) string {
	for _, s := range dicteeSoorten {
		for _, x := range s.woorden {
			if x == w {
				return s.sleutel
			}
		}
	}
	return ""
}

/* ---------------- voorlezen ---------------- */

// spreek leest een tekst voor met espeak-ng (offline, licht genoeg voor een netbook).
// Een variabele zodat de tests hem kunnen vervangen. Geeft false als er geen stem is.
var spreek = func(tekst string) bool { return spreekStem("nl", tekst) }

// spreekStem leest voor met een stem van espeak-ng ("nl", "en-gb").
func spreekStem(stem, tekst string) bool {
	spreekMu.Lock()
	defer spreekMu.Unlock()
	if spreekProc != nil && spreekProc.Process != nil {
		_ = spreekProc.Process.Kill() // de vorige zin niet laten doorpraten
	}
	pad, err := exec.LookPath("espeak-ng")
	if err != nil {
		return false
	}
	cmd := exec.Command(pad, "-v", stem, "-s", "125", "-g", "6", tekst)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return false
	}
	spreekProc = cmd
	go func() { _ = cmd.Wait() }()
	return true
}

var (
	spreekMu   sync.Mutex
	spreekProc *exec.Cmd
)

// voorleesTekst: het woord, het zinnetje, en nog een keer het woord, zoals bij een dictee op school.
func voorleesTekst(w dicteeWoord) string {
	return w.schoon() + ". " + strings.ReplaceAll(w.zin, "___", w.schoon()) + " " + w.schoon() + "."
}

/* ---------------- staat ---------------- */

const (
	dicteeKiezen = iota
	dicteeBezig
	dicteeKlaar
)

type dicteeStaat struct {
	scherm   int
	keuze    int // 0..len(dicteeSoorten): de laatste is "alles door elkaar"
	soort    string
	ronde    []dicteeWoord
	nr       int
	invoer   textinput.Model
	fout     string // wat er getypt werd bij een fout; dan moet het goed overgetypt worden
	gehad    map[string]bool
	lastig   []string
	vorige   string // "✓ bomen" na een goed antwoord
	geenStem bool
}

func nieuwDictee() *dicteeStaat {
	d := &dicteeStaat{}
	d.invoer = textinput.New()
	d.invoer.CharLimit = 30
	d.invoer.Prompt = "✎ "
	return d
}

func (d *dicteeStaat) soortBijKeuze() string {
	if d.keuze < len(dicteeSoorten) {
		return dicteeSoorten[d.keuze].sleutel
	}
	return dicteeAlles
}

func (d *dicteeStaat) huidig() dicteeWoord { return d.ronde[d.nr] }

func (m *model) dicteeStart() tea.Cmd {
	d := m.lr.dictee
	d.soort = d.soortBijKeuze()
	woorden := dicteeWoorden(d.soort)
	// een woord dat in twee categorieën staat (bed, brood, boot) maar één keer
	perSleutel := map[string]dicteeWoord{}
	var sleutels []string
	for _, w := range woorden {
		if _, dubbel := perSleutel[w.schoon()]; !dubbel {
			sleutels = append(sleutels, w.schoon())
			perSleutel[w.schoon()] = w
		}
	}
	d.ronde = nil
	for _, s := range laadLeren().kies(dicteeSet(d.soort), sleutels, 10, time.Now()) {
		d.ronde = append(d.ronde, perSleutel[s])
	}
	d.nr, d.fout, d.vorige, d.lastig, d.gehad = 0, "", "", nil, map[string]bool{}
	d.scherm = dicteeBezig
	d.invoer.SetValue("")
	d.geenStem = !spreek(voorleesTekst(d.huidig()))
	return d.invoer.Focus()
}

// dicteeNakijken kijkt het getypte woord na.
func (m *model) dicteeNakijken() {
	d := m.lr.dictee
	w := d.huidig()
	getypt := strings.TrimSpace(d.invoer.Value())
	if getypt == "" {
		return
	}
	goed := strings.EqualFold(getypt, w.schoon())
	if d.fout != "" { // overtypen na een fout
		if goed {
			d.fout = ""
			m.dicteeVolgende()
		} else {
			d.fout = getypt
			d.invoer.SetValue("")
		}
		return
	}
	if !d.gehad[w.schoon()] { // alleen de eerste keer telt
		d.gehad[w.schoon()] = true
		l := laadLeren()
		l.zet(dicteeSet(d.soort), w.schoon(), goed, time.Now())
		if d.soort != dicteeAlles { // ook in "alles door elkaar" weten we dan hoe het gaat
			l.zet(dicteeSet(dicteeAlles), w.schoon(), goed, time.Now())
		}
		l.bewaar()
	}
	if goed {
		d.vorige = "✓ " + w.schoon()
		m.dicteeVolgende()
		return
	}
	d.fout = getypt
	d.lastig = append(d.lastig, w.schoon())
	if len(d.ronde) < 14 { // later in de ronde nog een keer
		d.ronde = append(d.ronde, w)
	}
	d.invoer.SetValue("")
}

func (m *model) dicteeVolgende() {
	d := m.lr.dictee
	d.invoer.SetValue("")
	d.nr++
	if d.nr >= len(d.ronde) {
		d.scherm = dicteeKlaar
		d.invoer.Blur()
		return
	}
	d.geenStem = !spreek(voorleesTekst(d.huidig()))
}

// dicteeVraagBliep: waarom schrijf je het zo? Bliep legt het uit, binnen zijn gewone grenzen
// (dagelijkse vragen, filters, het weekoverzicht voor ouders).
func (m *model) dicteeVraagBliep() tea.Cmd {
	d := m.lr.dictee
	if d.fout == "" {
		return nil
	}
	if reden := m.maatjeDicht(); reden != "" {
		m.zeg(reden)
		return nil
	}
	vraag := m.t("dictee.askbuddy", d.huidig().schoon(), d.fout)
	m.tab = tabMaatje
	cmd := m.startSessie(modusVraag)
	if m.mj.sessie == nil {
		return cmd
	}
	return tea.Batch(cmd, m.stuur(vraag, true))
}

func (m *model) toetsDictee(msg tea.KeyMsg) (tea.Cmd, bool) {
	d := m.lr.dictee
	k := msg.String()
	switch d.scherm {
	case dicteeKiezen:
		switch k {
		case "up", "k":
			d.keuze = max(d.keuze-1, 0)
		case "down", "j":
			d.keuze = min(d.keuze+1, len(dicteeSoorten))
		case "enter":
			return m.dicteeStart(), true
		case "esc":
			m.lr.scherm = lerenMenu
		default:
			return nil, false
		}
		return nil, true
	case dicteeKlaar:
		switch k {
		case "enter", " ":
			return m.dicteeStart(), true
		case "esc":
			d.scherm = dicteeKiezen
		default:
			return nil, false
		}
		return nil, true
	}
	switch k {
	case "esc":
		d.invoer.Blur()
		d.scherm = dicteeKiezen
		return nil, true
	case "enter":
		m.dicteeNakijken()
		return nil, true
	case "tab":
		d.geenStem = !spreek(voorleesTekst(d.huidig()))
		return nil, true
	case "?":
		return m.dicteeVraagBliep(), true
	}
	var cmd tea.Cmd
	d.invoer, cmd = d.invoer.Update(msg)
	return cmd, true
}

/* ---------------- weergave ---------------- */

// verschil: welke letters in a niet in de langste gemeenschappelijke reeks met b zitten.
func verschil(a, b []rune) []bool {
	n, m := len(a), len(b)
	t := make([][]int, n+1)
	for i := range t {
		t[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				t[i][j] = t[i+1][j+1] + 1
			} else {
				t[i][j] = max(t[i+1][j], t[i][j+1])
			}
		}
	}
	anders := make([]bool, n)
	for i := range anders {
		anders[i] = true
	}
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case a[i] == b[j]:
			anders[i] = false
			i++
			j++
		case t[i+1][j] >= t[i][j+1]:
			i++
		default:
			j++
		}
	}
	return anders
}

// metNadruk laat de letters zien die anders zijn, met een streep eronder.
func metNadruk(woord, ander string, gewoon, nadruk lipgloss.Style) string {
	a, b := []rune(strings.ToLower(woord)), []rune(strings.ToLower(ander))
	anders := verschil(a, b)
	var s strings.Builder
	for i, r := range []rune(woord) {
		if anders[i] {
			s.WriteString(nadruk.Render(string(r)))
		} else {
			s.WriteString(gewoon.Render(string(r)))
		}
	}
	return s.String()
}

func (m model) viewDictee(p palet, kop string, breed int) string {
	d := m.lr.dictee
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	tekst := lipgloss.NewStyle().Width(breed - 6)
	switch d.scherm {
	case dicteeKiezen:
		l := laadLeren()
		var regels []string
		for i := 0; i <= len(dicteeSoorten); i++ {
			soort := dicteeAlles
			if i < len(dicteeSoorten) {
				soort = dicteeSoorten[i].sleutel
			}
			var sleutels []string
			for _, w := range dicteeWoorden(soort) {
				sleutels = append(sleutels, w.schoon())
			}
			kent := l.kent(dicteeSet(soort), sleutels)
			naam := fmt.Sprintf("%-28s", m.t("dictee.cat."+soort))
			stip := dim.Render(fmt.Sprintf("  %d/%d", kent, len(sleutels)))
			if i == d.keuze {
				regels = append(regels, acc.Render("▸ "+naam)+stip)
			} else {
				regels = append(regels, "  "+naam+stip)
			}
		}
		regels = append(regels, "", dim.Render(m.t("dictee.legend")))
		return kop + "\n\n" + paneel(p, m.t("dictee.pick"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.pickkeys"))
	case dicteeKlaar:
		regels := []string{acc.Render(m.t("dictee.done")), ""}
		if len(d.lastig) == 0 {
			regels = append(regels, m.t("dictee.allright"))
		} else {
			regels = append(regels, m.t("dictee.tricky"), "  "+strings.Join(d.lastig, "   "))
		}
		return kop + "\n\n" + paneel(p, m.t("learn.dictation"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.donekeys"))
	}

	w := d.huidig()
	zin := strings.ReplaceAll(w.zin, "___", "_____")
	regels := []string{dim.Render(m.t("dictee.count", d.nr+1, len(d.ronde))), "", "  " + lipgloss.NewStyle().Foreground(p.fg).Render(zin), ""}
	if d.geenStem {
		regels = append(regels, tekst.Foreground(p.geel).Render(m.t("dictee.nosound")), "")
	}
	if d.fout != "" {
		gewoon := lipgloss.NewStyle().Foreground(p.fg).Bold(true)
		regels = append(regels,
			lipgloss.NewStyle().Foreground(p.geel).Render(m.t("dictee.wrong")),
			"  "+dim.Render(fmt.Sprintf("%-6s", m.t("dictee.you")))+metNadruk(d.fout, w.schoon(), gewoon, lipgloss.NewStyle().Foreground(p.geel).Bold(true).Underline(true)),
			"  "+dim.Render(fmt.Sprintf("%-6s", m.t("dictee.correct")))+metNadruk(w.schoon(), d.fout, gewoon, lipgloss.NewStyle().Foreground(p.groen).Bold(true).Underline(true)),
		)
		if strings.Contains(w.woord, "|") {
			regels = append(regels, "  "+dim.Render(m.t("dictee.split", strings.ReplaceAll(w.woord, "|", " · "))))
		}
		if soort := soortVan(w); soort != "" {
			regels = append(regels, "", tekst.Foreground(p.fg).Render(m.t("dictee.rule."+soort)))
		}
		regels = append(regels, "", acc.Render(m.t("dictee.retype")))
	} else if d.vorige != "" {
		regels = append(regels, lipgloss.NewStyle().Foreground(p.groen).Render(d.vorige))
	}
	regels = append(regels, "", d.invoer.View())
	toetsen := m.t("dictee.keys")
	if d.fout != "" {
		toetsen = m.t("dictee.wrongkeys", m.inst.Maatje)
	}
	return kop + "\n\n" + paneel(p, m.t("learn.dictation"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(toetsen)
}
