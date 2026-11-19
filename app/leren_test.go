package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTijdInWoordenEnTerug(t *testing.T) {
	// elke tijd op de klok: in woorden zeggen en weer teruglezen geeft dezelfde tijd
	for _, taal := range []string{"nl", "en"} {
		for u := 1; u <= 12; u++ {
			for mi := 0; mi < 60; mi++ {
				w := tijdInWoorden(u, mi, taal)
				lu, lm, ok := leesTijd(w, taal)
				if !ok || lu != u || lm != mi {
					t.Fatalf("%s %d:%02d → %q → %d:%02d (%v)", taal, u, mi, w, lu, lm, ok)
				}
			}
		}
	}
	for _, g := range []struct {
		tekst, taal string
		u, mi       int
	}{
		{"kwart over drie", "nl", 3, 15}, {"half vier", "nl", 3, 30}, {"vijf voor half vier", "nl", 3, 25},
		{"tien over half twaalf", "nl", 11, 40}, {"3:15", "nl", 3, 15}, {"15.15", "nl", 3, 15}, {"12 uur", "nl", 12, 0},
		{"Drie uur", "nl", 3, 0}, {"5 voor 4", "nl", 3, 55}, {"half 1", "nl", 12, 30},
		{"half past three", "en", 3, 30}, {"half three", "en", 3, 30}, {"quarter to four", "en", 3, 45},
		{"twenty-five past seven", "en", 7, 25}, {"twenty five to nine", "en", 8, 35},
	} {
		u, mi, ok := leesTijd(g.tekst, g.taal)
		if !ok || u != g.u || mi != g.mi {
			t.Errorf("%q (%s): %d:%02d %v, verwacht %d:%02d", g.tekst, g.taal, u, mi, ok, g.u, g.mi)
		}
	}
	for _, fout := range []string{"", "banaan", "25:00", "drie over half"} {
		if _, _, ok := leesTijd(fout, "nl"); ok {
			t.Errorf("%q hoort niet begrepen te worden", fout)
		}
	}
}

func TestLeermotor(t *testing.T) {
	l := leerStand{}
	nu := time.Date(2026, 11, 20, 10, 0, 0, 0, time.Local)
	alle := []string{"a", "b", "c", "d", "e"}
	l.zet("x", "a", true, nu)
	l.zet("x", "b", false, nu)
	for i := 0; i < 4; i++ {
		l.zet("x", "c", true, nu)
	}
	if l["x"]["b"].Bakje != 0 || l["x"]["c"].Bakje != 4 || l.kent("x", alle) != 1 {
		t.Fatalf("bakjes: %+v %+v kent %d", l["x"]["b"], l["x"]["c"], l.kent("x", alle))
	}
	// vandaag aan de beurt: b (fout); daarna de nieuwe (d, e); a en c pas later
	k := l.kies("x", alle, 3, nu)
	heeft := map[string]bool{}
	for _, s := range k {
		heeft[s] = true
	}
	if !heeft["b"] || !heeft["d"] || !heeft["e"] {
		t.Errorf("gekozen: %v", k)
	}
	// een fout gaat terug naar het eerste bakje
	l.zet("x", "c", false, nu)
	if l["x"]["c"].Bakje != 0 || l["x"]["c"].Volgende != "2026-11-20" {
		t.Errorf("na een fout: %+v", l["x"]["c"])
	}
}

func TestDicteeWoordenlijst(t *testing.T) {
	for _, s := range dicteeSoorten {
		gezien := map[string]bool{}
		for _, w := range s.woorden {
			if strings.Count(w.zin, "___") != 1 {
				t.Errorf("%s: %q heeft geen plek voor het woord", s.sleutel, w.zin)
			}
			if gezien[w.schoon()] {
				t.Errorf("%s: %q dubbel", s.sleutel, w.schoon())
			}
			gezien[w.schoon()] = true
			if strings.Contains(w.woord, "|") && s.sleutel != "klank" {
				t.Errorf("%s: klankgroepen alleen bij de lange klank: %q", s.sleutel, w.woord)
			}
		}
		if tr("nl", "dictee.rule."+s.sleutel) == "dictee.rule."+s.sleutel || tr("en", "dictee.cat."+s.sleutel) == "dictee.cat."+s.sleutel {
			t.Errorf("%s: regel of naam ontbreekt", s.sleutel)
		}
	}
	a := verschil([]rune("boomen"), []rune("bomen"))
	n := 0
	for _, x := range a {
		if x {
			n++
		}
	}
	if n != 1 {
		t.Errorf("boomen tegen bomen: %d letters anders, verwacht 1 (%v)", n, a)
	}
}

func TestLerenTabblad(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m = toets(m, "3")
	if m.tab != tabLeren || !strings.Contains(m.View(), "Dictee") || !strings.Contains(m.View(), "Klokkijken") {
		t.Fatalf("3 hoort Leren te openen:\n%s", m.View())
	}
	// tafels via Leren, en met esc weer terug in het menu van Leren
	for i := 0; i < 3; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter")
	if m.lr.scherm != lerenDenk || m.dk.scherm != denkTafelsKies || !strings.Contains(m.View(), "Leren") {
		t.Fatalf("tafels niet geopend onder Leren:\n%s", m.View())
	}
	m = toets(m, "esc")
	if m.lr.scherm != lerenMenu {
		t.Errorf("esc hoort terug te gaan naar Leren, niet naar Denksport")
	}
	// Denksport heeft alleen nog de puzzels
	m = toets(m, tabToets(tabDenksport))
	v := m.View()
	if !strings.Contains(v, "Sudoku") || strings.Contains(v, "Tafels") {
		t.Errorf("Denksport:\n%s", v)
	}
}

