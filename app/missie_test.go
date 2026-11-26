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
			for _, k := range []string{"mission.part.%d"} {
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
	if v := m.View(); m.lr.scherm != lerenMissie || !strings.Contains(v, "Neergestort in de tuin") || !strings.Contains(v, "neergestort") {
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

	// weglopen kost geen gevecht
	m = toets(m, "b")
	if m.lr.missie.gevecht == nil || !strings.Contains(m.View(), "Baasgevecht") {
		t.Fatalf("baasgevecht niet begonnen:\n%s", m.View())
	}
	m = toets(m, "esc")
	if laadMissie().Tegoed != 1 {
		t.Fatalf("weglopen hoort geen gevecht te kosten: %+v", laadMissie())
	}

	win := func() {
		t.Helper()
		m = toets(m, "b")
		eerste := m.lr.missie.gevecht
		if eerste == nil {
			t.Fatalf("geen gevecht:\n%s", m.View())
		}
		for i := 0; i < baasLevens; i++ {
			g := m.lr.missie.gevecht
			for _, c := range fmt.Sprint(g.som.uitkomst()) {
				m = toets(m, string(c))
			}
			m = toets(m, "enter")
		}
		if g := m.lr.missie.gevecht; !g.klaar || !g.gewonnen {
			t.Fatalf("baas niet verslagen:\n%s", m.View())
		}
		m = toets(m, "enter")
	}
	verlies := func() {
		t.Helper()
		m = toets(m, "b")
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
		m = toets(m, "enter")
	}

	// vandaag: één gevecht, winnen
	win()
	s = laadMissie()
	if s.Energie != 45+bonusBaas || s.Tegoed != 0 || s.Baas != 1 || len(s.Gewonnen) != 1 {
		t.Fatalf("na winnen: %+v", s)
	}
	if v := m.View(); !strings.Contains(v, "Bekers gewonnen: 1") || !strings.Contains(v, "Morgen staat er weer een") {
		t.Fatalf("na winnen:\n%s", v)
	}
	m = toets(m, "b")
	if m.lr.missie.gevecht != nil {
		t.Fatal("zonder tegoed hoort er geen gevecht te zijn")
	}

	// morgen weer een, en verliezen kost het gevecht maar dezelfde baas blijft
	m.nu = m.nu.AddDate(0, 0, 1)
	verlies()
	s = laadMissie()
	if s.Tegoed != 0 || s.Baas != 1 {
		t.Fatalf("na verliezen: %+v", s)
	}

	// twee dagen niet gespeeld: twee gevechten op één dag, en dan twee verschillende bazen
	m.nu = m.nu.AddDate(0, 0, 2)
	m = toets(m, "esc")
	m = toets(m, "enter")
	if v := m.View(); !strings.Contains(v, "2 gevechten tegoed") {
		t.Fatalf("tegoed:\n%s", v)
	}
	win()
	win()
	if s = laadMissie(); s.Tegoed != 0 || s.Baas != 3 {
		t.Fatalf("na twee gevechten: %+v", s)
	}

	// lang weg: hooguit drie opgespaard
	m.nu = m.nu.AddDate(0, 0, 10)
	s = laadMissie()
	s.bijwerken(m.nu)
	if s.Tegoed != maxTegoed {
		t.Fatalf("na tien dagen: tegoed %d, wil %d", s.Tegoed, maxTegoed)
	}

	// een heel seizoen: het schip vliegt weg, seizoen 2 begint op de maan
	m = toets(m, "esc")
	s = laadMissie()
	s.Energie = seizoenLengte + 3
	s.bewaar()
	m = toets(m, "enter")
	if v := m.View(); !strings.Contains(v, "Seizoen 2: Gestrand op de maan") || !strings.Contains(v, "Volle tank") || !strings.Contains(v, "meteorieten") {
		t.Fatalf("nieuw seizoen:\n%s", v)
	}

	// ouders zien hoe lang de verhalen nog meegaan
	m.ouderModus = true
	if v := m.View(); !strings.Contains(v, "Voor ouders") || !strings.Contains(v, "seizoenen geschreven") {
		t.Fatalf("ouderinformatie:\n%s", v)
	}
	s = laadMissie()
	s.Energie = len(alleSeizoenen())*seizoenLengte + 5
	s.bewaar()
	if v := m.View(); !strings.Contains(v, "herhaald") {
		t.Fatalf("na alle seizoenen hoort de ouder te zien dat er nieuwe nodig zijn:\n%s", v)
	}
}

func TestBaasSom(t *testing.T) {
	v := tafelVoortgang{Reeks: map[string]int{}}
	for _, soort := range []int{baasMix, baasKeer, baasDelen, baasPittig} {
		vorige := tafelSom{}
		for i := 0; i < 300; i++ {
			s := baasSom(v, vorige, soort)
			if s == vorige || s.b < 2 || s.b > 10 || s.a < 2 || s.a > 10 {
				t.Fatalf("rare som %+v na %+v", s, vorige)
			}
			if (soort == baasKeer && s.delen) || (soort == baasDelen && !s.delen) || (soort == baasPittig && (s.b < 6 || s.b > 9)) {
				t.Fatalf("soort %d gaf %+v", soort, s)
			}
			vorige = s
		}
	}
}

func TestSeizoenen(t *testing.T) {
	alle := alleSeizoenen()
	if len(alle) < 4 {
		t.Fatalf("%d seizoenen", len(alle))
	}
	for i, sz := range alle {
		for _, k := range []string{"titel", "intro", "0", "1", "2", "3", "4", "5", "6", "7"} {
			for _, achter := range []string{"", ".en"} {
				tekst := sz.tekst[k+achter]
				if tekst == "" {
					t.Errorf("seizoen %d: %s%s ontbreekt", i+1, k, achter)
				}
				if strings.Contains(tekst, "\u2014") {
					t.Errorf("seizoen %d: gedachtestreepje in %s", i+1, k)
				}
			}
		}
	}
	// na het laatste geschreven seizoen gaan ze rond, zonder de neerstorting
	n := len(alle)
	for _, c := range []struct{ seizoen, wil int }{{1, 0}, {n, n - 1}, {n + 1, 1}, {n + n - 1, n - 1}, {n + n, 1}} {
		if got := seizoenVoor(c.seizoen).zeg("titel", "nl"); got != alle[c.wil].zeg("titel", "nl") {
			t.Errorf("seizoen %d: %q, wil %q", c.seizoen, got, alle[c.wil].zeg("titel", "nl"))
		}
	}
	if seizoenVoor(2).zeg("titel", "en") != "Stranded on the Moon" {
		t.Error("Engelse titel")
	}
	for i, b := range bazen {
		for _, r := range b.plaatje {
			if len(r) != 16 {
				t.Errorf("baas %d: rij %q is geen 16 breed", i, r)
			}
		}
		for _, k := range []string{"mission.boss.%d", "mission.taunt.%d"} {
			for _, taal := range []string{"nl", "en"} {
				if sl := fmt.Sprintf(k, i); tr(taal, sl) == sl {
					t.Errorf("%s ontbreekt (%s)", sl, taal)
				}
			}
		}
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
