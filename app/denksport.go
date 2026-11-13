package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Denksport: tafels oefenen (zonder timer, zonder punten) en een rekenmachine.

const (
	denkMenu = iota
	denkTafelsKies
	denkTafels
	denkOverzicht
	denkRekenmachine
)

// maxTafel: de tafels van 1 tot en met 12, elk van 1 × tot en met 12 ×.
const maxTafel = 12

type denkStaat struct {
	scherm, keuze int
	// tafels
	gekozen   [maxTafel + 1]bool // 1..maxTafel
	delen     bool               // ook deelsommen
	kiesSel   int
	vragen    []tafelSom
	nr        int
	antwoord  string
	terugblik string // "Goed!" of de goede som na een fout
	goed      bool
	fouten    []tafelSom
	klaar     bool
	// rekenmachine
	reken    textinput.Model
	geschied []string // som = uitkomst
	vorige   []string // eerdere sommen, voor ↑
}

type tafelSom struct {
	a, b  int
	delen bool
}

func (s tafelSom) tekst() string {
	if s.delen {
		return fmt.Sprintf("%d : %d", s.a*s.b, s.b)
	}
	return fmt.Sprintf("%d × %d", s.a, s.b)
}

func (s tafelSom) uitkomst() int {
	if s.delen {
		return s.a
	}
	return s.a * s.b
}

// sleutel: 7×8 en 8×7 tellen als dezelfde som; deelsommen apart.
func (s tafelSom) sleutel() string {
	a, b := min(s.a, s.b), max(s.a, s.b)
	if s.delen {
		return fmt.Sprintf("%d:%d", s.a*s.b, s.b)
	}
	return fmt.Sprintf("%dx%d", a, b)
}

/* ---------------- voortgang ---------------- */

// tafelVoortgang: per som hoe vaak achter elkaar goed. Drie keer goed = die zit erin.
type tafelVoortgang struct {
	Reeks   map[string]int `json:"reeks"`
	Gekozen []int          `json:"gekozen"`
	Delen   bool           `json:"delen"`
}

func tafelPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "tafels.json") }

func laadTafels() tafelVoortgang {
	v := tafelVoortgang{Reeks: map[string]int{}}
	if data, err := os.ReadFile(tafelPad()); err == nil {
		_ = json.Unmarshal(data, &v)
	}
	if v.Reeks == nil {
		v.Reeks = map[string]int{}
	}
	return v
}

func (v tafelVoortgang) bewaar() {
	_ = os.MkdirAll(filepath.Dir(tafelPad()), 0o755)
	data, _ := json.Marshal(v)
	_ = os.WriteFile(tafelPad(), data, 0o644)
}

func nieuweDenksport() *denkStaat {
	d := &denkStaat{}
	d.reken = textinput.New()
	d.reken.Prompt = "› "
	d.reken.CharLimit = 80
	v := laadTafels()
	for _, t := range v.Gekozen {
		if t >= 1 && t <= maxTafel {
			d.gekozen[t] = true
		}
	}
	if len(v.Gekozen) == 0 {
		for t := 1; t <= maxTafel; t++ {
			d.gekozen[t] = true
		}
	}
	d.delen = v.Delen
	return d
}

// maakRonde: tien sommen uit de gekozen tafels; sommen die er nog niet in zitten komen vaker.
func (d *denkStaat) maakRonde() {
	v := laadTafels()
	var pot []tafelSom
	for t := 1; t <= maxTafel; t++ {
		if !d.gekozen[t] {
			continue
		}
		for n := 1; n <= maxTafel; n++ {
			soorten := []bool{false}
			if d.delen {
				soorten = append(soorten, true)
			}
			for _, deel := range soorten {
				s := tafelSom{a: n, b: t, delen: deel}
				if !deel && rand.Intn(2) == 0 {
					s.a, s.b = t, n
				}
				gewicht := 4 - min(v.Reeks[s.sleutel()], 3) // 4 keer zo vaak als hij hem nog niet kent
				for i := 0; i < gewicht; i++ {
					pot = append(pot, s)
				}
			}
		}
	}
	d.vragen, d.nr, d.antwoord, d.terugblik, d.fouten, d.klaar = nil, 0, "", "", nil, false
	gehad := map[string]bool{}
	rand.Shuffle(len(pot), func(i, j int) { pot[i], pot[j] = pot[j], pot[i] })
	for _, s := range pot {
		if len(d.vragen) == 10 {
			break
		}
		if !gehad[s.sleutel()] {
			gehad[s.sleutel()] = true
			d.vragen = append(d.vragen, s)
		}
	}
}

