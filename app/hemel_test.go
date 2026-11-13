package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func hoekVerschil(a, b float64) float64 {
	d := math.Abs(norm360(a) - norm360(b))
	return math.Min(d, 360-d)
}

func TestPlaneetPosities(t *testing.T) {
	// 1 januari 2024, 0:00 UT: posities volgens NASA JPL Horizons (rechte klimming, declinatie in graden)
	tijd := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, g := range []struct {
		naam    string
		ra, dec float64
	}{{"jupiter", 33.685, 12.264}, {"saturn", 335.778, -11.839}, {"venus", 240.951, -18.769}, {"mars", 267.054, -23.961}} {
		ra, dec := planeetRADec(g.naam, tijd)
		if hoekVerschil(ra, g.ra) > 1 || math.Abs(dec-g.dec) > 1 {
			t.Errorf("%s: ra %.1f dec %.1f, verwacht ongeveer %.1f %.1f", g.naam, ra, dec, g.ra, g.dec)
		}
	}
	// volle maan 23 april 2024: maan en zon staan tegenover elkaar
	vol := time.Date(2024, 4, 23, 23, 49, 0, 0, time.UTC)
	mra, _ := maanRADec(vol)
	zra, _ := zonRADec(vol)
	if d := hoekVerschil(mra, zra); d < 170 {
		t.Errorf("volle maan: maan en zon %.0f° uit elkaar, verwacht ongeveer 180°", d)
	}
	// op 21 juni rond het middaguur staat de zon in Nederland hoog (ongeveer 61°) in het zuiden
	middag := time.Date(2024, 6, 21, 11, 40, 0, 0, time.UTC)
	ra, dec := zonRADec(middag)
	h, az := hoogteRichting(ra, dec, 52.1, 5.18, middag)
	if math.Abs(h-61) > 2 || hoekVerschil(az, 180) > 5 {
		t.Errorf("zon op 21 juni: hoogte %.1f, richting %.0f", h, az)
	}
}

func TestIssOverkomsten(t *testing.T) {
	l1 := "1 25544U 98067A   26316.88411738  .00007667  00000+0  14541-3 0  9991"
	l2 := "2 25544  51.6325 284.2547 0007335  19.0494 341.0767 15.49339933594804"
	vanaf := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	o := issOverkomsten(l1, l2, vanaf, 5, 52.1, 5.18)
	for _, x := range o {
		if x.maxHoogte < 15 || x.eind.Before(x.begin) || x.eind.Sub(x.begin) > 15*time.Minute {
			t.Errorf("rare overkomst: %+v", x)
		}
		if zonHoogte(x.top, 52.1, 5.18) > -6 {
			t.Errorf("overkomst bij daglicht: %v", x.top)
		}
	}
	t.Logf("%d zichtbare overkomsten", len(o))
}

func TestHemelScherm(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	_ = os.MkdirAll(filepath.Join(os.Getenv("XDG_CACHE_HOME"), "unscroll"), 0o755)
	_ = os.WriteFile(filepath.Join(os.Getenv("XDG_CACHE_HOME"), "unscroll", "iss.tle"), []byte("ISS (ZARYA)\n1 25544U 98067A   26316.88411738  .00007667  00000+0  14541-3 0  9991\n2 25544  51.6325 284.2547 0007335  19.0494 341.0767 15.49339933594804\n"), 0o644)
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 110, 36
	m = toets(m, "4")
	for i := 0; i < 4; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter")
	v := m.View()
	if !strings.Contains(v, "Vanavond rond") || !strings.Contains(v, "ruimtestation") || !strings.Contains(v, "Sterrenbeelden") {
		t.Fatalf("sterrenhemel:\n%s", v)
	}
	m = toets(m, "k")
	if !strings.Contains(m.View(), "recht boven je") {
		t.Errorf("hemelkaart:\n%s", m.View())
	}
}

func TestHemelUitleg(t *testing.T) {
	gezien := map[string]bool{}
	for _, o := range hemelOnderwerpen {
		if gezien[o.sleutel] {
			t.Errorf("dubbel onderwerp %s", o.sleutel)
		}
		gezien[o.sleutel] = true
		for i := 0; i < 2; i++ {
			for _, s := range []string{o.titel[i], o.feiten[i], o.tekst[i]} {
				if strings.TrimSpace(s) == "" || strings.Contains(s, "—") {
					t.Errorf("%s: lege tekst of gedachtestreep: %q", o.sleutel, s)
				}
			}
		}
	}
	for _, p := range []string{"mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune", "moon", "sun"} {
		if !gezien[p] {
			t.Errorf("onderwerp %s ontbreekt", p)
		}
	}

	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 110, 36
	m = toets(m, "4")
	for i := 0; i < 4; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter")
	m = toets(m, "u")
	if v := m.View(); !strings.Contains(v, "De zon") || !strings.Contains(v, "150 miljoen km") {
		t.Fatalf("uitleg:\n%s", v)
	}
	m = toets(m, "down")
	m = toets(m, "down")
	if v := m.View(); !strings.Contains(v, "fasen") || !strings.Contains(v, "volle maan") {
		t.Errorf("maanuitleg:\n%s", v)
	}
	m = toets(m, "up")
	m = toets(m, "up")
	m = toets(m, "up")
	if v := m.View(); !strings.Contains(v, "Het ISS en astronauten") {
		t.Errorf("rondgaan naar het laatste onderwerp:\n%s", v)
	}
	m = toets(m, "esc")
	if v := m.View(); !strings.Contains(v, "Sterrenbeelden") {
		t.Errorf("esc moet terug naar de hemel:\n%s", v)
	}
}
