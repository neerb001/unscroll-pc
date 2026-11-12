package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Programmeren met Hedy (https://hedy.org): de echte Hedy-taal, niveau voor niveau, met de
// opdrachten van hedy.org. Het pakket (Hedy, opdrachten, foutmeldingen) installeert de updater
// in /opt/unscroll/hedy. Een programma draait als apart Python-proces met grenzen.

type hedyBlok struct {
	Tekst string `json:"tekst"`
	Code  string `json:"code"`
}

type hedyOpdracht struct {
	Sleutel string     `json:"sleutel"`
	Naam    string     `json:"naam"`
	Blokken []hedyBlok `json:"blokken"`
}

type hedyStaat struct {
	inhoud           map[string]map[string][]hedyOpdracht
	fout             string
	niveau, opdracht int
	editor           textarea.Model
	inEditor         bool
	scroll           int
	uitvoer          []string
	proc             *hedyProces
	invoer           string
	volgID           int
}

func hedyMap() string {
	if p := os.Getenv("UNSCROLL_HEDY"); p != "" {
		return p
	}
	return "/opt/unscroll/hedy"
}

func nieuweHedy() *hedyStaat {
	h := &hedyStaat{niveau: 1}
	h.editor = textarea.New()
	h.editor.ShowLineNumbers = true
	h.editor.CharLimit = 5000
	h.editor.Prompt = ""
	return h
}

func (h *hedyStaat) laad() {
	if h.inhoud != nil {
		return
	}
	data, err := os.ReadFile(filepath.Join(hedyMap(), "inhoud.json"))
	if err != nil {
		h.fout = "notinstalled"
		return
	}
	if err := json.Unmarshal(data, &h.inhoud); err != nil {
		h.fout = err.Error()
	}
}

func (h *hedyStaat) opdrachten(taal string) []hedyOpdracht {
	t := h.inhoud[taal]
	if t == nil {
		t = h.inhoud["en"]
	}
	return t[fmt.Sprint(h.niveau)]
}

func (h *hedyStaat) aantalNiveaus(taal string) int {
	n := len(h.inhoud[taal])
	if n == 0 {
		n = len(h.inhoud["en"])
	}
	return max(n, 1)
}

func (h *hedyStaat) huidige(taal string) (hedyOpdracht, bool) {
	o := h.opdrachten(taal)
	if h.opdracht < 0 || h.opdracht >= len(o) {
		return hedyOpdracht{}, false
	}
	return o[h.opdracht], true
}

/* ---------------- bewaren ---------------- */

func hedyCodePad(niveau int, sleutel string) string {
	return filepath.Join(unscrollData("hedy"), fmt.Sprintf("%02d-%s.hedy", niveau, sleutel))
}

func (m *model) hedyBewaar() {
	h := m.od.hedy
	if o, ok := h.huidige(m.inst.Taal); ok {
		_ = os.MkdirAll(unscrollData("hedy"), 0o700)
		_ = os.WriteFile(hedyCodePad(h.niveau, o.Sleutel), []byte(h.editor.Value()), 0o600)
	}
}

func (m *model) hedyLaadCode() {
	h := m.od.hedy
	h.editor.SetValue("")
	h.uitvoer, h.scroll = nil, 0
	if o, ok := h.huidige(m.inst.Taal); ok {
		if data, err := os.ReadFile(hedyCodePad(h.niveau, o.Sleutel)); err == nil {
			h.editor.SetValue(string(data))
		}
	}
}

/* ---------------- uitvoeren ---------------- */

type hedyProces struct {
	id    int
	cmd   *exec.Cmd
	stdin io.WriteCloser
	uit   chan tea.Msg
	once  sync.Once
}

type hedyUitMsg struct {
	id    int
	tekst string
}

type hedyKlaarMsg struct {
	id  int
	err error
}

const hedyFout = "\x1eHEDYFOUT "

func (p *hedyProces) volgende() tea.Cmd {
	return func() tea.Msg { return <-p.uit }
}

func (p *hedyProces) stop() {
	p.once.Do(func() {
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
	})
}

func (m *model) hedyStart() tea.Cmd {
	h := m.od.hedy
	if h.proc != nil {
		return nil
	}
	m.hedyBewaar()
	code := h.editor.Value()
	if strings.TrimSpace(code) == "" {
		return nil
	}
	tmp, err := os.CreateTemp("", "unscroll-*.hedy")
	if err != nil {
		h.uitvoer = []string{"! " + err.Error()}
		return nil
	}
	_, _ = tmp.WriteString(code)
	tmp.Close()
	taal := m.inst.Taal
	if taal != "nl" {
		taal = "en"
	}
	cmd := exec.Command("python3", "-I", filepath.Join(hedyMap(), "unscroll_hedy.py"), fmt.Sprint(h.niveau), taal, tmp.Name())
	cmd.Dir = os.TempDir()
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + os.TempDir(), "LANG=C.UTF-8", "PYTHONIOENCODING=utf-8", "PYTHONUNBUFFERED=1"}
	stopMetOuder(cmd)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	h.volgID++
	p := &hedyProces{id: h.volgID, cmd: cmd, stdin: stdin, uit: make(chan tea.Msg, 64)}
	if err := cmd.Start(); err != nil {
		os.Remove(tmp.Name())
		h.uitvoer = []string{"! " + m.t("hedy.nopython")}
		return nil
	}
	h.proc, h.uitvoer, h.invoer, h.scroll = p, nil, "", 0
	go func() {
		r := bufio.NewReader(stdout)
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				p.uit <- hedyUitMsg{p.id, string(buf[:n])}
			}
			if err != nil {
				break
			}
		}
		err := cmd.Wait()
		os.Remove(tmp.Name())
		p.uit <- hedyKlaarMsg{p.id, err}
	}()
	// een programma mag hooguit tien minuten lopen (wachten op een antwoord meegerekend)
	go func() {
		time.Sleep(10 * time.Minute)
		p.stop()
	}()
	return p.volgende()
}

