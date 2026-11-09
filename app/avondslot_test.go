package main

import (
	"testing"
	"time"
)

func TestAvondslot(t *testing.T) {
	o := standaardOuders() // school 20:00, weekend 20:30, open 07:00
	ams := time.FixedZone("CET", 3600)
	gevallen := []struct {
		wanneer time.Time
		dicht   bool
	}{
		{time.Date(2026, 11, 9, 19, 59, 0, 0, ams), false},  // maandag
		{time.Date(2026, 11, 9, 20, 0, 0, 0, ams), true},    // maandag, bedtijd
		{time.Date(2026, 11, 13, 20, 15, 0, 0, ams), false}, // vrijdag, weekendavond
		{time.Date(2026, 11, 13, 20, 30, 0, 0, ams), true},  // vrijdag 20:30
		{time.Date(2026, 11, 10, 6, 59, 0, 0, ams), true},   // nog voor openingstijd
		{time.Date(2026, 11, 10, 7, 0, 0, 0, ams), false},   // open
	}
	for _, g := range gevallen {
		if got := isDicht(o, g.wanneer); got != g.dicht {
			t.Errorf("%s: dicht=%v, verwacht %v", g.wanneer.Format("Mon 15:04"), got, g.dicht)
		}
	}
	if !magTijdensSlot(o, tabLezen) || magTijdensSlot(o, tabMaatje) {
		t.Error("tijdens het slot mag lezen wel en het maatje niet")
	}
}

func TestPin(t *testing.T) {
	o := standaardOuders()
	o.PinSHA256 = pinHash("1956")
	if !o.pinKlopt("1956") || o.pinKlopt("0000") {
		t.Error("pincode-controle klopt niet")
	}
}
