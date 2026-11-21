package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSchipEnSeizoenen(t *testing.T) {
	for _, c := range []struct{ energie, seizoen, klaar int }{
		{0, 1, 0}, {14, 1, 0}, {15, 1, 1}, {100, 1, 4}, {seizoenLengte - 1, 1, len(missieDelen) - 1},
		{seizoenLengte, 2, 0}, {seizoenLengte + 15, 2, 1},
	} {
		s, k, _ := seizoenEnDelen(c.energie)
		if s != c.seizoen || k != c.klaar {
			t.Errorf("energie %d: seizoen %d, %d klaar; wil %d, %d", c.energie, s, k, c.seizoen, c.klaar)
		}
	}
	for klaar := 0; klaar <= len(missieDelen); klaar++ {
		p := schipPlaatje(klaar)
		if len(p) > 24 {
			t.Fatalf("schip te hoog: %d", len(p))
		}
		grijs := 0
		for _, r := range p {
			if len(r) != 20 {
				t.Fatalf("rij %q is geen 20 breed", r)
			}
			for _, c := range r {
				if c != '.' && plaatjeKleuren[c] == "" {
					t.Fatalf("onbekende kleur %q in %q", c, r)
				}
				if c == 'd' {
					grijs++
				}
			}
		}
		if (grijs == 0) != (klaar == len(missieDelen)) {
			t.Errorf("%d klaar: %d grijze pixels", klaar, grijs)
		}
	}
	for _, b := range bazen {
		for _, r := range b.plaatje {
			for _, c := range r {
				if c != '.' && plaatjeKleuren[c] == "" {
					t.Fatalf("baas: onbekende kleur %q", c)
				}
			}
		}
	}
	for i := range missieDelen {
		for _, taal := range []string{"nl", "en"} {
			for _, k := range []string{"mission.part.%d", "mission.story.%d"} {
				sl := fmt.Sprintf(k, i)
				if tr(taal, sl) == sl {
					t.Errorf("%s ontbreekt (%s)", sl, taal)
				}
			}
		}
	}
	for i := range bazen {
		if tr("nl", fmt.Sprintf("mission.boss.%d", i)) == fmt.Sprintf("mission.boss.%d", i) {
			t.Errorf("baas %d heeft geen naam", i)
		}
	}
}

func TestEnergieUitSchoolvakken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	l := laadLeren()
	l.zet("topo:x", "NLD", true, time.Now())
	l.zet("topo:x", "BEL", false, time.Now())
	if e := laadMissie().Energie; e != 1 {
		t.Fatalf("een goed antwoord in de leermotor: energie %d, wil 1", e)
	}
	// een tafelsom goed
	m = toets(m, tabToets(tabLeren))
	for i := 0; i < 8; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // tafels kiezen
	m = toets(m, "enter") // beginnen
	s := m.dk.vragen[m.dk.nr]
	for _, c := range fmt.Sprint(s.uitkomst()) {
		m = toets(m, string(c))
	}
	m = toets(m, "enter")
	if e := laadMissie().Energie; e != 2 {
		t.Fatalf("na een goede tafelsom: energie %d, wil 2", e)
	}
}