// hedyBericht verwerkt uitvoer van het programma.
func (m *model) hedyBericht(msg tea.Msg) tea.Cmd {
	h := m.od.hedy
	p := h.proc
	switch msg := msg.(type) {
	case hedyUitMsg:
		if p == nil || msg.id != p.id {
			return nil
		}
		tekst := msg.tekst
		if len(h.uitvoer) == 0 {
			h.uitvoer = []string{""}
		}
		for i, deel := range strings.Split(tekst, "\n") {
			if i > 0 {
				h.uitvoer = append(h.uitvoer, "")
			}
			h.uitvoer[len(h.uitvoer)-1] += deel
		}
		if len(h.uitvoer) > 2000 {
			h.uitvoer = h.uitvoer[len(h.uitvoer)-2000:]
		}
		h.scroll = 0
		return p.volgende()
	case hedyKlaarMsg:
		if p == nil || msg.id != p.id {
			return nil
		}
		h.proc = nil
		// een fout van Hedy staat op één regel met een markering
		var schoon []string
		for _, r := range h.uitvoer {
			if i := strings.Index(r, hedyFout); i >= 0 {
				var f struct {
					Tekst string `json:"tekst"`
					Regel any    `json:"regel"`
				}
				_ = json.Unmarshal([]byte(r[i+len(hedyFout):]), &f)
				if r[:i] != "" {
					schoon = append(schoon, r[:i])
				}
				schoon = append(schoon, "", "! "+strings.ReplaceAll(f.Tekst, "`", ""))
				continue
			}
			schoon = append(schoon, r)
		}
		for len(schoon) > 0 && strings.TrimSpace(schoon[len(schoon)-1]) == "" {
			schoon = schoon[:len(schoon)-1]
		}
		h.uitvoer = append(schoon, "", "· "+m.t("hedy.done"))
	}
	return nil
}

/* ---------------- toetsen ---------------- */

func (m *model) toetsHedy(msg tea.KeyMsg) (tea.Cmd, bool) {
	h := m.od.hedy
	k := msg.String()
	taal := m.inst.Taal

	// het programma draait: typen gaat naar het programma (antwoord op een vraag)
	if p := h.proc; p != nil {
		switch k {
		case "esc", "ctrl+c":
			p.stop()
			return nil, true
		case "enter":
			_, _ = io.WriteString(p.stdin, h.invoer+"\n")
			h.uitvoer[len(h.uitvoer)-1] += h.invoer
			h.invoer = ""
		case "backspace":
			if r := []rune(h.invoer); len(r) > 0 {
				h.invoer = string(r[:len(r)-1])
			}
		default:
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				h.invoer += string(msg.Runes)
			}
		}
		return nil, true
	}

	switch k {
	case "ctrl+r", "f5":
		return m.hedyStart(), true
	case "tab":
		h.inEditor = !h.inEditor
		if h.inEditor {
			return h.editor.Focus(), true
		}
		h.editor.Blur()
		return nil, true
	case "esc":
		if h.inEditor {
			h.inEditor = false
			h.editor.Blur()
			return nil, true
		}
		m.hedyBewaar()
		m.od.scherm = ontdekMenu
		return nil, true
	}
	if h.inEditor {
		var cmd tea.Cmd
		h.editor, cmd = h.editor.Update(msg)
		return cmd, true
	}
	switch k {
	case "up", "k":
		if h.opdracht > 0 {
			m.hedyBewaar()
			h.opdracht--
			m.hedyLaadCode()
		}
	case "down", "j":
		if h.opdracht < len(h.opdrachten(taal))-1 {
			m.hedyBewaar()
			h.opdracht++
			m.hedyLaadCode()
		}
	case "left", "right":
		n := h.niveau
		if k == "left" {
			n = max(n-1, 1)
		} else {
			n = min(n+1, h.aantalNiveaus(taal))
		}
		if n != h.niveau {
			m.hedyBewaar()
			h.niveau, h.opdracht = n, 0
			m.hedyLaadCode()
		}
	case "pgdown", " ":
		h.scroll += 5
	case "pgup":
		h.scroll = max(h.scroll-5, 0)
	case "v", "c":
		// voorbeeld in de editor zetten, zoals de gele knop op hedy.org
		if o, ok := h.huidige(taal); ok {
			for _, b := range o.Blokken {
				if b.Code != "" {
					h.editor.SetValue(b.Code)
					h.inEditor = true
					return h.editor.Focus(), true
				}
			}
		}
	case "enter":
		h.inEditor = true
		return h.editor.Focus(), true
	default:
		return nil, false
	}
	return nil, true
}

