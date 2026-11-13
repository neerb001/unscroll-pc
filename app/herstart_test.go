package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNieuweVersieKlaar(t *testing.T) {
	oudPad, oudeVersie := geinstalleerdPad, versie
	t.Cleanup(func() { geinstalleerdPad, versie = oudPad, oudeVersie })
	dir := t.TempDir()
	geinstalleerdPad = filepath.Join(dir, "unscroll")

	versie = "v0.1.0"
	if v := nieuweVersieKlaar(); v != "" {
		t.Errorf("zonder geïnstalleerd programma: %q", v)
	}
	nieuw := filepath.Join(dir, "releases", "v0.2.0", "unscroll")
	_ = os.MkdirAll(filepath.Dir(nieuw), 0o755)
	_ = os.WriteFile(nieuw, []byte("#!/bin/sh\n"), 0o755)
	if err := os.Symlink(nieuw, geinstalleerdPad); err != nil {
		t.Fatal(err)
	}
	if v := nieuweVersieKlaar(); v != "v0.2.0" {
		t.Errorf("nieuwe versie niet gezien: %q", v)
	}
	versie = "dev"
	if v := nieuweVersieKlaar(); v != "" {
		t.Errorf("een ontwikkelversie mag nooit herstarten: %q", v)
	}
	versie = "v0.1.0"
	zelf, _ := os.Executable()
	_ = os.Remove(geinstalleerdPad)
	_ = os.Symlink(zelf, geinstalleerdPad)
	if v := nieuweVersieKlaar(); v != "" {
		t.Errorf("draait al de geïnstalleerde versie: %q", v)
	}
}

// stopt geeft true als het commando (een reeks) uiteindelijk tea.Quit oplevert.
func stopt(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); ok {
		return true
	}
	v := reflect.ValueOf(msg)
	if v.Kind() == reflect.Slice && strings.Contains(v.Type().Name(), "sequence") {
		for i := 0; i < v.Len(); i++ {
			if c, ok := v.Index(i).Interface().(tea.Cmd); ok && c != nil {
				if _, ok := c().(tea.QuitMsg); ok {
					return true
				}
			}
		}
	}
	return false
}

func TestHerstartOpRustigMoment(t *testing.T) {
	oud := nieuweVersieKlaar
	t.Cleanup(func() { nieuweVersieKlaar = oud })
	klaar := "v9.9.9"
	nieuweVersieKlaar = func() string { return klaar }

	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	nacht := time.Date(2026, 11, 14, 3, 10, 0, 0, time.Local)
	tik := func(m model, t time.Time) (model, tea.Cmd) {
		u, cmd := m.Update(tickMsg(t))
		return u.(model), cmd
	}

	// midden op de dag: nooit
	m.laatsteToets = nacht.Add(-12 * time.Hour)
	if m2, cmd := tik(m, nacht.Add(-12*time.Hour+30*time.Minute)); stopt(cmd) || m2.nieuw != klaar {
		t.Errorf("herstart overdag, of nieuwe versie niet onthouden (%q)", m2.nieuw)
	}
	// 's nachts maar net nog iemand aan het typen
	m.laatsteToets = nacht.Add(-5 * time.Minute)
	if _, cmd := tik(m, nacht); stopt(cmd) {
		t.Error("herstart terwijl er net nog getypt is")
	}
	// 's nachts, maar de ouderstand staat aan (die gaat na 10 minuten stilte vanzelf uit)
	m.laatsteToets = nacht.Add(-time.Hour)
	m.nu = nacht
	m.ouderModus = true
	if m.rustigMoment() {
		t.Error("rustig moment terwijl de ouderstand aan staat")
	}
	m.ouderModus = false
	// 's nachts op een scherm met misschien niet bewaard werk
	m.tab = tabSchrijfhoek
	if _, cmd := tik(m, nacht); stopt(cmd) {
		t.Error("herstart in de schrijfhoek")
	}
	m.tab = tabStart
	// er speelt nog iets
	m.l.nu = &afspelen{soort: soortSpNummer}
	if _, cmd := tik(m, nacht); stopt(cmd) {
		t.Error("herstart terwijl er muziek speelt")
	}
	m.l.nu.stand.Pauze = true
	m.l.spotify = nil
	m.l.nu = nil
	// rustig: herstarten
	if _, cmd := tik(m, nacht); !stopt(cmd) {
		t.Error("geen herstart op een rustig moment")
	}
	// avondslot dicht, ook vroeg op de avond
	m.ouders.Avondslot.Schooldagen, m.ouders.Avondslot.Weekend, m.ouders.Avondslot.Open = "20:00", "20:00", "07:00"
	avond := time.Date(2026, 11, 11, 21, 30, 0, 0, time.Local)
	m.laatsteToets = avond.Add(-20 * time.Minute)
	if _, cmd := tik(m, avond); !stopt(cmd) {
		t.Error("geen herstart tijdens het avondslot")
	}
	// geen nieuwe versie: niets
	klaar = ""
	if _, cmd := tik(m, nacht); stopt(cmd) {
		t.Error("herstart zonder nieuwe versie")
	}
}

func TestHerstartInOuderstand(t *testing.T) {
	oud := nieuweVersieKlaar
	t.Cleanup(func() { nieuweVersieKlaar = oud })
	nieuweVersieKlaar = func() string { return "v9.9.9" }
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.nieuw = "v9.9.9"
	m.tab = tabSysteem
	if v := m.View(); !strings.Contains(v, "v9.9.9 staat klaar") {
		t.Errorf("systeemscherm toont de nieuwe versie niet:\n%s", v)
	}
	m.ouderModus = true
	if v := m.View(); !strings.Contains(v, "[R] nu herstarten") {
		t.Errorf("ouderstand toont de herstartknop niet:\n%s", v)
	}
	u, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	if !stopt(cmd) {
		t.Error("R in de ouderstand herstart niet")
	}
	m = u.(model)
	m.ouderModus = false
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")}); stopt(cmd) {
		t.Error("R zonder ouderstand mag niet herstarten")
	}
}
