package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Leren: oefenen voor school, rustig en zonder punten. Dictee, flitslezen en klokkijken zijn
// van hier; de tafels en de rekenmachines delen hun schermen met Denksport (dat houdt sudoku,
// raadsels en schaken).

const (
	lerenMenu = iota
	lerenDictee
	lerenFlits
	lerenKlok
	lerenDenk // een scherm van Denksport (tafels, rekenmachine), maar dan onder Leren
)

type lerenItem struct {
	sleutel string
	scherm  int
	denk    int // bij lerenDenk: welk Denksport-scherm
}

var lerenItems = []lerenItem{
	{"learn.dictation", lerenDictee, 0},
	{"learn.flash", lerenFlits, 0},
	{"learn.clock", lerenKlok, 0},
	{"tables.title", lerenDenk, denkTafelsKies},
	{"calc.title", lerenDenk, denkKnoppen},
	{"calc.typed", lerenDenk, denkRekenmachine},
}

// leerSchermen: deze Denksport-schermen horen bij Leren
var leerSchermen = map[int]bool{denkTafelsKies: true, denkTafels: true, denkOverzicht: true, denkKnoppen: true, denkRekenmachine: true}

type lerenStaat struct {
	scherm, keuze int
	dictee        *dicteeStaat
	flits         *flitsStaat
	klok          *klokStaat
}

func nieuwLeren() *lerenStaat {
	return &lerenStaat{dictee: nieuwDictee(), flits: nieuwFlitsen(), klok: nieuweKlok()}
}

// lerenOpen: bij het openen van Leren of Denksport niet in een scherm van de ander blijven hangen.
func (m *model) tabOpen(t int) {
	d := m.dk
	switch t {
	case tabDenksport:
		if leerSchermen[d.scherm] {
			d.reken.Blur()
			d.scherm = denkMenu
		}
		if m.lr.scherm == lerenDenk {
			m.lr.scherm = lerenMenu
		}
	case tabLeren:
		if m.lr.scherm == lerenDenk && !leerSchermen[d.scherm] {
			m.lr.scherm = lerenMenu
		}
	}
}

func (m *model) toetsLeren(msg tea.KeyMsg) (tea.Cmd, bool) {
	l := m.lr
	switch l.scherm {
	case lerenDictee:
		return m.toetsDictee(msg)
	case lerenFlits:
		return m.toetsFlits(msg)
	case lerenKlok:
		return m.toetsKlok(msg)
	case lerenDenk:
		cmd, ok := m.toetsDenk(msg)
		if m.dk.scherm == denkMenu { // esc in een Denksport-scherm: terug naar het menu van Leren
			l.scherm = lerenMenu
		}
		return cmd, ok
	}
	switch msg.String() {
	case "up", "k":
		l.keuze = (l.keuze + len(lerenItems) - 1) % len(lerenItems)
	case "down", "j":
		l.keuze = (l.keuze + 1) % len(lerenItems)
	case "enter":
		it := lerenItems[l.keuze]
		l.scherm = it.scherm
		switch it.scherm {
		case lerenDictee:
			l.dictee.scherm = dicteeKiezen
		case lerenFlits:
			l.flits.scherm = flitsKiezen
		case lerenKlok:
			l.klok.terug = nil
			l.klok.nieuweTijd()
			return l.klok.invoer.Focus(), true
		case lerenDenk:
			return m.openDenk(it.denk), true
		}
	default:
		return nil, false
	}
	return nil, true
}

func (m model) viewLeren(p palet) string {
	l := m.lr
	breed := min(m.breed-8, 90)
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.learn"))
	switch l.scherm {
	case lerenDictee:
		return m.viewDictee(p, kop, breed)
	case lerenFlits:
		return m.viewFlits(p, kop, breed)
	case lerenKlok:
		return m.viewKlok(p, kop, breed)
	case lerenDenk:
		return m.viewDenk(p)
	}
	dim := lipgloss.NewStyle().Foreground(p.dim)
	var regels []string
	for i, it := range lerenItems {
		naam := fmt.Sprintf(" %-22s ", m.t(it.sleutel))
		uitleg := m.t(it.sleutel + ".d")
		if i == l.keuze {
			regels = append(regels, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸"+naam)+lipgloss.NewStyle().Background(p.sel).Foreground(p.dim).Render(uitleg+" "))
		} else {
			regels = append(regels, " "+naam+dim.Render(uitleg))
		}
	}
	regels = append(regels, "", dim.Render(m.t("buddy.start")), dim.Render(m.t("tables.later")))
	return kop + "\n\n" + paneel(p, m.t("buddy.choose"), strings.Join(regels, "\n"), breed, true)
}
