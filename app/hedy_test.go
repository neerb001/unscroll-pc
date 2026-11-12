package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// nepHedy: een klein script dat doet alsof het Hedy is.
const nepHedyScript = `import sys, json
code = open(sys.argv[3]).read()
if "toets" in code:
    print("\x1eHEDYWIS", flush=True)
    print("\x1eHEDYTOETS", flush=True)
    k = input()
    print("je drukte " + k)
    sys.exit(0)
if "fout" in code:
    print("\x1eHEDYFOUT " + json.dumps({"tekst": "Het lijkt erop dat je een fout maakte op regel 1.", "regel": 1}))
    sys.exit(2)
print("Hoe heet jij? ", end="", flush=True)
naam = input()
print("Hallo " + naam + "! niveau " + sys.argv[1] + " taal " + sys.argv[2])
`

func hedyTestModel(t *testing.T) model {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("geen python3")
	}
	dir := t.TempDir()
	t.Setenv("UNSCROLL_HEDY", dir)
	t.Setenv("HOME", t.TempDir())
	_ = os.WriteFile(filepath.Join(dir, "unscroll_hedy.py"), []byte(nepHedyScript), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "inhoud.json"), []byte(`{"nl":{"1":[{"sleutel":"parrot","naam":"papegaai","blokken":[{"tekst":"Maak je eigen papegaai!"},{"code":"print Ik ben papegaai Hedy\nvraag Wie ben jij?"}]},{"sleutel":"dice","naam":"dobbelsteen","blokken":[{"tekst":"Gooi de dobbelsteen."}]}],"2":[{"sleutel":"rock","naam":"steen","blokken":[{"tekst":"Steen, papier, schaar"}]}]}}`), 0o644)
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 110, 34
	m = toets(m, "4")
	for i := 0; i < 3; i++ {
		m = toets(m, "down")
	}
	return toets(m, "enter")
}

// wachtend: een opdracht die nog loopt (het programma wacht op invoer); die lezen we later verder,
// net als Bubble Tea zelf.
var wachtend chan tea.Msg

// draaiHedy voert de berichten van het programma uit tot het klaar is of op invoer wacht.
func draaiHedy(t *testing.T, m model, cmd tea.Cmd) model {
	for i := 0; i < 50 && (cmd != nil || wachtend != nil); i++ {
		uit := wachtend
		if uit == nil {
			uit = make(chan tea.Msg, 1)
			go func(c tea.Cmd) { uit <- c() }(cmd)
		}
		wachtend = nil
		select {
		case msg := <-uit:
			var nm tea.Model
			nm, cmd = m.Update(msg)
			m = nm.(model)
		case <-time.After(1500 * time.Millisecond):
			wachtend = uit
			return m // wacht op invoer
		}
	}
	return m
}

func TestHedyProgrammeren(t *testing.T) {
	m := hedyTestModel(t)
	if !strings.Contains(m.View(), "Maak je eigen papegaai") || !strings.Contains(m.View(), "Niveau 1") {
		t.Fatalf("opdracht niet te zien:\n%s", m.View())
	}
	m = toets(m, "v") // voorbeeld overnemen
	if !strings.Contains(m.od.hedy.editor.Value(), "vraag Wie ben jij?") {
		t.Fatalf("voorbeeld niet in de editor: %q", m.od.hedy.editor.Value())
	}
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = draaiHedy(t, nm.(model), cmd)
	if m.od.hedy.proc == nil || !strings.Contains(m.View(), "Hoe heet jij?") {
		t.Fatalf("programma vraagt niets:\n%s", m.View())
	}
	m = toets(m, "Sam")
	nm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = draaiHedy(t, nm.(model), cmd)
	if m.od.hedy.proc != nil || !strings.Contains(m.View(), "Hallo Sam! niveau 1 taal nl") {
		t.Fatalf("antwoord kwam niet aan:\n%s", m.View())
	}
	// code is bewaard per opdracht
	if data, _ := os.ReadFile(hedyCodePad(1, "parrot")); !strings.Contains(string(data), "papegaai") {
		t.Error("code niet bewaard")
	}
	// fout van Hedy wordt netjes getoond
	m.od.hedy.editor.SetValue("fout")
	nm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = draaiHedy(t, nm.(model), cmd)
	if !strings.Contains(m.View(), "fout maakte op regel") || strings.Contains(m.View(), "HEDYFOUT") {
		t.Errorf("foutmelding niet netjes:\n%s", m.View())
	}
	// toetsen: het programma wacht op één toets en krijgt hem zonder Enter
	m.od.hedy.editor.SetValue("toets")
	nm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = draaiHedy(t, nm.(model), cmd)
	if !m.od.hedy.wachtToets || !strings.Contains(m.View(), "druk op een toets") {
		t.Fatalf("programma wacht niet op een toets:\n%s", m.View())
	}
	nm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = draaiHedy(t, nm.(model), cmd)
	if !strings.Contains(m.View(), "je drukte x") {
		t.Fatalf("toets kwam niet aan:\n%s", m.View())
	}
	// volgende niveau
	m = toets(m, "esc")
	m = toets(m, "right")
	if m.od.hedy.niveau != 2 || !strings.Contains(m.View(), "Steen, papier") {
		t.Errorf("niveau wisselen:\n%s", m.View())
	}
}

func TestHedyNietGeinstalleerd(t *testing.T) {
	t.Setenv("UNSCROLL_HEDY", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m = toets(m, "4")
	for i := 0; i < 3; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "not installed") {
		t.Errorf("uitleg ontbreekt:\n%s", m.View())
	}
}
