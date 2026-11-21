package main

import (
	"strings"
	"testing"
	"time"
)

func TestNiveauUit(t *testing.T) {
	for _, c := range []struct {
		gezien                bool
		kent, meester, totaal int
		want                  int
	}{
		{false, 0, 0, 10, niveauNiet},
		{true, 0, 0, 10, niveauBegonnen},
		{true, 4, 0, 10, niveauBegonnen},
		{true, 5, 0, 10, niveauBekend},
		{true, 9, 0, 10, niveauSterk},
		{true, 10, 9, 10, niveauMeester},
		{true, 0, 0, 0, niveauNiet},
	} {
		if got := niveauUit(c.gezien, c.kent, c.meester, c.totaal); got != c.want {
			t.Errorf("niveauUit(%v, %d, %d, %d) = %d, wil %d", c.gezien, c.kent, c.meester, c.totaal, got, c.want)
		}
	}
}

func TestHeelal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 30
	m.nu = time.Now()

	for _, v := range m.heelalVakken(m.nu) {
		if len(v.sterren) == 0 {
			t.Errorf("%s heeft geen sterren", v.naam)
		}
		for _, s := range v.sterren {
			if s.niveau != niveauNiet || s.totaal == 0 {
				t.Errorf("%s / %s: %+v bij een lege start", v.naam, s.naam, s)
			}
		}
	}

	// een topo-onderdeel helemaal als meester, de tafel van 7 sterk
	var o topoOefening
	for _, x := range topoOefeningen {
		if x.vraag != "kijk" {
			o = x
			break
		}
	}
	l := laadLeren()
	for _, d := range o.dingen() {
		st := l.stand(o.set(), d.Code)
		st.Bakje, st.Volgende = meesterBakje, m.nu.AddDate(0, 0, 3).Format("2006-01-02")
	}
	l.bewaar()
	tv := laadTafels()
	for n := 1; n <= maxTafel; n++ {
		tv.Reeks[tafelSom{a: n, b: 7}.sleutel()] = 3
	}
	tv.bewaar()

	vakken := m.heelalVakken(m.nu)
	if s := vakken[0].sterren[6]; s.niveau != niveauSterk {
		t.Errorf("tafel van 7: %+v, wil sterk", s)
	}
	if s := vakken[1].sterren[0]; s.niveau != niveauMeester || s.aanDeBeurt != 0 {
		t.Errorf("topo: %+v, wil meester en niets aan de beurt", s)
	}

	// via het menu van Leren: bovenaan
	m = toets(m, tabToets(tabLeren))
	m = toets(m, "enter")
	v := m.View()
	if m.lr.scherm != lerenHeelal || !strings.Contains(v, "Mijn heelal") || !strings.Contains(v, "meester") || !strings.Contains(v, "Tafels") {
		t.Fatalf("heelal:\n%s", v)
	}
	m = toets(m, "down")
	m = toets(m, "enter")
	if v := m.View(); !m.lr.heelal.open || !strings.Contains(v, m.t("topo.map."+o.kaart)) {
		t.Fatalf("topografie open:\n%s", v)
	}
	for _, k := range []string{"down", "pgdown", "up", "esc"} {
		m = toets(m, k)
		_ = m.View()
	}
	if m.lr.scherm != lerenHeelal || m.lr.heelal.open {
		t.Fatal("esc hoort terug naar het heelal te gaan")
	}
	m = toets(m, "esc")
	if m.lr.scherm != lerenMenu {
		t.Fatal("esc hoort terug naar het menu te gaan")
	}

	// op de tekstconsole gewone tekens
	t.Setenv("TERM", "linux")
	if sterTeken(niveauMeester) != "#" {
		t.Error("op de console hoort een meester-ster een # te zijn")
	}
}
