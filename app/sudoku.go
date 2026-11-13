package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Sudoku: de app maakt de puzzels zelf, altijd met precies één oplossing. Geen klok, geen punten.
// Met c controleer je wat je hebt ingevuld, met h krijg je één vakje cadeau.

type sudokuRooster [81]int

type sudoku struct {
	Puzzel    sudokuRooster `json:"puzzel"`
	Oplossing sudokuRooster `json:"oplossing"`
	Ingevuld  sudokuRooster `json:"ingevuld"`
	Niveau    int           `json:"niveau"` // 0 makkelijk, 1 gemiddeld, 2 moeilijk
	Hints     int           `json:"hints"`
}

func magHier(r *sudokuRooster, cel, cijfer int) bool {
	rij, kol := cel/9, cel%9
	br, bk := rij/3*3, kol/3*3
	for i := 0; i < 9; i++ {
		if r[rij*9+i] == cijfer || r[i*9+kol] == cijfer || r[(br+i/3)*9+bk+i%3] == cijfer {
			return false
		}
	}
	return true
}

// vul: een volledig, willekeurig ingevuld rooster (terugzoeken met geschudde cijfers).
func vul(r *sudokuRooster, rnd *rand.Rand) bool {
	for cel := 0; cel < 81; cel++ {
		if r[cel] != 0 {
			continue
		}
		cijfers := rnd.Perm(9)
		for _, c := range cijfers {
			if magHier(r, cel, c+1) {
				r[cel] = c + 1
				if vul(r, rnd) {
					return true
				}
				r[cel] = 0
			}
		}
		return false
	}
	return true
}

// telOplossingen telt tot max; genoeg om te zien of er precies één is.
func telOplossingen(r *sudokuRooster, max int) int {
	// het lege vakje met de minste mogelijkheden eerst: veel sneller
	beste, minst := -1, 10
	for cel := 0; cel < 81; cel++ {
		if r[cel] != 0 {
			continue
		}
		n := 0
		for c := 1; c <= 9; c++ {
			if magHier(r, cel, c) {
				n++
			}
		}
		if n < minst {
			beste, minst = cel, n
		}
	}
	if beste < 0 {
		return 1
	}
	tel := 0
	for c := 1; c <= 9 && tel < max; c++ {
		if magHier(r, beste, c) {
			r[beste] = c
			tel += telOplossingen(r, max-tel)
			r[beste] = 0
		}
	}
	return tel
}

var sudokuGegeven = []int{40, 32, 26} // ongeveer zoveel cijfers staan er al

func maakSudoku(niveau int, rnd *rand.Rand) sudoku {
	var vol sudokuRooster
	vul(&vol, rnd)
	puzzel := vol
	weg := 81 - sudokuGegeven[niveau]
	for _, cel := range rnd.Perm(81) {
		if weg == 0 {
			break
		}
		oud := puzzel[cel]
		puzzel[cel] = 0
		kopie := puzzel
		if telOplossingen(&kopie, 2) != 1 {
			puzzel[cel] = oud // dan zou er meer dan één oplossing zijn
			continue
		}
		weg--
	}
	return sudoku{Puzzel: puzzel, Oplossing: vol, Ingevuld: puzzel, Niveau: niveau}
}

func sudokuPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "sudoku.json") }

func laadSudoku() *sudoku {
	var s sudoku
	if data, err := os.ReadFile(sudokuPad()); err == nil && json.Unmarshal(data, &s) == nil && s.Oplossing[0] != 0 {
		return &s
	}
	return nil
}

func (s *sudoku) bewaar() {
	_ = os.MkdirAll(filepath.Dir(sudokuPad()), 0o755)
	data, _ := json.Marshal(s)
	_ = os.WriteFile(sudokuPad(), data, 0o644)
}

func (s *sudoku) klaar() bool { return s.Ingevuld == s.Oplossing }

/* ---------------- scherm ---------------- */

type sudokuStaat struct {
	s          *sudoku
	cel        int
	kiesNiveau bool
	niveauSel  int
	maken      bool
	toonFout   bool
}

type sudokuMsg struct{ s sudoku }

func (m *model) sudokuOpen() tea.Cmd {
	st := m.dk.sd
	if st.s == nil {
		st.s = laadSudoku()
	}
	if st.s == nil {
		st.kiesNiveau = true
	}
	return nil
}

