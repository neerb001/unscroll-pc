package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mijn heelal: alles wat je bij Leren kunt oefenen, als sterren. Elk vak is een sterrenbeeld,
// elk onderdeel een ster die feller gaat schijnen naarmate je het beter kent. Zo zie je in één
// keer wat je al kunt en wat er nog komt.
//
// Alleen schoolvakken (rekenen, topografie, spelling, Engels, geschiedenis): lezen, luisteren,
// spelletjes en avonturen doet een kind uit zichzelf, daar hoort geen beloning bij.
//
// Vijf niveaus, uit de leermotor (de Leitner-bakjes) en de tafels:
//
//	0 nog niet   niets geoefend
//	1 begonnen   wel geoefend
//	2 bekend     de helft ken je goed
//	3 sterk      bijna alles (90%) ken je goed
//	4 meester    bijna alles weet je na meer dan een week nog (bakje 4 of hoger)

const (
	niveauNiet = iota
	niveauBegonnen
	niveauBekend
	niveauSterk
	niveauMeester
)

// meesterBakje: in dit bakje zit een kaart pas als je hem verspreid over ruim een week steeds
// goed had (1 + 2 + 4 dagen).
const meesterBakje = 4

type heelalSter struct {
	naam         string
	niveau       int
	kent, totaal int
	aanDeBeurt   int // kaarten die vandaag herhaald kunnen worden
}

type heelalVak struct {
	naam    string
	sterren []heelalSter
}

// niveauVan: hoe goed je deze kaarten kent, gekeken over een of meer sets (Engels kun je
// vertalen en luisteren: wat je in een van beide goed kent, telt).
func niveauVan(l leerStand, sets []string, sleutels []string, nu time.Time) heelalSter {
	vandaag := nu.Format("2006-01-02")
	s := heelalSter{totaal: len(sleutels)}
	gezien, meester := false, 0
	for _, k := range sleutels {
		bakje, beurt := -1, false
		for _, set := range sets {
			if st := l[set][k]; st != nil {
				gezien = true
				bakje = max(bakje, st.Bakje)
				beurt = beurt || st.Volgende <= vandaag
			}
		}
		if bakje >= kentBakje {
			s.kent++
		}
		if bakje >= meesterBakje {
			meester++
		}
		if bakje >= 0 && beurt {
			s.aanDeBeurt++
		}
	}
	s.niveau = niveauUit(gezien, s.kent, meester, s.totaal)
	return s
}

func niveauUit(gezien bool, kent, meester, totaal int) int {
	switch {
	case !gezien || totaal == 0:
		return niveauNiet
	case meester*10 >= totaal*9:
		return niveauMeester
	case kent*10 >= totaal*9:
		return niveauSterk
	case kent*2 >= totaal:
		return niveauBekend
	}
	return niveauBegonnen
}

// tafelSter: de tafels houden hun eigen reeksen bij (drie keer achter elkaar goed = die zit
// erin). Meester: zes keer achter elkaar goed.
func tafelSter(v tafelVoortgang, t int) heelalSter {
	s := heelalSter{totaal: maxTafel}
	gezien, meester := false, 0
	for n := 1; n <= maxTafel; n++ {
		r, ok := v.Reeks[tafelSom{a: n, b: t}.sleutel()]
		gezien = gezien || ok
		if r >= 3 {
			s.kent++
		}
		if r >= 6 {
			meester++
		}
	}
	s.niveau = niveauUit(gezien, s.kent, meester, s.totaal)
	return s
}

func (m *model) heelalVakken(nu time.Time) []heelalVak {
	l := laadLeren()
	var vakken []heelalVak

	tv := laadTafels()
	tafels := heelalVak{naam: m.t("tables.title")}
	for t := 1; t <= 10; t++ {
		s := tafelSter(tv, t)
		s.naam = m.t("tables.table", t)
		tafels.sterren = append(tafels.sterren, s)
	}
	vakken = append(vakken, tafels)

	topo := heelalVak{naam: m.t("learn.topo")}
	for _, o := range topoOefeningen {
		if o.vraag == "kijk" {
			continue
		}
		var codes []string
		for _, d := range o.dingen() {
			codes = append(codes, d.Code)
		}
		s := niveauVan(l, []string{o.set()}, codes, nu)
		s.naam = m.t("topo.map."+o.kaart) + ": " + m.t(o.sleutel)
		topo.sterren = append(topo.sterren, s)
	}
	vakken = append(vakken, topo)

	dictee := heelalVak{naam: m.t("learn.dictation")}
	for _, d := range dicteeSoorten {
		var sl []string
		for _, w := range dicteeWoorden(d.sleutel) {
			sl = append(sl, w.schoon())
		}
		s := niveauVan(l, []string{dicteeSet(d.sleutel), dicteeSet(dicteeAlles)}, sl, nu)
		s.naam = m.t("dictee.cat." + d.sleutel)
		dictee.sterren = append(dictee.sterren, s)
	}
	for _, set := range flitsSets {
		s := niveauVan(l, []string{flitsSet(set)}, flitsWoorden(set), nu)
		s.naam = m.t("learn.flash") + ": " + m.t("flits.set."+set)
		dictee.sterren = append(dictee.sterren, s)
	}
	vakken = append(vakken, dictee)

	engels := heelalVak{naam: m.t("learn.english")}
	for _, th := range engelsThemas {
		var sl []string
		for _, w := range engelsWoorden(th.sleutel) {
			sl = append(sl, w.sleutel())
		}
		sets := []string{engelsSet(false, th.sleutel), engelsSet(true, th.sleutel), engelsSet(false, dicteeAlles), engelsSet(true, dicteeAlles)}
		s := niveauVan(l, sets, sl, nu)
		s.naam = m.t("engels.theme." + th.sleutel)
		engels.sterren = append(engels.sterren, s)
	}
	vakken = append(vakken, engels)

	gesch := heelalVak{naam: m.t("learn.history")}
	for _, i := range []int{geschVak, geschJaar} {
		var sl []string
		for _, x := range alleGebeurtenissen() {
			if i == geschVak || !x.rond {
				sl = append(sl, x.tekst)
			}
		}
		s := niveauVan(l, []string{geschSet(i)}, sl, nu)
		s.naam = m.t([]string{"hist.timeline", "hist.whichvak", "hist.whichyear"}[i])
		gesch.sterren = append(gesch.sterren, s)
	}
	vakken = append(vakken, gesch)
	return vakken
}

