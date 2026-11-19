package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func poortModel(t *testing.T) model {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal, m.inst.Naam = "nl", "Sam"
	m.breed, m.hoog = 112, 29
	m.nu = time.Date(2026, 11, 20, 15, 0, 0, 0, time.Local) // middag: geen avondslot
	m.ouders.EerstOefenen.Sommen, m.ouders.EerstOefenen.Tafel = 5, 7
	return m
}

func TestEerstOefenen(t *testing.T) {
	m := poortModel(t)
	if !strings.Contains(m.View(), "Eerst even oefenen, Sam!") || !strings.Contains(m.View(), "tafel van 7") || !strings.Contains(m.View(), "0 van de 5 goed") {
		t.Fatalf("het oefenscherm hoort eerst te komen:\n%s", m.View())
	}
	// cijfers zijn antwoorden, geen tabbladen
	m = toets(m, "3")
	if m.tab != tabStart || m.poort.antwoord != "3" {
		t.Fatalf("een cijfer hoort in het antwoord te gaan: tab %d, antwoord %q", m.tab, m.poort.antwoord)
	}
	m = toets(m, "backspace")
	// fout telt niet
	s := m.poort.som
	if s.b != 7 {
		t.Fatalf("som buiten de tafel van 7: %s", s.tekst())
	}
	m = toets(m, fmt.Sprint(s.uitkomst()+1))
	m = toets(m, "enter")
	if laadPoort(m.nu).Goed != 0 || !strings.Contains(m.View(), "Bijna") {
		t.Fatalf("een fout antwoord hoort niet te tellen:\n%s", m.View())
	}
	// goed telt, met de voortgang in beeld
	for i := 1; i <= 4; i++ {
		m = toets(m, fmt.Sprint(m.poort.som.uitkomst()))
		m = toets(m, "enter")
	}
	if !strings.Contains(m.View(), "4 van de 5 goed") || !strings.Contains(m.View(), "● ● ● ● ○") {
		t.Fatalf("voortgang:\n%s", m.View())
	}
	m = toets(m, tabToets(tabLeren))
	m = toets(m, "enter") // nog dicht: dit is een antwoord
	if m.tab == tabLeren {
		t.Fatal("de computer gaat te vroeg open")
	}
	m = toets(m, "backspace")
	m = toets(m, "backspace")
	m = toets(m, fmt.Sprint(m.poort.som.uitkomst()))
	m = toets(m, "enter")
	if m.poortDicht() || !strings.Contains(m.melding, "De computer is open") {
		t.Fatalf("na 5 goed hoort hij open te gaan: %q", m.melding)
	}
	m = toets(m, tabToets(tabLeren))
	if m.tab != tabLeren {
		t.Error("na het oefenen hoort alles open te zijn")
	}
	// bewaard: na een herstart dezelfde dag nog steeds open, de volgende dag weer dicht
	if laadPoort(m.nu).Goed != 5 {
		t.Error("de voortgang is niet bewaard")
	}
	morgen := m.nu.AddDate(0, 0, 1)
	m.nu = morgen
	if !m.poortDicht() {
		t.Error("de volgende dag hoort het weer eerst oefenen te zijn")
	}
	// in de tafels telt het ook mee
	if len(laadTafels().Reeks) == 0 {
		t.Error("de sommen horen ook mee te tellen bij de tafels")
	}
}

