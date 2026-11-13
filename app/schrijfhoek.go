package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Schrijfhoek: een eenvoudige takenlijst en de eigen verhalen teruglezen. Alles is privé:
// het staat alleen in de thuismap van het kind en komt niet in de ouderstand.

const (
	schrijfMenu = iota
	schrijfTaken
	schrijfVerhalen
	schrijfLees
	schrijfWoord
)

type taak struct {
	Tekst   string    `json:"tekst"`
	Klaar   bool      `json:"klaar"`
	Gemaakt time.Time `json:"gemaakt"`
	KlaarOp time.Time `json:"klaar_op,omitempty"`
}

type schrijfStaat struct {
	scherm, keuze int
	taken         []taak
	sel           int
	invoer        textinput.Model
	typen         bool
	bewerk        int // -1: nieuwe taak
	weg           int // bevestigen: nog een keer Delete om weg te halen
	verhalen      []verhaalBestand
	verhaalSel    int
	leesRegels    []string
	leesTitel     string
	leesBoven     int
	wb            *woordStaat
}

func nieuweSchrijfhoek() *schrijfStaat {
	s := &schrijfStaat{weg: -1, bewerk: -1, wb: nieuwWoordenboek()}
	s.invoer = textinput.New()
	s.invoer.CharLimit = 120
	s.invoer.Prompt = "☐ "
	s.taken = laadTaken()
	return s
}

func takenPad() string { return filepath.Join(unscrollData(""), "taken.json") }

func laadTaken() []taak {
	var t []taak
	if data, err := os.ReadFile(takenPad()); err == nil {
		_ = json.Unmarshal(data, &t)
	}
	return t
}

func bewaarTaken(t []taak) {
	_ = os.MkdirAll(filepath.Dir(takenPad()), 0o700)
	data, _ := json.MarshalIndent(t, "", "  ")
	_ = os.WriteFile(takenPad(), data, 0o600)
}

// sorteerTaken: open taken boven, in de volgorde waarin ze gemaakt zijn; afgevinkte eronder.
func sorteerTaken(t []taak) {
	sort.SliceStable(t, func(i, j int) bool { return !t[i].Klaar && t[j].Klaar })
}

func openTaken(t []taak) []taak {
	var uit []taak
	for _, x := range t {
		if !x.Klaar {
			uit = append(uit, x)
		}
	}
	return uit
}

/* ---------------- toetsen ---------------- */