/* ---------------- tekenen ---------------- */

// sterTekens: per niveau een teken; op de eenvoudige tekstconsole gewone letters.
func sterTeken(niveau int) string {
	if alleenLetters() {
		return []string{".", "o", "O", "*", "#"}[niveau]
	}
	return []string{"·", "○", "◉", "★", "✪"}[niveau]
}

func sterKleur(p palet, niveau int) lipgloss.Color {
	return []lipgloss.Color{p.lijn, p.dim, p.acc, p.geel, p.groen}[niveau]
}

type heelalStaat struct {
	keuze int  // welk vak
	open  bool // de sterren van dat vak met naam
	boven int
}

func (m *model) toetsHeelal(msg tea.KeyMsg) (tea.Cmd, bool) {
	h := m.lr.heelal
	n := len(m.heelalVakken(m.nu))
	switch k := msg.String(); k {
	case "esc":
		if h.open {
			h.open = false
		} else {
			m.lr.scherm = lerenMenu
		}
	case "enter", "right", "l":
		h.open, h.boven = true, 0
	case "left", "h":
		h.open = false
	case "up", "k", "down", "j", "pgup", "pgdown":
		if h.open {
			scrollToets(k, &h.boven, max(m.hoog-14, 3))
			return nil, true
		}
		if k == "up" || k == "k" {
			h.keuze = (h.keuze + n - 1) % n
		} else if k == "down" || k == "j" {
			h.keuze = (h.keuze + 1) % n
		}
	default:
		return nil, false
	}
	return nil, true
}

func (m model) viewHeelal(p palet, kop string, breed int) string {
	h := m.lr.heelal
	vakken := m.heelalVakken(m.nu)
	h.keuze = min(h.keuze, len(vakken)-1)
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	ster := func(niveau int) string {
		return lipgloss.NewStyle().Foreground(sterKleur(p, niveau)).Bold(niveau >= niveauSterk).Render(sterTeken(niveau))
	}

	telling := make([]int, niveauMeester+1)
	beurt, beurtVak := 0, ""
	var volgende *heelalSter
	for _, v := range vakken {
		for i, s := range v.sterren {
			telling[s.niveau]++
			if s.aanDeBeurt > beurt {
				beurt, beurtVak = s.aanDeBeurt, v.naam+": "+s.naam
			}
			if s.niveau == niveauBegonnen && (volgende == nil || s.kent > volgende.kent) {
				volgende = &v.sterren[i]
			}
		}
	}
	legenda := make([]string, 0, niveauMeester+1)
	for n := niveauNiet; n <= niveauMeester; n++ {
		legenda = append(legenda, ster(n)+" "+m.t(fmt.Sprintf("univ.level.%d", n))+dim.Render(fmt.Sprintf(" %d", telling[n])))
	}

	var regels []string
	if h.open {
		v := vakken[h.keuze]
		var rijen []string
		for _, s := range v.sterren {
			naam := s.naam
			if lipgloss.Width(naam) > breed-26 {
				naam = string([]rune(naam)[:max(breed-29, 5)]) + "…"
			}
			regel := " " + ster(s.niveau) + "  " + naam + strings.Repeat(" ", max(breed-22-lipgloss.Width(naam), 1)) +
				dim.Render(fmt.Sprintf("%3d/%-3d", s.kent, s.totaal))
			rijen = append(rijen, regel)
		}
		hoog := max(m.hoog-14, 3)
		h.boven = max(0, min(h.boven, len(rijen)-hoog))
		eind := min(len(rijen), h.boven+hoog)
		regels = append(regels, acc.Render(v.naam), "")
		regels = append(regels, rijen[h.boven:eind]...)
		if eind < len(rijen) {
			regels = append(regels, dim.Render(m.t("univ.more")))
		}
		regels = append(regels, "", strings.Join(legenda, "   "))
		return kop + "\n\n" + paneel(p, m.t("univ.title"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("univ.openkeys"))
	}

	naamBreed := 0
	for _, v := range vakken {
		naamBreed = max(naamBreed, lipgloss.Width(v.naam))
	}
	for i, v := range vakken {
		var rij strings.Builder
		for _, s := range v.sterren {
			rij.WriteString(ster(s.niveau))
			rij.WriteString(" ")
		}
		naam := v.naam + strings.Repeat(" ", naamBreed-lipgloss.Width(v.naam))
		if i == h.keuze {
			naam = acc.Render("▸ " + naam)
		} else {
			naam = "  " + naam
		}
		regels = append(regels, naam+" "+rij.String(), "")
	}
	regels = append(regels, strings.Join(legenda, "   "), "")
	switch {
	case beurt > 0:
		regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("univ.due", beurt, beurtVak)))
	case volgende != nil:
		regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("univ.next", volgende.naam)))
	default:
		regels = append(regels, dim.Render(m.t("univ.start")))
	}
	return kop + "\n\n" + paneel(p, m.t("univ.title"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("univ.keys"))
}