/* ---------------- toetsen ---------------- */

func (m *model) toetsDenk(msg tea.KeyMsg) (tea.Cmd, bool) {
	d := m.dk
	k := msg.String()
	switch d.scherm {
	case denkMenu:
		switch k {
		case "up", "k", "down", "j":
			d.keuze = 1 - d.keuze
		case "enter":
			if d.keuze == 0 {
				d.scherm = denkTafelsKies
			} else {
				d.scherm = denkRekenmachine
				return d.reken.Focus(), true
			}
		default:
			return nil, false
		}
		return nil, true

	case denkTafelsKies:
		switch k {
		case "up", "k":
			d.kiesSel = max(d.kiesSel-1, 0)
		case "down", "j":
			d.kiesSel = min(d.kiesSel+1, maxTafel)
		case " ", "x":
			if d.kiesSel < maxTafel {
				d.gekozen[d.kiesSel+1] = !d.gekozen[d.kiesSel+1]
			} else {
				d.delen = !d.delen
			}
		case "a":
			alle := true
			for t := 1; t <= maxTafel; t++ {
				alle = alle && d.gekozen[t]
			}
			for t := 1; t <= maxTafel; t++ {
				d.gekozen[t] = !alle
			}
		case "o":
			d.scherm = denkOverzicht
		case "enter":
			v := laadTafels()
			v.Gekozen, v.Delen = nil, d.delen
			for t := 1; t <= maxTafel; t++ {
				if d.gekozen[t] {
					v.Gekozen = append(v.Gekozen, t)
				}
			}
			if len(v.Gekozen) == 0 {
				m.zeg(m.t("tables.pickone"))
				return nil, true
			}
			v.bewaar()
			d.maakRonde()
			d.scherm = denkTafels
		case "esc":
			d.scherm = denkMenu
		default:
			if len(k) == 1 && k[0] >= '1' && k[0] <= '9' { // cijfer: die tafel aan of uit
				d.gekozen[k[0]-'0'] = !d.gekozen[k[0]-'0']
				return nil, true
			}
			return nil, false
		}
		return nil, true

	case denkTafels:
		if d.klaar {
			switch k {
			case "enter", " ":
				d.maakRonde()
			case "esc":
				d.scherm = denkTafelsKies
			case "o":
				d.scherm = denkOverzicht
			default:
				return nil, false
			}
			return nil, true
		}
		switch {
		case k == "esc":
			d.scherm = denkTafelsKies
		case k == "backspace":
			if len(d.antwoord) > 0 {
				d.antwoord = d.antwoord[:len(d.antwoord)-1]
			}
		case msg.Type == tea.KeyRunes && strings.Trim(k, "0123456789") == "":
			d.antwoord = (d.antwoord + k)[:min(len(d.antwoord+k), 4)]
		case k == "enter":
			if d.antwoord == "" {
				return nil, true
			}
			s := d.vragen[d.nr]
			n, _ := strconv.Atoi(d.antwoord)
			v := laadTafels()
			if n == s.uitkomst() {
				d.goed, d.terugblik = true, m.t("tables.right")
				v.Reeks[s.sleutel()]++
			} else {
				d.goed, d.terugblik = false, m.t("tables.wrong", s.tekst(), s.uitkomst())
				v.Reeks[s.sleutel()] = 0
				d.fouten = append(d.fouten, s)
				if len(d.vragen) < 14 { // nog een keer, later in de ronde
					d.vragen = append(d.vragen, s)
				}
			}
			v.bewaar()
			d.antwoord = ""
			d.nr++
			if d.nr >= len(d.vragen) {
				d.klaar = true
			}
		default:
			return nil, true // andere toetsen doen hier niets (ook geen tabwissel met cijfers)
		}
		return nil, true

	case denkOverzicht:
		if k == "esc" || k == "enter" || k == "o" {
			d.scherm = denkTafelsKies
			return nil, true
		}
		return nil, false

	case denkRekenmachine:
		switch k {
		case "esc":
			d.reken.Blur()
			d.scherm = denkMenu
		case "enter":
			som := strings.TrimSpace(d.reken.Value())
			if som == "" {
				return nil, true
			}
			komma := ","
			if m.inst.Taal != "nl" {
				komma = "."
			}
			v, err := reken(som)
			regel := som + "  =  "
			switch {
			case errors.Is(err, errDeelNul):
				regel = som + "  →  " + m.t("calc.divzero")
			case errors.Is(err, errTeGroot):
				regel = som + "  →  " + m.t("calc.toobig")
			case err != nil:
				regel = som + "  →  " + m.t("calc.wrong")
			default:
				regel += toonGetal(v, komma)
			}
			d.geschied = append(d.geschied, regel)
			d.vorige = append(d.vorige, som)
			d.reken.SetValue("")
		case "up":
			if len(d.vorige) > 0 {
				d.reken.SetValue(d.vorige[len(d.vorige)-1])
				d.reken.CursorEnd()
			}
		case "ctrl+l":
			d.geschied = nil
		default:
			var cmd tea.Cmd
			d.reken, cmd = d.reken.Update(msg)
			return cmd, true
		}
		return nil, true
	}
	return nil, false
}