func (m *model) toetsSchrijf(msg tea.KeyMsg) (tea.Cmd, bool) {
	s := m.sh
	k := msg.String()
	switch s.scherm {
	case schrijfMenu:
		switch k {
		case "up", "k":
			s.keuze = (s.keuze + 2) % 3
		case "down", "j":
			s.keuze = (s.keuze + 1) % 3
		case "enter":
			switch s.keuze {
			case 0:
				s.scherm, s.taken = schrijfTaken, laadTaken()
			case 1:
				s.scherm = schrijfWoord
				s.wb.invoer.SetValue("")
				return s.wb.invoer.Focus(), true
			default:
				s.scherm, s.verhalen, s.verhaalSel = schrijfVerhalen, eerdereVerhalen(100), 0
			}
		default:
			return nil, false
		}
		return nil, true

	case schrijfTaken:
		if s.typen {
			switch k {
			case "esc":
				s.typen = false
				s.invoer.Blur()
			case "enter":
				tekst := strings.TrimSpace(s.invoer.Value())
				s.typen = false
				s.invoer.Blur()
				if tekst == "" {
					return nil, true
				}
				if s.bewerk >= 0 && s.bewerk < len(s.taken) {
					s.taken[s.bewerk].Tekst = tekst
				} else {
					// nieuwe taak onderaan de open taken
					s.taken = append(s.taken, taak{Tekst: tekst, Gemaakt: time.Now()})
					sorteerTaken(s.taken)
					s.sel = len(openTaken(s.taken)) - 1
				}
				bewaarTaken(s.taken)
			default:
				var cmd tea.Cmd
				s.invoer, cmd = s.invoer.Update(msg)
				return cmd, true
			}
			return nil, true
		}
		if k != "delete" && k != "backspace" {
			s.weg = -1
		}
		switch k {
		case "up", "k":
			s.sel = max(s.sel-1, 0)
		case "down", "j":
			s.sel = min(s.sel+1, max(len(s.taken)-1, 0))
		case "n", "a", "+":
			s.typen, s.bewerk = true, -1
			s.invoer.SetValue("")
			return s.invoer.Focus(), true
		case "e":
			if s.sel < len(s.taken) {
				s.typen, s.bewerk = true, s.sel
				s.invoer.SetValue(s.taken[s.sel].Tekst)
				s.invoer.CursorEnd()
				return s.invoer.Focus(), true
			}
		case " ", "enter", "x":
			if s.sel < len(s.taken) {
				t := &s.taken[s.sel]
				t.Klaar = !t.Klaar
				if t.Klaar {
					t.KlaarOp = time.Now()
					m.zeg(m.t("todo.welldone"))
				}
				gekozen := *t
				sorteerTaken(s.taken)
				for i := range s.taken {
					if s.taken[i] == gekozen {
						s.sel = i
					}
				}
				bewaarTaken(s.taken)
			}
		case "delete", "backspace":
			if s.sel < len(s.taken) {
				if s.weg != s.sel {
					s.weg = s.sel // eerst bevestigen
					return nil, true
				}
				s.taken = append(s.taken[:s.sel], s.taken[s.sel+1:]...)
				s.sel = min(s.sel, max(len(s.taken)-1, 0))
				s.weg = -1
				bewaarTaken(s.taken)
			}
		case "o":
			s.taken = openTaken(s.taken)
			s.sel = min(s.sel, max(len(s.taken)-1, 0))
			bewaarTaken(s.taken)
		case "esc":
			s.scherm = schrijfMenu
		default:
			return nil, false
		}
		return nil, true

	case schrijfVerhalen:
		switch k {
		case "up", "k":
			s.verhaalSel = max(s.verhaalSel-1, 0)
		case "down", "j":
			s.verhaalSel = min(s.verhaalSel+1, max(len(s.verhalen)-1, 0))
		case "enter":
			if s.verhaalSel < len(s.verhalen) {
				v := s.verhalen[s.verhaalSel]
				data, err := os.ReadFile(v.pad)
				if err != nil {
					return nil, true
				}
				breed := max(min(m.breed-14, 76), 30)
				s.leesRegels = strings.Split(lipgloss.NewStyle().Width(breed).Render(strings.TrimSpace(string(data))), "\n")
				s.leesTitel, s.leesBoven, s.scherm = v.titel, 0, schrijfLees
			}
		case "esc":
			s.scherm = schrijfMenu
		default:
			return nil, false
		}
		return nil, true

	case schrijfWoord:
		return m.toetsWoord(msg)

	case schrijfLees:
		pagina := max(m.hoog-10, 5)
		switch k {
		case "down", "j":
			s.leesBoven = min(s.leesBoven+1, max(len(s.leesRegels)-pagina, 0))
		case "up", "k":
			s.leesBoven = max(s.leesBoven-1, 0)
		case "pgdown", " ", "right":
			s.leesBoven = min(s.leesBoven+pagina-1, max(len(s.leesRegels)-pagina, 0))
		case "pgup", "left":
			s.leesBoven = max(s.leesBoven-pagina+1, 0)
		case "esc":
			s.scherm = schrijfVerhalen
		default:
			return nil, false
		}
		return nil, true
	}
	return nil, false
}

/* ---------------- weergave ---------------- */