func TestDicteeScherm(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, nep, _ := maatjeModel(t)
	var gezegd []string
	oud := spreek
	spreek = func(s string) bool { gezegd = append(gezegd, s); return true }
	defer func() { spreek = oud }()
	m.breed, m.hoog = 110, 36
	m = toets(m, tabToets(tabLeren))
	m = toets(m, "enter") // dictee
	m = toets(m, "down")  // b of d
	m = toets(m, "enter")
	d := m.lr.dictee
	if d.scherm != dicteeBezig || len(d.ronde) != 10 || len(gezegd) != 1 {
		t.Fatalf("dictee niet begonnen: scherm %d, %d woorden, %d keer gesproken", d.scherm, len(d.ronde), len(gezegd))
	}
	woord := d.huidig().schoon()
	if !strings.Contains(gezegd[0], woord) {
		t.Errorf("voorlezen: %q, woord %q", gezegd[0], woord)
	}
	// verkeerd: b en d omgedraaid
	fout := strings.NewReplacer("b", "d", "d", "b").Replace(woord)
	m = toets(m, fout)
	m = toets(m, "enter")
	if d.fout == "" || !strings.Contains(m.View(), "bed") || !strings.Contains(m.View(), "Typ het woord nu goed") {
		t.Fatalf("na een fout geen uitleg:\n%s", m.View())
	}
	if st := laadLeren()[dicteeSet("bd")][woord]; st == nil || st.Fout != 1 {
		t.Errorf("fout niet onthouden: %+v", st)
	}
	// Tab: nog een keer horen
	m = toets(m, "tab")
	if len(gezegd) != 2 {
		t.Error("Tab hoort het woord nog een keer te laten horen")
	}
	// ? vraagt Bliep waarom, binnen zijn gewone grenzen
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = voer(nm.(model), cmd)
	if m.tab != tabMaatje || m.mj.sessie == nil || len(nep.hoofdVerzoeken()) != 1 || !strings.Contains(m.View(), woord) {
		t.Fatalf("Bliep kreeg de vraag niet:\n%s", m.View())
	}
	// terug naar het dictee: het staat er nog, en goed overtypen gaat naar het volgende woord
	m = toets(m, "esc")
	m = toets(m, tabToets(tabLeren))
	m = toets(m, woord)
	m = toets(m, "enter")
	if d.fout != "" || d.nr != 1 {
		t.Fatalf("goed overgetypt maar niet verder: nr %d, fout %q", d.nr, d.fout)
	}
	// de rest goed
	for d.scherm == dicteeBezig {
		m = toets(m, d.huidig().schoon())
		m = toets(m, "enter")
	}
	if !strings.Contains(m.View(), "Deze komen later nog eens terug") || !strings.Contains(m.View(), woord) {
		t.Errorf("einde van de ronde:\n%s", m.View())
	}
}

func TestFlitslezen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 100, 30
	m = toets(m, tabToets(tabLeren))
	m = toets(m, "down")
	m = toets(m, "enter") // flitslezen
	m = toets(m, "+")
	if m.lr.flits.duur != 1400*time.Millisecond {
		t.Errorf("+ hoort het woord langer te laten staan: %v", m.lr.flits.duur)
	}
	m = toets(m, "enter") // b en d
	f := m.lr.flits
	woord := f.ronde[0]
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	m = nm.(model)
	if f.scherm != flitsTonen || cmd == nil || !strings.Contains(m.View(), groot(woord)) {
		t.Fatalf("het woord staat niet in beeld:\n%s", m.View())
	}
	m.flitsBericht(flitsMsg{f.id}) // de tijd is om
	if f.scherm != flitsWeg || strings.Contains(m.View(), groot(woord)) {
		t.Fatalf("het woord hoort weg te zijn:\n%s", m.View())
	}
	m = toets(m, " ")
	m = toets(m, "n") // niet goed gelezen: komt later terug
	if f.nr != 1 || f.ronde[len(f.ronde)-1] != woord {
		t.Errorf("niet goed gelezen woord komt niet terug: %v", f.ronde)
	}
}

func TestKlokkijken(t *testing.T) {
	t.Setenv("TERM", "foot") // braille; op de Linux-console worden het stipjes
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 100, 30
	m = toets(m, tabToets(tabLeren))
	m = toets(m, "down")
	m = toets(m, "down")
	m = toets(m, "enter")
	k := m.lr.klok
	k.uur, k.minuut = 3, 15
	braille := strings.IndexFunc(m.View(), func(r rune) bool { return r > 0x2800 && r <= 0x28FF }) >= 0
	if v := m.View(); !strings.Contains(v, "Hoe laat is het?") || !strings.Contains(v, "12") || !braille {
		t.Fatalf("geen klok te zien:\n%s", v)
	}
	m = toets(m, "kwart over drie")
	m = toets(m, "enter")
	if !k.goed || !strings.Contains(m.View(), "Goed!") {
		t.Fatalf("kwart over drie hoort goed te zijn: %q", k.uitslag)
	}
	m = toets(m, "enter") // volgende klok
	k.uur, k.minuut = 3, 30
	m = toets(m, "half drie")
	m = toets(m, "enter")
	if k.goed || !strings.Contains(k.uitslag, "half vier") || len(k.terug) != 1 {
		t.Errorf("half drie bij 3:30: %q", k.uitslag)
	}
	m = toets(m, "right")
	if k.niveau != 2 || k.nagekeken {
		t.Errorf("→ hoort moeilijker te maken: niveau %d", k.niveau)
	}
}