/* ---------------- weergave ---------------- */

func (m model) viewDenk(p palet) string {
	d := m.dk
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	breed := min(m.breed-8, 90)
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.puzzles"))
	switch d.scherm {
	case denkTafelsKies:
		v := laadTafels()
		var regels []string
		for t := 1; t <= maxTafel; t++ {
			vak := "☐"
			if d.gekozen[t] {
				vak = "☑"
			}
			kent := 0
			for n := 1; n <= maxTafel; n++ {
				if v.Reeks[tafelSom{a: n, b: t}.sleutel()] >= 3 {
					kent++
				}
			}
			regel := fmt.Sprintf("%s %s", vak, m.t("tables.table", t)) + dim.Render(fmt.Sprintf("  %s", strings.Repeat("●", kent)+strings.Repeat("·", maxTafel-kent)))
			if t-1 == d.kiesSel {
				regel = acc.Render("▸ ") + regel
			} else {
				regel = "  " + regel
			}
			regels = append(regels, regel)
		}
		vak := "☐"
		if d.delen {
			vak = "☑"
		}
		deel := vak + " " + m.t("tables.division")
		if d.kiesSel == maxTafel {
			deel = acc.Render("▸ ") + deel
		} else {
			deel = "  " + deel
		}
		regels = append(regels, "", deel, "", dim.Render(m.t("tables.legend")))
		return kop + "\n\n" + paneel(p, m.t("tables.pick"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("tables.pickkeys"))

	case denkTafels:
		if d.klaar {
			regels := []string{acc.Render(m.t("tables.done")), ""}
			if len(d.fouten) == 0 {
				regels = append(regels, m.t("tables.allright"))
			} else {
				var lastig []string
				gehad := map[string]bool{}
				for _, s := range d.fouten {
					if !gehad[s.sleutel()] {
						gehad[s.sleutel()] = true
						lastig = append(lastig, fmt.Sprintf("%s = %d", s.tekst(), s.uitkomst()))
					}
				}
				regels = append(regels, m.t("tables.tricky"), "  "+strings.Join(lastig, "   "))
			}
			return kop + "\n\n" + paneel(p, m.t("tables.title"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("tables.donekeys"))
		}
		s := d.vragen[d.nr]
		groot := lipgloss.NewStyle().Bold(true).Foreground(p.fg).Render(s.tekst()+" = ") + acc.Render(d.antwoord+"▌")
		var regels []string
		regels = append(regels, dim.Render(m.t("tables.count", d.nr+1, len(d.vragen))), "", "    "+groot, "")
		if d.terugblik != "" {
			kleur := p.groen
			if !d.goed {
				kleur = p.geel
			}
			regels = append(regels, lipgloss.NewStyle().Foreground(kleur).Render(d.terugblik))
		}
		return kop + "\n\n" + paneel(p, m.t("tables.title"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("tables.keys"))

	case denkOverzicht:
		v := laadTafels()
		kopregel := "      "
		for n := 1; n <= maxTafel; n++ {
			kopregel += fmt.Sprintf("%4d", n)
		}
		regels := []string{dim.Render(kopregel)}
		for t := 1; t <= maxTafel; t++ {
			var cellen []string
			for n := 1; n <= maxTafel; n++ {
				r := v.Reeks[tafelSom{a: n, b: t}.sleutel()]
				kleur := p.lijn
				switch {
				case r >= 3:
					kleur = p.groen
				case r > 0:
					kleur = p.geel
				}
				cellen = append(cellen, lipgloss.NewStyle().Foreground(kleur).Render(fmt.Sprintf("%4d", n*t)))
			}
			regels = append(regels, fmt.Sprintf("%3d × ", t)+strings.Join(cellen, ""))
		}
		regels = append(regels, "", lipgloss.NewStyle().Foreground(p.groen).Render(m.t("tables.known"))+"  "+
			lipgloss.NewStyle().Foreground(p.geel).Render(m.t("tables.learning"))+"  "+lipgloss.NewStyle().Foreground(p.lijn).Render(m.t("tables.notyet")))
		return kop + "\n\n" + paneel(p, m.t("tables.overview"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("tables.overviewkeys"))

	case denkRekenmachine:
		h := max(m.hoog-12, 4)
		regels := d.geschied
		if len(regels) > h {
			regels = regels[len(regels)-h:]
		}
		var opgemaakt []string
		for i, r := range regels {
			stijl := dim
			if i == len(regels)-1 {
				stijl = lipgloss.NewStyle().Foreground(p.fg)
			}
			if som, uit, ok := strings.Cut(r, "  =  "); ok {
				opgemaakt = append(opgemaakt, stijl.Render(som+"  =  ")+acc.Render(uit))
			} else {
				opgemaakt = append(opgemaakt, lipgloss.NewStyle().Foreground(p.geel).Render(r))
			}
		}
		if len(opgemaakt) == 0 {
			opgemaakt = append(opgemaakt, dim.Render(m.t("calc.examples")))
		}
		return kop + "\n\n" + paneel(p, m.t("calc.title"), strings.Join(opgemaakt, "\n")+"\n\n"+d.reken.View(), breed, true) + "\n" + dim.Render(m.t("calc.keys"))
	}
	var regels []string
	for i, sl := range []string{"tables.title", "calc.title"} {
		naam := fmt.Sprintf(" %-18s ", m.t(sl))
		uitleg := m.t(sl + ".d")
		if i == d.keuze {
			regels = append(regels, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸"+naam)+lipgloss.NewStyle().Background(p.sel).Foreground(p.dim).Render(uitleg+" "))
		} else {
			regels = append(regels, " "+naam+dim.Render(uitleg))
		}
	}
	regels = append(regels, "", dim.Render(m.t("buddy.start")), dim.Render(m.t("tables.later")))
	return kop + "\n\n" + paneel(p, m.t("buddy.choose"), strings.Join(regels, "\n"), breed, true)
}