func TestEerstOefenenUitzonderingen(t *testing.T) {
	m := poortModel(t)
	// ouders kunnen erlangs met o en de pincode
	m = toets(m, "o")
	if m.tab != tabSysteem || !m.pinOpen {
		t.Fatalf("o hoort naar de pincode te gaan: tab %d", m.tab)
	}
	m.ouderModus = true
	if m.poortDicht() {
		t.Error("in de ouderstand hoort alles open te zijn")
	}
	m.ouderModus = false
	m = toets(m, "esc")
	m = toets(m, tabToets(tabLezen))
	if m.tab == tabLezen {
		t.Error("vanuit Systeem mag je niet om het oefenen heen")
	}
	// tijdens het avondslot niet: dan moet het luisterboek gewoon kunnen
	m.ouders.Avondslot.Schooldagen, m.ouders.Avondslot.Weekend, m.ouders.Avondslot.Open = "20:00", "20:30", "07:00"
	m.nu = time.Date(2026, 11, 19, 22, 0, 0, 0, time.Local)
	if m.poortDicht() {
		t.Error("tijdens het avondslot hoort het oefenen niet te gelden")
	}
	// tafel 0: door elkaar, tafels 1 tot en met 10
	m.nu = time.Date(2026, 11, 20, 15, 0, 0, 0, time.Local)
	m.ouders.EerstOefenen.Tafel = 0
	tafels := map[int]bool{}
	for i := 0; i < 200; i++ {
		m.poortNieuweSom()
		tafels[m.poort.som.b] = true
		if m.poort.som.b < 1 || m.poort.som.b > 10 {
			t.Fatalf("buiten de tafels 1 tot en met 10: %s", m.poort.som.tekst())
		}
	}
	if len(tafels) < 5 {
		t.Errorf("door elkaar geeft maar %d tafels", len(tafels))
	}
	// uit
	m.ouders.EerstOefenen.Sommen = 0
	if m.poortDicht() {
		t.Error("0 sommen hoort uit te betekenen")
	}
}

func TestEerstOefenenInstellen(t *testing.T) {
	ouderTestOmgeving(t)
	data := []byte("max_volume: 70\n")
	nieuw, err := pasYAMLAan(data, []wijziging{{"eerst_oefenen.sommen", "5"}, {"eerst_oefenen.tafel", "7"}})
	if err != nil || !strings.Contains(string(nieuw), "sommen: 5") || !strings.Contains(string(nieuw), "tafel: 7") {
		t.Fatalf("instellen: %v\n%s", err, nieuw)
	}
	if _, err := pasYAMLAan(data, []wijziging{{"eerst_oefenen.sommen", "99"}}); err == nil {
		t.Error("99 sommen hoort te veel te zijn")
	}
}

// Elke 45 minuten een nieuwe ronde; daarna terug naar waar het kind was.
func TestEerstOefenenElkeZoveelMinuten(t *testing.T) {
	m := poortModel(t)
	m.ouders.EerstOefenen.ElkeMinuten = 45
	rond := func() {
		for m.poortDicht() {
			m = toets(m, fmt.Sprint(m.poort.som.uitkomst()))
			m = toets(m, "enter")
		}
	}
	m.poortNieuweSom()
	rond()
	open := m.nu
	m = toets(m, tabToets(tabLeren))
	if m.tab != tabLeren {
		t.Fatal("na de eerste ronde hoort alles open te zijn")
	}
	m.nu = open.Add(44 * time.Minute)
	if m.poortDicht() {
		t.Fatal("na 44 minuten hoort het nog open te zijn")
	}
	// na 45 minuten: bij de volgende tik weer de sommen, met de teller op 0
	m.nu = open.Add(45 * time.Minute)
	nm, _ := m.Update(tickMsg(m.nu))
	m = nm.(model)
	if m.tab != tabStart || !m.poortDicht() || !strings.Contains(m.View(), "0 van de 5 goed") {
		t.Fatalf("na 45 minuten hoort een nieuwe ronde te komen:\n%s", m.View())
	}
	// na de ronde terug naar Leren, en weer 45 minuten open
	m.poortNieuweSom()
	rond()
	if m.tab != tabLeren {
		t.Errorf("na de ronde hoort het kind terug te zijn bij Leren, niet bij tabblad %d", m.tab)
	}
	m.nu = m.nu.Add(30 * time.Minute)
	if m.poortDicht() {
		t.Error("na een nieuwe ronde begint de tijd opnieuw")
	}
	// 0 minuten: alleen de eerste keer van de dag
	m.ouders.EerstOefenen.ElkeMinuten = 0
	m.nu = m.nu.Add(2 * time.Hour) // nog voor het avondslot
	if m.poortDicht() {
		t.Error("met 0 minuten hoort het maar één keer per dag te zijn")
	}
}
