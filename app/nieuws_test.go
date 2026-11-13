package main

import (
	"os"
	"strings"
	"testing"
)

func TestVersieVergelijken(t *testing.T) {
	if !versieNieuwer("v0.22.0", "v0.21.9") || !versieNieuwer("v0.10.0", "v0.9.0") || versieNieuwer("v0.22.0", "v0.22.0") || !versieNieuwer("v0.1.0", "") {
		t.Error("versies vergelijken klopt niet")
	}
	if n := nieuwsSinds("v0.21.0", "v0.22.0"); len(n) != 1 || n[0].versie != "v0.22.0" {
		t.Errorf("nieuws sinds 0.21: %v", n)
	}
	if n := nieuwsSinds("v0.22.0", "v0.23.0"); len(n) != 0 {
		t.Errorf("al gezien, toch nieuws: %v", n)
	}
	if n := nieuwsSinds("v0.21.0", "v0.21.5"); len(n) != 0 {
		t.Errorf("nieuws van een versie die nog niet draait: %v", n)
	}
	for _, n := range nieuwsLijst {
		for _, s := range n.tekst {
			if strings.TrimSpace(s) == "" || strings.Contains(s, "—") {
				t.Errorf("%s: lege tekst of gedachtestreep", n.versie)
			}
		}
	}
}

func TestNieuwsOpStartscherm(t *testing.T) {
	oud := versie
	t.Cleanup(func() { versie = oud })
	versie = "v0.23.0"

	// een computer die al instellingen had maar nog nooit nieuws zag: het nieuwste stukje
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.inst.NieuwsGezien = "" // zoals een instellingenbestand van voor deze functie
	m.nieuwsBijStart(true)
	m.tab = tabStart
	if v := m.View(); !strings.Contains(v, "nieuw!") || !strings.Contains(v, "schaakbord") {
		t.Fatalf("geen nieuws op het startscherm:\n%s", v)
	}
	m = toets(m, "x")
	if v := m.View(); strings.Contains(v, "schaakbord") {
		t.Errorf("x sluit het nieuws niet:\n%s", v)
	}
	if l := laadInstellingen(); l.NieuwsGezien != "v0.23.0" {
		t.Errorf("gezien niet bewaard: %q", l.NieuwsGezien)
	}
	// opnieuw starten: geen nieuws meer
	m2, _ := testModel(t, Geheimen{})
	m2.inst = laadInstellingen()
	m2.nieuwsBijStart(true)
	if len(m2.nieuws) != 0 {
		t.Errorf("nieuws komt terug na het sluiten: %v", m2.nieuws)
	}

	// een nieuwe computer: geen nieuws, alles is nieuw
	m3, _ := testModel(t, Geheimen{})
	_ = os.Remove(instellingenPad())
	m3.inst.NieuwsGezien = ""
	m3.nieuwsBijStart(false)
	if len(m3.nieuws) != 0 || m3.inst.NieuwsGezien != "v0.23.0" {
		t.Errorf("nieuwe computer: %v %q", m3.nieuws, m3.inst.NieuwsGezien)
	}
}
