package main

import (
	"strings"
	"testing"

	"github.com/notnil/chess"
)

func TestPuzzelData(t *testing.T) {
	ps := allePuzzels()
	if len(ps) != 2000 {
		t.Fatalf("verwacht 2000 puzzels, kreeg %d", len(ps))
	}
	per := map[int]int{}
	for i, p := range ps {
		f, err := chess.FEN(p.FEN)
		if err != nil {
			t.Fatalf("puzzel %d: FEN %v", i, err)
		}
		g := chess.NewGame(f, chess.UseNotation(chess.UCINotation{}))
		for _, z := range strings.Fields(p.Zetten) {
			if err := g.MoveStr(z); err != nil {
				t.Fatalf("puzzel %d: zet %s ongeldig: %v", i, z, err)
			}
		}
		per[p.Niveau]++
	}
	if per[0] != 500 || per[3] != 500 {
		t.Errorf("verdeling over niveaus: %v", per)
	}
}

// zetMetCursor speelt een zet zoals het kind: cursor naar het stuk, Enter, cursor naar het doel, Enter.
func zetMetCursor(m model, zet string) model {
	van, naar := chess.Square(int(zet[0]-'a')+int(zet[1]-'1')*8), chess.Square(int(zet[2]-'a')+int(zet[3]-'1')*8)
	m.dk.sk.cursor = van
	m = toets(m, "enter")
	m.dk.sk.cursor = naar
	return toets(m, "enter")
}

func TestSchaakPuzzelOplossen(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m.dk = nieuweDenksport()
	m.inst.Taal = "nl"
	m.breed, m.hoog = 100, 34
	m = toets(m, "5")
	for i := 0; i < 5; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // schaken
	m = toets(m, "enter") // puzzels
	m = toets(m, "enter") // net begonnen
	s := m.dk.sk
	if s.puzzel == nil || !strings.Contains(m.View(), "Jij speelt met") {
		t.Fatalf("geen puzzel:\n%s", m.View())
	}
	// eerst een foute zet proberen (een andere geldige zet die niet de oplossing is en geen mat geeft)
	for _, z := range s.spel.ValidMoves() {
		proef := s.spel.Clone()
		_ = proef.MoveStr(z.String())
		if z.String() != s.oplossing[s.stap] && proef.Method() != chess.Checkmate && z.Promo() == chess.NoPieceType {
			m = zetMetCursor(m, z.String())
			break
		}
	}
	if !strings.Contains(m.View(), "niet de beste zet") {
		t.Fatalf("foute zet niet herkend:\n%s", m.View())
	}
	// hint, dan de oplossing spelen
	m = toets(m, "h")
	if s.tipVan < 0 {
		t.Error("hint toont geen stuk")
	}
	for !s.opgelost {
		m = zetMetCursor(m, s.oplossing[s.stap])
	}
	if !strings.Contains(m.View(), "Opgelost") {
		t.Errorf("niet opgelost:\n%s", m.View())
	}
}

func TestSchaakTegenComputer(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // zonder Stockfish speelt niveau 1 toch
	m, _ := testModel(t, Geheimen{})
	m.dk = nieuweDenksport()
	m.inst.Taal = "nl"
	s := m.dk.sk
	s.niveau = 1
	m.dk.scherm = denkSchaak
	m.tab = tabDenksport
	nm := m
	cmd := nm.nieuwePartij(soortComputer)
	if cmd != nil {
		t.Fatal("als wit begint het kind")
	}
	m = zetMetCursor(m, "e2e4") // de computer (niveau 1, zonder Stockfish) antwoordt meteen
	if len(s.zetten) != 2 || s.spel.Position().Turn() != chess.White || s.denkt {
		t.Fatalf("computer heeft niet gezet: %v", s.zetten)
	}
	m = toets(m, "u") // terugnemen: eigen zet en die van de computer
	if len(s.zetten) != 0 {
		t.Errorf("terugnemen: %v", s.zetten)
	}
}

func TestStockfish(t *testing.T) {
	e, err := startMotor()
	if err != nil {
		t.Skip("geen Stockfish:", err)
	}
	defer e.stop()
	// mat in 1: wit Dh5xf7
	z, err := e.besteZet("r1bqkbnr/pppp1ppp/2n5/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4", 20, "movetime 300")
	if err != nil || z != "h5f7" {
		t.Errorf("Stockfish vindt het herdersmat niet: %q %v", z, err)
	}
	for _, n := range motorNiveaus {
		if _, err := e.besteZet("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", n.skill, n.zoek); err != nil {
			t.Errorf("niveau %+v: %v", n, err)
		}
	}
}
