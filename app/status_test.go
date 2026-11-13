package main

import (
	"strings"
	"testing"
	"time"
)

func TestStatusbalkPastAltijd(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.nu = time.Date(2026, 11, 13, 21, 45, 0, 0, time.Local)
	for _, breed := range []int{140, 112, 92, 80, 64, 48, 36} {
		for _, melding := range []string{"", "Ouderstand aan. Nieuwe versie v0.23.0 klaar: druk op R om nu te herstarten."} {
			m.breed = breed
			m.melding, m.meldingTot = melding, time.Now().Add(time.Minute)
			v := m.viewStatus(palet{})
			if strings.Contains(v, "\n") {
				t.Errorf("breedte %d: balk breekt over twee regels:\n%s", breed, v)
			}
			if !strings.Contains(v, "21:45") {
				t.Errorf("breedte %d: klok ontbreekt: %q", breed, v)
			}
			if breed >= 64 && !strings.Contains(v, "8") {
				t.Errorf("breedte %d: tabblad 8 ontbreekt: %q", breed, v)
			}
		}
	}
}
