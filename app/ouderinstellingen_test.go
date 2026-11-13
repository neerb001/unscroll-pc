package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPasYAMLAan(t *testing.T) {
	voorbeeld, err := os.ReadFile("../config/ouders.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	uit, err := pasYAMLAan(voorbeeld, []wijziging{
		{"avondslot.schooldagen", "19:45"}, {"max_volume", "55"}, {"maatje.model", "claude-haiku-5-5"},
		{"spotify.goedgekeurd", "+https://open.spotify.com/album/abc123"}, {"spotify.zoeken", "true"},
		{"vandaag.breedtegraad", "52.09"}, {"vandaag.plaats", "Utrecht"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var o Ouders
	if err := yaml.Unmarshal(uit, &o); err != nil {
		t.Fatalf("resultaat niet te lezen: %v\n%s", err, uit)
	}
	if o.Avondslot.Schooldagen != "19:45" || o.MaxVolume != 55 || o.Maatje.Model != "claude-haiku-5-5" || !o.Spotify.Zoeken ||
		len(o.Spotify.Goedgekeurd) != 1 || o.Vandaag.Breedtegraad != 52.09 || o.Vandaag.Plaats != "Utrecht" {
		t.Errorf("waarden: %+v", o)
	}
	if !bytes.Contains(uit, []byte("# Settings for parents")) || !bytes.Contains(uit, []byte("Sunday to Thursday")) {
		t.Errorf("commentaar verdwenen:\n%s", uit)
	}
	uit, err = pasYAMLAan(uit, []wijziging{{"spotify.goedgekeurd", "-0"}})
	if err != nil || bytes.Contains(uit, []byte("abc123")) {
		t.Errorf("link weghalen: %v", err)
	}
	for _, fout := range []wijziging{{"pin_sha256", "x"}, {"avondslot.open", "25:00"}, {"max_volume", "300"}, {"maatje.model", "gpt"},
		{"spotify.goedgekeurd", "+https://example.com"}, {"maatje.aan", "misschien"}} {
		if _, err := pasYAMLAan(voorbeeld, []wijziging{fout}); err == nil {
			t.Errorf("%+v hoort geweigerd te worden", fout)
		}
	}
}

func ouderTestOmgeving(t *testing.T) string {
	pad := filepath.Join(t.TempDir(), "ouders.yaml")
	voorbeeld, _ := os.ReadFile("../config/ouders.example.yaml")
	voorbeeld = bytes.Replace(voorbeeld, []byte(`pin_sha256: ""`), []byte(`pin_sha256: "`+pinHash("4321")+`"`), 1)
	_ = os.WriteFile(pad, voorbeeld, 0o644)
	t.Setenv("UNSCROLL_OUDERS", pad)
	return pad
}

func TestOuderZetPincode(t *testing.T) {
	pad := ouderTestOmgeving(t)
	if err := ouderZet(strings.NewReader(`{"pin":"0000","wijzigingen":[{"sleutel":"max_volume","waarde":"90"}]}`)); err == nil {
		t.Fatal("verkeerde pincode hoort geweigerd te worden")
	}
	if err := ouderZet(strings.NewReader(`{"pin":"4321","wijzigingen":[{"sleutel":"max_volume","waarde":"90"}]}`)); err != nil {
		t.Fatal(err)
	}
	o, _ := laadOuders()
	if o.MaxVolume != 90 {
		t.Errorf("max_volume: %d", o.MaxVolume)
	}
	if _, err := os.Stat(pad + ".nieuw"); err == nil {
		t.Error("tijdelijk bestand blijft staan")
	}
}

func TestOuderInstellingenScherm(t *testing.T) {
	oud := ouderZetUitvoeren
	ouderZetUitvoeren = func(v []byte) ([]byte, error) { return nil, ouderZet(bytes.NewReader(v)) }
	defer func() { ouderZetUitvoeren = oud }()
	m, _ := testModel(t, Geheimen{})
	ouderTestOmgeving(t)
	m.ouders, _ = laadOuders()
	m.inst.Taal = "nl"
	m.breed, m.hoog = 100, 40
	m = toets(m, "8")
	m = toets(m, "o")
	m = toets(m, "4321")
	m = toets(m, "enter")
	if !m.ouderModus {
		t.Fatal("ouderstand niet aan")
	}
	m = toets(m, "i")
	if !strings.Contains(m.View(), "Instellingen voor ouders") {
		t.Fatalf("instellingen:\n%s", m.View())
	}
	m = toets(m, "right") // avondslot schooldagen 20:00 → 20:15
	if m.ouders.Avondslot.Schooldagen != "20:15" {
		t.Errorf("schooldagen na →: %q (%s)", m.ouders.Avondslot.Schooldagen, m.oi.melding)
	}
	for i := 0; i < 7; i++ { // naar het model
		m = toets(m, "down")
	}
	m = toets(m, "left")
	if m.ouders.Maatje.Model != "claude-sonnet-5-5" {
		t.Errorf("model: %q", m.ouders.Maatje.Model)
	}
	m = toets(m, "esc")
	m = toets(m, "o") // ouderstand uit: pincode vergeten
	if m.ouderPin != "" || m.oi.open {
		t.Error("na de ouderstand hoort de pincode weg te zijn")
	}
}
