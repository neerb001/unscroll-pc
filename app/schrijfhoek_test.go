package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTakenlijst(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.sh = nieuweSchrijfhoek()
	m.inst.Taal = "nl"
	m = toets(m, "6")
	m = toets(m, "enter") // Takenlijst
	if !strings.Contains(m.View(), "Nog geen taken") {
		t.Fatalf("lege lijst:\n%s", m.View())
	}
	for _, tekst := range []string{"gymspullen inpakken", "boek terugbrengen", "opa bellen"} {
		m = toets(m, "n")
		m = toets(m, tekst)
		m = toets(m, "enter")
	}
	if len(m.sh.taken) != 3 || !strings.Contains(m.View(), "3 open") {
		t.Fatalf("taken niet toegevoegd:\n%s", m.View())
	}
	m = toets(m, "up")
	m = toets(m, "up") // gymspullen
	m = toets(m, " ")  // afvinken: zakt naar onderen
	if !m.sh.taken[2].Klaar || m.sh.taken[2].Tekst != "gymspullen inpakken" || !strings.Contains(m.View(), "2 open") {
		t.Fatalf("afvinken: %+v", m.sh.taken)
	}
	m = toets(m, "up")
	m = toets(m, "up") // boek terugbrengen
	m = toets(m, "e")
	m = toets(m, " voor vrijdag")
	m = toets(m, "enter")
	if m.sh.taken[0].Tekst != "boek terugbrengen voor vrijdag" {
		t.Fatalf("aanpassen: %+v", m.sh.taken)
	}
	m = toets(m, "delete") // eerst bevestigen
	if len(m.sh.taken) != 3 || !strings.Contains(m.View(), "nog een keer Delete") {
		t.Fatal("weghalen hoort eerst om bevestiging te vragen")
	}
	m = toets(m, "delete")
	if len(m.sh.taken) != 2 {
		t.Fatalf("weghalen: %+v", m.sh.taken)
	}
	m = toets(m, "o") // afgevinkte opruimen
	if len(m.sh.taken) != 1 || m.sh.taken[0].Tekst != "opa bellen" {
		t.Fatalf("opruimen: %+v", m.sh.taken)
	}
	// bewaard, en te zien op het startscherm
	if len(laadTaken()) != 1 {
		t.Error("taken niet bewaard")
	}
	m = toets(m, "0")
	if !strings.Contains(m.View(), "opa bellen") {
		t.Errorf("open taak niet op het startscherm:\n%s", m.View())
	}
}

func TestVerhalenTeruglezen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.MkdirAll(verhalenMap(), 0o700)
	_ = os.WriteFile(filepath.Join(verhalenMap(), "2026-11-10-1600.txt"), []byte("Er was eens een robot die kon vliegen.\n\nHij vloog naar de maan.\n"), 0o600)
	m, _ := testModel(t, Geheimen{})
	m.sh = nieuweSchrijfhoek()
	m.inst.Taal = "nl"
	m = toets(m, "6")
	m = toets(m, "down")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "robot die kon vliegen") {
		t.Fatalf("verhalenlijst:\n%s", m.View())
	}
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "Hij vloog naar de maan") {
		t.Fatalf("verhaal lezen:\n%s", m.View())
	}
}
