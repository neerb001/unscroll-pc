package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// klokMinuten zet "20:30" om naar minuten na middernacht.
func klokMinuten(s string, standaard int) int {
	var u, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &u, &m); err != nil || u < 0 || u > 23 || m < 0 || m > 59 {
		return standaard
	}
	return u*60 + m
}

// bedtijd geeft de sluittijd voor de avond van t.
// Vrijdag- en zaterdagavond zijn weekendavonden: de dag erna is er geen school.
func bedtijd(o Ouders, t time.Time) string {
	switch t.Weekday() {
	case time.Friday, time.Saturday:
		return o.Avondslot.Weekend
	}
	return o.Avondslot.Schooldagen
}

// isDicht zegt of het avondslot op tijdstip t actief is.
func isDicht(o Ouders, t time.Time) bool {
	nu := t.Hour()*60 + t.Minute()
	open := klokMinuten(o.Avondslot.Open, 7*60)
	dicht := klokMinuten(bedtijd(o, t), 20*60)
	return nu < open || nu >= dicht
}

// magTijdensSlot zegt welke tabbladen tijdens het avondslot open blijven.
func magTijdensSlot(o Ouders, tab int) bool {
	switch tab {
	case tabLezen:
		return o.Avondslot.LezenMag
	case tabLuisteren:
		return o.Avondslot.LuisterboekMag
	}
	return false
}

// volgendeOpening geeft het eerstvolgende moment (vanaf t) waarop alles weer open gaat.
func volgendeOpening(o Ouders, t time.Time) time.Time {
	open := klokMinuten(o.Avondslot.Open, 7*60)
	d := time.Date(t.Year(), t.Month(), t.Day(), open/60, open%60, 0, 0, t.Location())
	if !d.After(t) {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

// pauzeUit betekent: avondslot uit tot een ouder hem weer aanzet.
var pauzeUit = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// De pauze wordt bewaard, zodat hij een herstart van de app overleeft.
func pauzePad() string { return filepath.Join(filepath.Dir(instellingenPad()), "pauze.json") }

func laadPauze() time.Time {
	var p struct {
		Tot time.Time `json:"tot"`
	}
	if data, err := os.ReadFile(pauzePad()); err == nil {
		_ = json.Unmarshal(data, &p)
	}
	return p.Tot
}

func bewaarPauze(tot time.Time) {
	_ = os.MkdirAll(filepath.Dir(pauzePad()), 0o755)
	data, _ := json.Marshal(struct {
		Tot time.Time `json:"tot"`
	}{tot})
	_ = os.WriteFile(pauzePad(), data, 0o644)
}
