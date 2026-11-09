package main

import (
	"fmt"
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
