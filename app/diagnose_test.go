package main

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDiagnose(t *testing.T) {
	nav := nepNavidrome(t)
	defer nav.Close()
	var gesloten []string
	abs := nepABS(t, &gesloten)
	defer abs.Close()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("UNSCROLL_HEDY", t.TempDir())
	t.Setenv("PATH", t.TempDir()) // geen Stockfish, geen python3

	m, _ := testModel(t, Geheimen{"NAVIDROME_URL": nav.URL, "NAVIDROME_GEBRUIKER": "kind", "NAVIDROME_WACHTWOORD": "geheim",
		"ABS_URL": abs.URL, "ABS_TOKEN": "sleutel"})
	ouderTestOmgeving(t)
	m.ouders, m.oudersFout = laadOuders()
	m.inst.Taal = "nl"
	m.breed, m.hoog = 110, 40
	schrijfVerslag(5 * time.Minute) // alsof de app ooit vastzat

	// zonder ouderstand doet d niets
	m.tab = tabSysteem
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}); cmd != nil && m.diag.open {
		t.Fatal("diagnose open zonder ouderstand")
	}
	m.ouderModus = true
	u, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = u.(model)
	if !m.diag.open || cmd == nil {
		t.Fatal("d opent de diagnose niet")
	}
	begin := time.Now()
	msg := cmd()
	if time.Since(begin) > 35*time.Second {
		t.Errorf("diagnose duurt te lang: %v", time.Since(begin))
	}
	u, _ = m.Update(msg)
	m = u.(model)

	regels := map[string]diagRegel{}
	for _, r := range m.diag.regels {
		regels[r.sleutel] = r
	}
	for _, k := range diagVolgorde {
		if _, ok := regels[k]; !ok {
			t.Errorf("onderdeel %s ontbreekt", k)
		}
	}
	verwacht := map[string]struct {
		status int
		tekst  string
	}{
		"abs":       {diagGoed, "2 bibliotheken"},
		"navidrome": {diagFout, "not found"}, // de nepserver kent ping niet: de precieze fout moet erin staan
		"parents":   {diagGoed, "avondslot vandaag"},
		"storage":   {diagGoed, "bewaren lukt"},
		"watchdog":  {diagLet, "vast"},
		"hedy":      {diagFout, "niet geïnstalleerd"},
		"spotify":   {diagUit, "niet ingesteld"},
		"ai":        {diagUit, ""},
	}
	for k, v := range verwacht {
		r := regels[k]
		if r.status != v.status || !strings.Contains(r.tekst, v.tekst) {
			t.Errorf("%s: status %d %q, verwacht %d met %q", k, r.status, r.tekst, v.status, v.tekst)
		}
	}
	v := m.View()
	for _, s := range []string{"Diagnose", "Audiobookshelf", "✗", "not found", "w verslag"} {
		if !strings.Contains(v, s) {
			t.Errorf("scherm mist %q:\n%s", s, v)
		}
	}
	// verslag van de waakhond lezen en weggooien
	m = toets(m, "w")
	if v := m.View(); !strings.Contains(v, "zat vast") || !strings.Contains(v, "goroutine") {
		t.Errorf("verslag niet te lezen:\n%s", v)
	}
	u, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = u.(model)
	if _, err := os.Stat(vastgelopenPad()); err == nil {
		t.Error("x gooit het verslag niet weg")
	}
	if cmd == nil {
		t.Error("na weggooien hoort de diagnose opnieuw te lopen")
	}
	m = toets(m, "esc")
	if m.diag.open {
		t.Error("esc sluit de diagnose niet")
	}
}
