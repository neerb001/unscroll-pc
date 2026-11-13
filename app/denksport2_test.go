package main

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestSudokuMaken(t *testing.T) {
	for niveau := 0; niveau < 3; niveau++ {
		begin := time.Now()
		s := maakSudoku(niveau, rand.New(rand.NewSource(int64(niveau+1))))
		duur := time.Since(begin)
		gegeven := 0
		for i := 0; i < 81; i++ {
			if s.Puzzel[i] != 0 {
				gegeven++
				if s.Puzzel[i] != s.Oplossing[i] {
					t.Fatal("puzzel past niet bij de oplossing")
				}
			}
		}
		kopie := s.Puzzel
		if telOplossingen(&kopie, 2) != 1 {
			t.Errorf("niveau %d heeft niet precies één oplossing", niveau)
		}
		// de oplossing is een geldige sudoku
		for cel := 0; cel < 81; cel++ {
			r := s.Oplossing
			c := r[cel]
			r[cel] = 0
			if !magHier(&r, cel, c) {
				t.Fatalf("oplossing ongeldig bij vakje %d", cel)
			}
		}
		t.Logf("niveau %d: %d gegeven, %v", niveau, gegeven, duur)
		if gegeven > sudokuGegeven[niveau]+6 {
			t.Errorf("niveau %d: te veel gegeven cijfers (%d)", niveau, gegeven)
		}
	}
}

func TestSudokuScherm(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m.dk = nieuweDenksport()
	m.inst.Taal = "nl"
	m.breed, m.hoog = 100, 34
	m = toets(m, "5")
	for i := 0; i < 3; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // Sudoku: kies niveau
	m = toets(m, "enter") // makkelijk
	s := m.dk.sd.s
	if s == nil {
		t.Fatal("geen sudoku gemaakt")
	}
	// een leeg vakje fout invullen, controleren, dan met hints afmaken
	leeg := -1
	for i := 0; i < 81; i++ {
		if s.Puzzel[i] == 0 {
			leeg = i
			break
		}
	}
	m.dk.sd.cel = leeg
	fout := s.Oplossing[leeg]%9 + 1
	m = toets(m, string(rune('0'+fout)))
	m = toets(m, "c")
	if !strings.Contains(m.View(), "kloppen nog niet") {
		t.Fatalf("controleren:\n%s", m.View())
	}
	for !s.klaar() {
		m = toets(m, "h")
	}
	if !strings.Contains(m.View(), "Gelukt") {
		t.Errorf("klaar:\n%s", m.View())
	}
	if laadSudoku() == nil {
		t.Error("sudoku niet bewaard")
	}
}

func TestRaadsels(t *testing.T) {
	r := raadselsNL[0]
	for _, goed := range []string{"een kam", "Kam!", "KAM", "het is een kam"} {
		if !r.klopt(goed) {
			t.Errorf("%q hoort goed te zijn", goed)
		}
	}
	for _, fout := range []string{"", "auto", "ka"} {
		if r.klopt(fout) {
			t.Errorf("%q hoort fout te zijn", fout)
		}
	}
	if !raadselsNL[13].klopt("R") {
		t.Error("de r van Parijs")
	}
	rnd := rand.New(rand.NewSource(3))
	for i := 0; i < 200; i++ {
		rij := maakRij(rnd, "nl")
		if !strings.HasPrefix(rij.Vraag, "Welk getal") || len(rij.Antwoord) != 1 || rij.Hint == "" {
			t.Fatalf("rij: %+v", rij)
		}
	}
	// via het scherm: raden, hint, antwoord, volgende
	t.Setenv("UNSCROLL_INSTELLINGEN", t.TempDir()+"/i.json")
	m, _ := testModel(t, Geheimen{})
	m.dk = nieuweDenksport()
	m.inst.Taal = "nl"
	m = toets(m, "5")
	for i := 0; i < 4; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter")
	eerste := m.dk.rd.huidig.Vraag
	m = toets(m, "banaan")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "is het niet") {
		t.Fatalf("fout antwoord:\n%s", m.View())
	}
	m = toets(m, "?")
	if !strings.Contains(m.View(), "Hint:") {
		t.Fatalf("hint:\n%s", m.View())
	}
	m = toets(m, m.dk.rd.huidig.Antwoord[0])
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "Goed!") {
		t.Fatalf("goed antwoord:\n%s", m.View())
	}
	m = toets(m, "enter")
	if m.dk.rd.huidig.Vraag == eerste || !m.dk.rd.huidig.getallen {
		t.Errorf("na een raadsel hoort een getallenrij te komen: %q", m.dk.rd.huidig.Vraag)
	}
}
