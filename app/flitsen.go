package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Flitslezen: een woord staat heel even in beeld en het kind leest het hardop. Daarna kijkt het
// zelf of het klopte. Dat traint snel herkennen van woordbeelden, en met de b/d-set ook het
// verschil tussen b en d. Woorden die niet goed gingen komen later in de ronde en op een
// volgende dag terug. Geen punten: het kind beoordeelt zichzelf.

const (
	flitsKiezen    = iota
	flitsKlaarVoor // wacht op spatie
	flitsTonen     // het woord staat in beeld
	flitsWeg       // het woord is weg: hardop lezen
	flitsCheck     // het woord staat er weer: goed gelezen?
	flitsKlaar
)

// de sets: b en d (met p, omdat die er ook op lijkt) of alle dictee-woorden
var flitsSets = []string{"bd", dicteeAlles}

var flitsExtraBD = []string{"bed", "bad", "dak", "bak", "doos", "boos", "dal", "bal", "dek", "bek", "bos", "dop", "pad", "bord", "dorp", "bidden", "pop", "bodem", "deeg", "beer", "peer", "duif", "buik", "dief", "pad"}

type flitsStaat struct {
	scherm int
	keuze  int
	duur   time.Duration
	ronde  []string
	nr, id int
	gehad  map[string]bool
	lastig []string
}

type flitsMsg struct{ id int }

func nieuwFlitsen() *flitsStaat { return &flitsStaat{duur: 1200 * time.Millisecond} }

func flitsWoorden(set string) []string {
	gezien := map[string]bool{}
	var uit []string
	voeg := func(w string) {
		if !gezien[w] {
			gezien[w] = true
			uit = append(uit, w)
		}
	}
	if set == "bd" {
		for _, w := range dicteeWoorden("bd") {
			voeg(w.schoon())
		}
		for _, w := range flitsExtraBD {
			voeg(w)
		}
		return uit
	}
	for _, w := range dicteeWoorden(dicteeAlles) {
		voeg(w.schoon())
	}
	return uit
}

func flitsSet(set string) string { return "flits:" + set }

func (m *model) flitsStart() {
	f := m.lr.flits
	set := flitsSets[f.keuze]
	f.ronde = laadLeren().kies(flitsSet(set), flitsWoorden(set), 10, time.Now())
	f.nr, f.gehad, f.lastig = 0, map[string]bool{}, nil
	f.scherm = flitsKlaarVoor
}

func (m *model) flitsBericht(msg flitsMsg) {
	f := m.lr.flits
	if msg.id == f.id && f.scherm == flitsTonen {
		f.scherm = flitsWeg
	}
}

func (m *model) toetsFlits(msg tea.KeyMsg) (tea.Cmd, bool) {
	f := m.lr.flits
	k := msg.String()
	if k == "esc" {
		if f.scherm == flitsKiezen {
			m.lr.scherm = lerenMenu
		} else {
			f.scherm = flitsKiezen
		}
		return nil, true
	}
	switch k {
	case "+", "=":
		f.duur = min(f.duur+200*time.Millisecond, 3*time.Second)
		return nil, true
	case "-":
		f.duur = max(f.duur-200*time.Millisecond, 200*time.Millisecond)
		return nil, true
	}
	switch f.scherm {
	case flitsKiezen:
		switch k {
		case "up", "k", "down", "j":
			f.keuze = 1 - f.keuze
		case "enter", " ":
			m.flitsStart()
		default:
			return nil, false
		}
	case flitsKlaarVoor:
		if k != " " && k != "enter" {
			return nil, false
		}
		f.scherm = flitsTonen
		f.id++
		id := f.id
		return tea.Tick(f.duur, func(time.Time) tea.Msg { return flitsMsg{id} }), true
	case flitsTonen:
		return nil, true // even kijken; toetsen doen nu niets
	case flitsWeg:
		if k != " " && k != "enter" {
			return nil, false
		}
		f.scherm = flitsCheck
	case flitsCheck:
		var goed bool
		switch k {
		case "j", "y":
			goed = true
		case "n":
		default:
			return nil, false
		}
		w := f.ronde[f.nr]
		if !f.gehad[w] {
			f.gehad[w] = true
			l := laadLeren()
			l.zet(flitsSet(flitsSets[f.keuze]), w, goed, time.Now())
			l.bewaar()
		}
		if !goed {
			f.lastig = append(f.lastig, w)
			if len(f.ronde) < 14 {
				f.ronde = append(f.ronde, w)
			}
		}
		f.nr++
		f.scherm = flitsKlaarVoor
		if f.nr >= len(f.ronde) {
			f.scherm = flitsKlaar
		}
	case flitsKlaar:
		switch k {
		case "enter", " ":
			m.flitsStart()
		default:
			return nil, false
		}
	}
	return nil, true
}

// groot zet spaties tussen de letters, zodat elke letter goed te zien is.
func groot(w string) string { return strings.Join(strings.Split(w, ""), " ") }

func (m model) viewFlits(p palet, kop string, breed int) string {
	f := m.lr.flits
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	snelheid := dim.Render(m.t("flits.speed", fmt.Sprintf("%.1f", f.duur.Seconds())))
	if m.inst.Taal == "nl" {
		snelheid = strings.Replace(snelheid, ".", ",", 1)
	}
	if f.scherm == flitsKiezen {
		l := laadLeren()
		var regels []string
		for i, set := range flitsSets {
			woorden := flitsWoorden(set)
			naam := fmt.Sprintf("%-20s", m.t("flits.set."+set))
			stip := dim.Render(fmt.Sprintf("  %d/%d", l.kent(flitsSet(set), woorden), len(woorden)))
			if i == f.keuze {
				regels = append(regels, acc.Render("▸ "+naam)+stip)
			} else {
				regels = append(regels, "  "+naam+stip)
			}
		}
		regels = append(regels, "", snelheid, "", dim.Render(m.t("dictee.legend")))
		return kop + "\n\n" + paneel(p, m.t("learn.flash"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("flits.pickkeys"))
	}
	if f.scherm == flitsKlaar {
		regels := []string{acc.Render(m.t("dictee.done")), ""}
		if len(f.lastig) == 0 {
			regels = append(regels, m.t("flits.allright"))
		} else {
			regels = append(regels, m.t("flits.tricky"), "  "+strings.Join(f.lastig, "   "))
		}
		return kop + "\n\n" + paneel(p, m.t("learn.flash"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.donekeys"))
	}

	// het woord in het midden van een vak, even hoog in elke fase zodat er niets verspringt
	woord := ""
	uitleg := ""
	switch f.scherm {
	case flitsKlaarVoor:
		woord = dim.Render("·")
		uitleg = m.t("flits.ready")
	case flitsTonen:
		woord = lipgloss.NewStyle().Bold(true).Foreground(p.fg).Render(groot(f.ronde[f.nr]))
	case flitsWeg:
		woord = dim.Render(strings.Repeat("? ", len([]rune(f.ronde[f.nr]))))
		uitleg = m.t("flits.gone")
	case flitsCheck:
		woord = acc.Render(groot(f.ronde[f.nr]))
		uitleg = m.t("flits.check")
	}
	vak := lipgloss.Place(breed-6, 5, lipgloss.Center, lipgloss.Center, woord)
	regels := []string{dim.Render(m.t("dictee.count", f.nr+1, len(f.ronde))), vak, lipgloss.NewStyle().Width(breed - 6).Render(uitleg), "", snelheid}
	return kop + "\n\n" + paneel(p, m.t("learn.flash"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("flits.keys"))
}
