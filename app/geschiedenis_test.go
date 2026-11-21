package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTijdvakken(t *testing.T) {
	if len(tijdvakken) != 10 {
		t.Fatalf("%d tijdvakken", len(tijdvakken))
	}
	gezien := map[string]bool{}
	for i, v := range tijdvakken {
		if i > 0 && v.van != tijdvakken[i-1].tot {
			t.Errorf("%s sluit niet aan op het tijdvak ervoor", v.naam)
		}
		if len(v.gebeurd) < 3 || v.uitleg == "" {
			t.Errorf("%s: te weinig", v.naam)
		}
		for _, g := range v.gebeurd {
			// een gebeurtenis hoort in het tijdvak waar zijn jaartal in valt
			if g.jaar < v.van || g.jaar > v.tot {
				t.Errorf("%q (%d) valt niet in %s (%d tot %d)", g.tekst, g.jaar, v.naam, v.van, v.tot)
			}
			if gezien[g.tekst] {
				t.Errorf("%q dubbel", g.tekst)
			}
			gezien[g.tekst] = true
		}
	}
	if jaarTekst(-50, true, "nl") != "rond 50 v.Chr." || jaarTekst(1648, false, "nl") != "1648" {
		t.Errorf("jaartallen: %q %q", jaarTekst(-50, true, "nl"), jaarTekst(1648, false, "nl"))
	}
}

func TestGeschiedenisOefenen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 29
	m = toets(m, tabToets(tabLeren))
	for i := 0; i < 7; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // geschiedenis
	m = toets(m, "enter") // tijdlijn
	if !strings.Contains(m.View(), "Tijd van jagers en boeren") || !strings.Contains(m.View(), "hunebedden") {
		t.Fatalf("tijdlijn:\n%s", m.View())
	}
	for i := 0; i < 20; i++ {
		m = toets(m, "right")
	}
	if !strings.Contains(m.View(), "Tijd van televisie en computer") || !strings.Contains(m.View(), "nu") {
		t.Errorf("het laatste tijdvak:\n%s", m.View())
	}
	if n := strings.Count(m.View(), "\n") + 1; n > 29 {
		t.Errorf("past niet op het scherm van de Eee PC: %d regels", n)
	}
	g := m.lr.gesch
	// in welk tijdvak?
	m = toets(m, "esc")
	m = toets(m, "down")
	m = toets(m, "enter")
	x := g.ronde[0]
	m = toets(m, string(rune('0'+(x.vak+1)%10)))
	if x.vak == 9 {
		m = toets(m, "backspace")
		m = toets(m, "10")
	}
	m = toets(m, "enter")
	if !g.goed {
		t.Errorf("%q hoort in tijdvak %d: %q", x.tekst, x.vak+1, g.uitslag)
	}
	if n := strings.Count(m.View(), "\n") + 1; n > 29 {
		t.Errorf("past niet op het scherm van de Eee PC: %d regels", n)
	}
	// welk jaartal? (nooit een jaartal dat maar ongeveer bekend is)
	m = toets(m, "esc")
	m = toets(m, "down")
	m = toets(m, "enter")
	for _, r := range g.ronde {
		if r.rond {
			t.Errorf("vraag naar een onzeker jaartal: %q", r.tekst)
		}
	}
	y := g.ronde[0]
	antwoord := jaarTekst(y.jaar, false, "nl")
	m = toets(m, antwoord)
	m = toets(m, "enter")
	if !g.goed {
		t.Errorf("%q in %s: %q", y.tekst, antwoord, g.uitslag)
	}
	m = toets(m, "enter")
	m = toets(m, "1999")
	m = toets(m, "enter")
	if g.goed && g.ronde[1].jaar != 1999 {
		t.Error("1999 hoort fout te zijn")
	}
	// wat was eerder?
	m = toets(m, "esc")
	m = toets(m, "down")
	m = toets(m, "enter")
	p := g.paren[0]
	goed := "1"
	if p[1].jaar < p[0].jaar {
		goed = "2"
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(goed)})
	m = nm.(model)
	if !g.goed || !strings.Contains(m.View(), "Goed!") {
		t.Errorf("wat was eerder: %q", g.uitslag)
	}
}
