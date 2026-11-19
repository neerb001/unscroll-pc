package main

import (
	"errors"
	"strings"
	"testing"
)

func TestUitzettenEnHerstarten(t *testing.T) {
	var gedaan []string
	antwoord := error(nil)
	oud := afsluitOpdracht
	afsluitOpdracht = func(soort string) ([]byte, error) {
		gedaan = append(gedaan, soort)
		if antwoord != nil {
			return []byte("geen toestemming"), antwoord
		}
		return nil, nil
	}
	defer func() { afsluitOpdracht = oud }()
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 29
	m = toets(m, tabToets(tabSysteem))
	if !strings.Contains(m.View(), "[z] uitzetten") || !strings.Contains(m.View(), "[r] opnieuw opstarten") {
		t.Fatalf("de knoppen staan niet bij Systeem:\n%s", m.View())
	}
	// eerst vragen, niet meteen
	m = toets(m, "z")
	if len(gedaan) != 0 || !strings.Contains(m.melding, "Druk nog een keer op z") {
		t.Fatalf("eerst vragen: %v %q", gedaan, m.melding)
	}
	// een andere toets: toch niet
	m = toets(m, "+")
	m = toets(m, "z")
	if len(gedaan) != 0 {
		t.Fatal("na een andere toets hoort het opnieuw gevraagd te worden")
	}
	// tweede z: uitzetten
	m = toets(m, "z")
	if len(gedaan) != 1 || gedaan[0] != "poweroff" {
		t.Fatalf("uitzetten: %v", gedaan)
	}
	// herstarten, en een fout netjes melden
	antwoord = errors.New("exit status 1")
	m = toets(m, "r")
	m = toets(m, "r")
	if len(gedaan) != 2 || gedaan[1] != "reboot" || !strings.Contains(m.melding, "lukte niet") {
		t.Errorf("herstarten: %v %q", gedaan, m.melding)
	}
	// esc breekt af
	m = toets(m, "z")
	m = toets(m, "esc")
	m = toets(m, tabToets(tabSysteem))
	m = toets(m, "z")
	if len(gedaan) != 2 {
		t.Error("na esc hoort het opnieuw gevraagd te worden")
	}
}
