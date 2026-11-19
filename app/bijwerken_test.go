package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// De root-kant: zonder de goede pincode start de updater niet.
func TestOuderUpdatePincode(t *testing.T) {
	ouderTestOmgeving(t)
	dir := t.TempDir()
	merk := filepath.Join(dir, "gedraaid")
	script := filepath.Join(dir, "unscroll-update")
	_ = os.WriteFile(script, []byte("#!/bin/sh\necho \"$1\" > "+merk+"\necho 'Already up to date (v9.9.9)'\n"), 0o755)
	oud := updaterPad
	updaterPad = script
	defer func() { updaterPad = oud }()

	var uit bytes.Buffer
	if err := ouderUpdate(strings.NewReader(`{"pin":"0000"}`), &uit); err == nil {
		t.Fatal("een verkeerde pincode hoort geweigerd te worden")
	}
	if _, err := os.Stat(merk); err == nil {
		t.Fatal("de updater draaide met een verkeerde pincode")
	}
	if err := ouderUpdate(strings.NewReader(`{"pin":"4321"}`), &uit); err != nil {
		t.Fatal(err)
	}
	if arg, _ := os.ReadFile(merk); strings.TrimSpace(string(arg)) != "--nu" || !strings.Contains(uit.String(), "Already up to date") {
		t.Errorf("updater niet goed gestart: arg %q, uitvoer %q", arg, uit.String())
	}
}

func TestNuBijwerkenInDeOuderstand(t *testing.T) {
	var verzoeken []string
	antwoord, fout := "", error(nil)
	oud := ouderUpdateUitvoeren
	ouderUpdateUitvoeren = func(v []byte) ([]byte, error) {
		verzoeken = append(verzoeken, string(v))
		return []byte(antwoord), fout
	}
	defer func() { ouderUpdateUitvoeren = oud }()
	m, _ := testModel(t, Geheimen{})
	ouderTestOmgeving(t)
	m.ouders, _ = laadOuders()
	m.inst.Taal = "nl"
	m.breed, m.hoog = 120, 40
	m = toets(m, "8")

	// zonder ouderstand gebeurt er niets
	m = toets(m, "U")
	if len(verzoeken) != 0 {
		t.Fatal("bijwerken zonder ouderstand")
	}

	m = toets(m, "o")
	m = toets(m, "4321")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "[U] nu bijwerken") {
		t.Fatalf("de toets staat niet in beeld:\n%s", m.View())
	}

	// al de nieuwste
	antwoord = "Scripts updated.\nAlready up to date (v0.25.3)"
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("U")})
	m = nm.(model)
	if !m.bijwerken || !strings.Contains(m.View(), "bezig met bijwerken") {
		t.Fatalf("je ziet niet dat hij bezig is:\n%s", m.View())
	}
	if again := m.nuBijwerken(); again != nil {
		t.Error("twee keer tegelijk bijwerken hoort niet te kunnen")
	}
	m = voer(m, cmd)
	if m.bijwerken || !strings.Contains(m.melding, "al de nieuwste versie") {
		t.Errorf("melding: %q", m.melding)
	}
	if len(verzoeken) != 1 || !strings.Contains(verzoeken[0], `"pin":"4321"`) {
		t.Errorf("verzoek: %v", verzoeken)
	}

	// nieuwe versie gezet: dan meteen kunnen herstarten
	oudKlaar := nieuweVersieKlaar
	nieuweVersieKlaar = func() string { return "v0.26.0" }
	antwoord = "Hedy updated.\nUpdated to v0.26.0. Active after the next restart of the app."
	m = toets(m, "U")
	nieuweVersieKlaar = oudKlaar
	if !strings.Contains(m.melding, "v0.26.0") || !strings.Contains(m.melding, "R") {
		t.Errorf("melding na bijwerken: %q", m.melding)
	}

	// de sudo-regel ontbreekt nog op dit apparaat
	antwoord, fout = "sudo: a password is required", errors.New("exit status 1")
	m = toets(m, "U")
	if !strings.Contains(m.melding, "volgende automatische update") {
		t.Errorf("melding zonder sudo-regel: %q", m.melding)
	}

	// geen internet
	antwoord, fout = "GitHub not reachable", nil
	m = toets(m, "U")
	if !strings.Contains(m.melding, "niet bereikbaar") {
		t.Errorf("melding zonder internet: %q", m.melding)
	}

	// iets anders mis: de laatste regel van de updater
	antwoord, fout = "Checksum mismatch, update aborted", errors.New("exit status 1")
	m = toets(m, "U")
	if !strings.Contains(m.melding, "lukte niet") || !strings.Contains(m.melding, "Checksum mismatch") {
		t.Errorf("melding bij een fout: %q", m.melding)
	}
}