/* ---------------- weergave ---------------- */

func (m model) viewHedy(p palet) string {
	h := m.od.hedy
	dim := lipgloss.NewStyle().Foreground(p.dim)
	breed := m.breed - 6
	if h.fout != "" {
		tekst := h.fout
		if tekst == "notinstalled" {
			tekst = m.t("hedy.notinstalled")
		}
		return lipgloss.NewStyle().Bold(true).Render(m.t("disc.code")) + "\n\n" +
			paneel(p, "Hedy", lipgloss.NewStyle().Foreground(p.geel).Width(breed-6).Render(tekst), breed, true)
	}
	taal := m.inst.Taal
	o, _ := h.huidige(taal)
	alle := h.opdrachten(taal)

	// opdracht: tekst met voorbeeldcode
	var tekst []string
	for _, b := range o.Blokken {
		if b.Code != "" {
			for _, r := range strings.Split(b.Code, "\n") {
				tekst = append(tekst, lipgloss.NewStyle().Foreground(p.groen).Render("  "+r))
			}
		} else {
			tekst = append(tekst, strings.Split(lipgloss.NewStyle().Width(breed-6).Render(b.Tekst), "\n")...)
		}
		tekst = append(tekst, "")
	}
	hoogteOpdracht := max((m.hoog-8)*2/5, 4)
	h.scroll = min(h.scroll, max(len(tekst)-hoogteOpdracht, 0))
	tekst = tekst[h.scroll:min(h.scroll+hoogteOpdracht, len(tekst))]
	titel := fmt.Sprintf("%s %d · %s (%d/%d)", m.t("hedy.level"), h.niveau, o.Naam, h.opdracht+1, len(alle))

	// editor en uitvoer naast elkaar
	hoogteOnder := max(m.hoog-8-hoogteOpdracht-6, 4)
	// een paneel is 2 tekens breder dan zijn breedte (de rand): links + 2 + 1 + rechts + 2 = breed
	links := (breed - 5) / 2
	rechts := breed - links - 5
	h.editor.SetWidth(links - 4)
	h.editor.SetHeight(hoogteOnder)
	uit := h.uitvoer
	if h.proc != nil {
		uit = append(append([]string(nil), uit...), "")
		uit[len(uit)-1] = ""
		if n := len(h.uitvoer); n > 0 {
			uit = append(append([]string(nil), h.uitvoer[:n-1]...), h.uitvoer[n-1]+lipgloss.NewStyle().Foreground(p.acc).Render(h.invoer+"▌"))
		}
	}
	var uitRegels []string
	for _, r := range uit {
		if strings.HasPrefix(r, "! ") {
			r = lipgloss.NewStyle().Foreground(p.geel).Render(r[2:])
		} else if strings.HasPrefix(r, "· ") {
			r = dim.Render(r[2:])
		}
		uitRegels = append(uitRegels, strings.Split(lipgloss.NewStyle().Width(rechts-2).Render(r), "\n")...)
	}
	if len(uitRegels) > hoogteOnder {
		uitRegels = uitRegels[len(uitRegels)-hoogteOnder:]
	}
	for len(uitRegels) < hoogteOnder { // even hoog als de editor
		uitRegels = append(uitRegels, "")
	}
	uitTitel := m.t("hedy.output")
	if h.proc != nil {
		uitTitel = m.t("hedy.running")
	}
	onder := lipgloss.JoinHorizontal(lipgloss.Top,
		paneel(p, m.t("hedy.code"), h.editor.View(), links, h.inEditor),
		" ",
		paneel(p, uitTitel, strings.Join(uitRegels, "\n"), rechts, h.proc != nil))
	toetsen := m.t("hedy.keys")
	if h.inEditor {
		toetsen = m.t("hedy.editkeys")
	}
	if h.proc != nil {
		toetsen = m.t("hedy.runkeys")
	}
	return paneel(p, titel, strings.Join(tekst, "\n"), breed, !h.inEditor && h.proc == nil) + "\n" + onder + "\n" + dim.Render(toetsen)
}
