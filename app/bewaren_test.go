package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVeiligSchrijven(t *testing.T) {
	dir := t.TempDir()
	pad := filepath.Join(dir, "taken.json")
	if err := veiligSchrijven(pad, []byte("oud"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := veiligSchrijven(pad, []byte("nieuw"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(pad); string(data) != "nieuw" {
		t.Errorf("inhoud: %q", data)
	}
	if info, _ := os.Stat(pad); info.Mode().Perm() != 0o600 {
		t.Errorf("rechten: %v", info.Mode().Perm())
	}
	if l, _ := os.ReadDir(dir); len(l) != 1 {
		t.Errorf("tijdelijke bestanden blijven liggen: %v", l)
	}
	// een map die niet bestaat: een fout, en het oude bestand blijft heel
	if err := veiligSchrijven(filepath.Join(dir, "weg", "x.json"), []byte("x"), 0o644); err == nil {
		t.Error("geen fout bij een map die niet bestaat")
	}
	log := filepath.Join(dir, "dagboek.jsonl")
	_ = veiligToevoegen(log, []byte("een\n"), 0o600)
	_ = veiligToevoegen(log, []byte("twee\n"), 0o600)
	if data, _ := os.ReadFile(log); string(data) != "een\ntwee\n" {
		t.Errorf("toevoegen: %q", data)
	}
}