func (m *model) toetsSudoku(msg tea.KeyMsg) (tea.Cmd, bool) {
	st := m.dk.sd
	k := msg.String()
	if st.maken {
		return nil, true
	}
	if st.kiesNiveau {
		switch k {
		case "up", "k":
			st.niveauSel = max(st.niveauSel-1, 0)
		case "down", "j":
			st.niveauSel = min(st.niveauSel+1, 2)
		case "enter":
			st.kiesNiveau, st.maken = false, true
			niveau := st.niveauSel
			return func() tea.Msg { return sudokuMsg{maakSudoku(niveau, rand.New(rand.NewSource(rand.Int63())))} }, true
		case "esc":
			if st.s != nil {
				st.kiesNiveau = false
			} else {
				m.dk.scherm = denkMenu
			}
		}
		return nil, true
	}
	s := st.s
	switch {
	case k == "esc":
		m.dk.scherm = denkMenu
	case k == "up" || k == "k":
		st.cel = (st.cel + 72) % 81
	case k == "down" || k == "j":
		st.cel = (st.cel + 9) % 81
	case k == "left":
		st.cel = st.cel/9*9 + (st.cel%9+8)%9
	case k == "right":
		st.cel = st.cel/9*9 + (st.cel%9+1)%9
	case len(k) == 1 && k[0] >= '1' && k[0] <= '9':
		if s.Puzzel[st.cel] == 0 && !s.klaar() {
			s.Ingevuld[st.cel] = int(k[0] - '0')
			st.toonFout = false
			if s.klaar() {
				m.zeg(m.t("sudoku.solved"))
			}
			s.bewaar()
		}
	case k == "0" || k == "backspace" || k == "delete" || k == " ":
		if s.Puzzel[st.cel] == 0 && !s.klaar() {
			s.Ingevuld[st.cel] = 0
			s.bewaar()
		}
	case k == "c":
		st.toonFout = !st.toonFout
	case k == "h":
		// hint: het gekozen vakje, of anders het eerste lege of foute vakje
		cel := st.cel
		if s.Puzzel[cel] != 0 || s.Ingevuld[cel] == s.Oplossing[cel] {
			for i := 0; i < 81; i++ {
				if s.Ingevuld[i] != s.Oplossing[i] {
					cel = i
					break
				}
			}
		}
		if s.Ingevuld[cel] != s.Oplossing[cel] {
			s.Ingevuld[cel] = s.Oplossing[cel]
			s.Hints++
			st.cel = cel
			if s.klaar() {
				m.zeg(m.t("sudoku.solved"))
			}
			s.bewaar()
		}
	case k == "n":
		st.kiesNiveau, st.niveauSel = true, s.Niveau
	default:
		return nil, false
	}
	return nil, true
}

func (m *model) sudokuBericht(msg sudokuMsg) {
	st := m.dk.sd
	st.maken = false
	s := msg.s
	st.s, st.cel, st.toonFout = &s, 40, false
	s.bewaar()
}

func (m model) viewSudoku(p palet) string {
	st := m.dk.sd
	dim := lipgloss.NewStyle().Foreground(p.dim)
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.puzzles")) + "  " + dim.Render(m.t("sudoku.title"))
	if st.maken {
		return kop + "\n\n" + dim.Render(m.t("sudoku.making"))
	}
	if st.kiesNiveau {
		var r []string
		for i := 0; i < 3; i++ {
			naam := m.t([]string{"sudoku.easy", "sudoku.medium", "sudoku.hard"}[i])
			if i == st.niveauSel {
				r = append(r, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸ "+naam+" "))
			} else {
				r = append(r, "  "+naam)
			}
		}
		return kop + "\n\n" + paneel(p, m.t("sudoku.pick"), strings.Join(r, "\n"), 40, true) + "\n" + dim.Render(m.t("buddy.start"))
	}
	s := st.s
	gekozen := s.Ingevuld[st.cel]
	lijn := lipgloss.NewStyle().Foreground(p.lijn)
	var b strings.Builder
	rand_ := func(l, mid, r string) string {
		seg := strings.Repeat("─", 9)
		return lijn.Render(l + seg + mid + seg + mid + seg + r)
	}
	b.WriteString(rand_("┌", "┬", "┐") + "\n")
	for rij := 0; rij < 9; rij++ {
		if rij == 3 || rij == 6 {
			b.WriteString(rand_("├", "┼", "┤") + "\n")
		}
		b.WriteString(lijn.Render("│"))
		for kol := 0; kol < 9; kol++ {
			cel := rij*9 + kol
			c := s.Ingevuld[cel]
			tekst := " · "
			stijl := lipgloss.NewStyle().Foreground(p.dim)
			if c != 0 {
				tekst = " " + string(rune('0'+c)) + " "
				stijl = lipgloss.NewStyle().Foreground(p.acc)
				if s.Puzzel[cel] != 0 {
					stijl = lipgloss.NewStyle().Foreground(p.fg).Bold(true)
				}
			}
			if c != 0 && c == gekozen && cel != st.cel {
				stijl = stijl.Background(p.bg2).Underline(true) // hetzelfde cijfer elders
			}
			if st.toonFout && c != 0 && s.Puzzel[cel] == 0 && c != s.Oplossing[cel] {
				stijl = stijl.Foreground(p.rood).Bold(true)
			}
			if cel == st.cel {
				stijl = stijl.Background(p.acc).Foreground(p.bg).Bold(true)
			}
			b.WriteString(stijl.Render(tekst))
			if kol == 2 || kol == 5 || kol == 8 {
				b.WriteString(lijn.Render("│"))
			}
		}
		b.WriteString("\n")
	}
	b.WriteString(rand_("└", "┴", "┘"))
	niveau := m.t([]string{"sudoku.easy", "sudoku.medium", "sudoku.hard"}[s.Niveau])
	info := []string{dim.Render(niveau)}
	leeg := 0
	for i := 0; i < 81; i++ {
		if s.Ingevuld[i] == 0 {
			leeg++
		}
	}
	switch {
	case s.klaar():
		info = append(info, "", lipgloss.NewStyle().Foreground(p.groen).Bold(true).Render(m.t("sudoku.solved")))
	case st.toonFout:
		fout := 0
		for i := 0; i < 81; i++ {
			if s.Ingevuld[i] != 0 && s.Ingevuld[i] != s.Oplossing[i] {
				fout++
			}
		}
		if fout == 0 {
			info = append(info, "", lipgloss.NewStyle().Foreground(p.groen).Render(m.t("sudoku.allgood", leeg)))
		} else {
			info = append(info, "", lipgloss.NewStyle().Foreground(p.rood).Render(m.t("sudoku.wrong", fout)))
		}
	default:
		info = append(info, "", dim.Render(m.t("sudoku.empty", leeg)))
	}
	return kop + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, b.String(), "    ", strings.Join(info, "\n")) + "\n\n" + dim.Render(m.t("sudoku.keys"))
}