func TestMissieEnBaas(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 30
	m.nu = time.Date(2026, 11, 20, 16, 0, 0, 0, time.Local)

	m = toets(m, tabToets(tabLeren))
	m = toets(m, "down")
	m = toets(m, "enter")
	if v := m.View(); m.lr.scherm != lerenMissie || !strings.Contains(v, "Missie Bliep") || !strings.Contains(v, "neergestort") {
		t.Fatalf("missie:\n%s", v)
	}

	// energie erbij: bij de volgende keer openen staat het nieuwe onderdeel in beeld
	s := laadMissie()
	s.Energie = 45
	s.bewaar()
	m = toets(m, "esc")
	m = toets(m, "enter")
	if v := m.View(); !strings.Contains(v, "Gerepareerd: het raam") || !strings.Contains(v, "linkervleugel") {
		t.Fatalf("na 45 energie:\n%s", v)
	}

	// baasgevecht: weglopen telt niet
	m = toets(m, "b")
	if m.lr.missie.gevecht == nil || !strings.Contains(m.View(), "Baasgevecht") {
		t.Fatalf("baasgevecht niet begonnen:\n%s", m.View())
	}
	m = toets(m, "esc")
	if !laadMissie().baasKan(m.nu) {
		t.Fatal("weglopen hoort niet als verloren te tellen")
	}

	// winnen
	m = toets(m, "b")
	for i := 0; i < baasLevens; i++ {
		g := m.lr.missie.gevecht
		for _, c := range fmt.Sprint(g.som.uitkomst()) {
			m = toets(m, string(c))
		}
		m = toets(m, "enter")
	}
	g := m.lr.missie.gevecht
	if !g.klaar || !g.gewonnen || !strings.Contains(m.View(), "verslagen") {
		t.Fatalf("baas niet verslagen:\n%s", m.View())
	}
	s = laadMissie()
	if s.Energie != 45+bonusBaas || s.baasKan(m.nu) || !s.baasVerslagen(m.nu) {
		t.Fatalf("na winnen: %+v", s)
	}
	m = toets(m, "enter")
	if m.lr.missie.gevecht != nil || !strings.Contains(m.View(), "Bekers gewonnen: 1") {
		t.Fatalf("terug naar de missie:\n%s", m.View())
	}
	m = toets(m, "b")
	if m.lr.missie.gevecht != nil {
		t.Fatal("twee keer per week winnen hoort niet te kunnen")
	}

	// volgende week: verliezen, en dan pas morgen weer
	m.nu = m.nu.AddDate(0, 0, 7)
	m = toets(m, "b")
	if m.lr.missie.gevecht == nil || m.lr.missie.gevecht.wie == welkeBaas(m.nu.AddDate(0, 0, -7)) {
		t.Fatal("volgende week hoort er een nieuwe baas te zijn")
	}
	for i := 0; i < baasSchilden; i++ {
		g := m.lr.missie.gevecht
		for _, c := range fmt.Sprint(g.som.uitkomst() + 1) {
			m = toets(m, string(c))
		}
		m = toets(m, "enter")
	}
	if g := m.lr.missie.gevecht; !g.klaar || g.gewonnen || !strings.Contains(m.View(), "te sterk") {
		t.Fatalf("hoort verloren te zijn:\n%s", m.View())
	}
	if laadMissie().baasKan(m.nu) || !laadMissie().baasKan(m.nu.AddDate(0, 0, 1)) {
		t.Fatal("na verliezen: vandaag niet meer, morgen weer wel")
	}

	// een heel seizoen: het schip vliegt weg, seizoen 2 begint
	m = toets(m, "esc")
	m = toets(m, "esc")
	s = laadMissie()
	s.Energie = seizoenLengte + 3
	s.bewaar()
	m = toets(m, "enter")
	if v := m.View(); !strings.Contains(v, "Seizoen 2") || !strings.Contains(v, "Volle tank") {
		t.Fatalf("nieuw seizoen:\n%s", v)
	}
}

func TestBaasSom(t *testing.T) {
	v := tafelVoortgang{Reeks: map[string]int{}}
	vorige := tafelSom{}
	for i := 0; i < 500; i++ {
		s := baasSom(v, vorige)
		if s == vorige || s.b < 2 || s.b > 10 || s.a < 2 || s.a > 10 {
			t.Fatalf("rare som %+v na %+v", s, vorige)
		}
		vorige = s
	}
}

func TestUitlegModus(t *testing.T) {
	if len(maatjeModi) != aantalModi || len(modusSleutels) != aantalModi {
		t.Fatalf("%d modi, %d instructies, %d namen", aantalModi, len(maatjeModi), len(modusSleutels))
	}
	a := &MaatjeAI{leeftijd: 11}
	sys := a.systeem("Bliep", modusUitleggen, "nl")
	if !strings.Contains(sys, "the child is the teacher") || !strings.Contains(sys, "Dutch") {
		t.Fatalf("instructie:\n%s", sys)
	}
	for _, taal := range []string{"nl", "en"} {
		for _, k := range []string{"buddy.m.teach", "buddy.m.teach.d", fmt.Sprintf("buddy.open.%d", modusUitleggen)} {
			if tr(taal, k) == k {
				t.Errorf("%s ontbreekt (%s)", k, taal)
			}
		}
	}
	if modusBestand[modusUitleggen] == "" {
		t.Error("uitleg-gesprekken horen een eigen bestandsnaam te hebben")
	}
}