func (m model) viewSchrijf(p palet) string {
	s := m.sh
	dim := lipgloss.NewStyle().Foreground(p.dim)
	breed := min(m.breed-8, 90)
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.write"))
	kies := func(i int, naam, uitleg string) string {
		naam = fmt.Sprintf(" %-16s ", naam)
		if i == s.keuze {
			return lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸"+naam) + lipgloss.NewStyle().Background(p.sel).Foreground(p.dim).Render(uitleg+" ")
		}
		return " " + naam + dim.Render(uitleg)
	}
	switch s.scherm {
	case schrijfWoord:
		return kop + "\n\n" + paneel(p, m.t("dict.title"), m.viewWoord(p, breed), breed, true) + "\n" + dim.Render(m.t("dict.keys"))
	case schrijfTaken:
		var regels []string
		open := len(openTaken(s.taken))
		if len(s.taken) == 0 && !s.typen {
			regels = append(regels, dim.Render(m.t("todo.empty")))
		}
		for i, t := range s.taken {
			vak, stijl := "☐ ", lipgloss.NewStyle().Foreground(p.fg)
			if t.Klaar {
				vak, stijl = "☑ ", lipgloss.NewStyle().Foreground(p.dim).Strikethrough(true)
			}
			tekst := t.Tekst
			if s.typen && s.bewerk == i {
				regels = append(regels, "  "+s.invoer.View())
				continue
			}
			regel := vak + stijl.Render(tekst)
			if i == s.weg {
				regel += lipgloss.NewStyle().Foreground(p.geel).Render("  " + m.t("todo.confirm"))
			}
			if i == s.sel && !s.typen {
				regels = append(regels, lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render("▸ ")+regel)
			} else {
				regels = append(regels, "  "+regel)
			}
			if i == open-1 && open < len(s.taken) {
				regels = append(regels, "") // ruimte tussen open en klaar
			}
		}
		if s.typen && s.bewerk < 0 {
			regels = append(regels, "", "  "+s.invoer.View())
		}
		hoogte := max(m.hoog-10, 5)
		if len(regels) > hoogte {
			begin := min(max(s.sel-hoogte/2, 0), len(regels)-hoogte)
			regels = regels[begin : begin+hoogte]
		}
		titel := m.t("todo.title", open)
		toetsen := m.t("todo.keys")
		if s.typen {
			toetsen = m.t("todo.typekeys")
		}
		return kop + "\n\n" + paneel(p, titel, strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(toetsen)

	case schrijfVerhalen:
		var regels []string
		if len(s.verhalen) == 0 {
			regels = append(regels, dim.Render(m.t("write.nostories")))
		}
		hoogte := max(m.hoog-10, 5)
		begin := min(max(s.verhaalSel-hoogte/2, 0), max(len(s.verhalen)-hoogte, 0))
		for i := begin; i < len(s.verhalen) && i < begin+hoogte; i++ {
			v := s.verhalen[i]
			naam := v.tijd.Format("2 Jan 2006") + "  " + v.titel
			if i == s.verhaalSel {
				regels = append(regels, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸ "+naam+" "))
			} else {
				regels = append(regels, "  "+naam)
			}
		}
		return kop + "\n\n" + paneel(p, m.t("write.stories"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("write.storykeys"))

	case schrijfLees:
		pagina := max(m.hoog-10, 5)
		eind := min(s.leesBoven+pagina, len(s.leesRegels))
		tekst := strings.Join(s.leesRegels[s.leesBoven:eind], "\n")
		return kop + "\n\n" + paneel(p, s.leesTitel, tekst, breed, true) + "\n" + dim.Render(m.t("write.readkeys"))
	}
	regels := []string{
		kies(0, m.t("todo.name"), m.t("todo.desc", len(openTaken(s.taken)))),
		kies(1, m.t("dict.title"), m.t("dict.desc")),
		kies(2, m.t("write.stories"), m.t("write.storiesdesc")),
		"", dim.Render(m.t("buddy.start")), dim.Render(m.t("write.private")),
	}
	return kop + "\n\n" + paneel(p, m.t("buddy.choose"), strings.Join(regels, "\n"), breed, true)
}

// takenKort: de open taken voor het startscherm.
func (m model) takenKort(max_ int) string {
	open := openTaken(m.sh.taken)
	if len(open) == 0 {
		return m.t("todo.none")
	}
	var r []string
	for i, t := range open {
		if i == max_ {
			r = append(r, m.t("todo.more", len(open)-max_))
			break
		}
		r = append(r, "☐ "+t.Tekst)
	}
	return strings.Join(r, "\n")
}
