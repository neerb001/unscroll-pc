package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/notnil/chess"
)

// Schaken: puzzels uit de puzzeldatabase van Lichess (CC0, https://database.lichess.org),
// spelen tegen de computer (Stockfish, acht niveaus) of samen aan één toetsenbord.

//go:embed schaak/puzzels.json.gz
var puzzelData []byte

type schaakPuzzel struct {
	FEN    string   `json:"f"`
	Zetten string   `json:"m"`
	Rating int      `json:"r"`
	Themas []string `json:"t"`
	Niveau int      `json:"n"`
}

var (
	puzzelsEens sync.Once
	puzzels     []schaakPuzzel
)

func allePuzzels() []schaakPuzzel {
	puzzelsEens.Do(func() {
		if r, err := gzip.NewReader(bytes.NewReader(puzzelData)); err == nil {
			_ = json.NewDecoder(r).Decode(&puzzels)
		}
	})
	return puzzels
}

/* ---------------- Stockfish ---------------- */

type motor struct {
	mu  sync.Mutex
	cmd *exec.Cmd
	in  io.WriteCloser
	uit *bufio.Scanner
}

var errGeenMotor = errors.New("stockfish not installed")

func startMotor() (*motor, error) {
	pad, err := exec.LookPath("stockfish")
	if err != nil {
		pad = "/usr/games/stockfish" // daar zet Debian hem
		if _, err := os.Stat(pad); err != nil {
			return nil, errGeenMotor
		}
	}
	cmd := exec.Command(pad)
	stopMetOuder(cmd)
	in, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	e := &motor{cmd: cmd, in: in, uit: bufio.NewScanner(out)}
	e.stuur("uci")
	if !e.wachtOp("uciok") {
		e.stop()
		return nil, errGeenMotor
	}
	e.stuur("setoption name Threads value 1")
	e.stuur("setoption name Hash value 16")
	return e, nil
}

func (e *motor) stuur(s string) { _, _ = io.WriteString(e.in, s+"\n") }

func (e *motor) wachtOp(begin string) bool {
	for e.uit.Scan() {
		if strings.HasPrefix(e.uit.Text(), begin) {
			return true
		}
	}
	return false
}

// besteZet laat Stockfish een zet kiezen op een sterkte (skill 0-20) en met een grens.
func (e *motor) besteZet(fen string, skill int, zoek string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stuur(fmt.Sprintf("setoption name Skill Level value %d", skill))
	e.stuur("position fen " + fen)
	e.stuur("go " + zoek)
	for e.uit.Scan() {
		if r := e.uit.Text(); strings.HasPrefix(r, "bestmove ") {
			velden := strings.Fields(r)
			if len(velden) >= 2 && velden[1] != "(none)" {
				return velden[1], nil
			}
			return "", errors.New("no move")
		}
	}
	return "", errGeenMotor
}

func (e *motor) stop() {
	if e == nil {
		return
	}
	e.stuur("quit")
	if e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
	}
}

// niveaus 1-8: sterkte en hoe ver de computer mag rekenen. Niveau 1 kiest half willekeurig.
var motorNiveaus = []struct {
	skill int
	zoek  string
}{{0, "depth 1"}, {0, "depth 1"}, {2, "depth 2"}, {5, "depth 4"}, {8, "depth 6"}, {11, "movetime 300"}, {15, "movetime 500"}, {20, "movetime 1000"}}

/* ---------------- staat ---------------- */

const (
	schaakMenu = iota
	schaakPuzzelKies
	schaakNiveauKies
	schaakBord
)

const (
	soortPuzzel = iota
	soortComputer
	soortSamen
)

type schaakStaat struct {
	scherm, keuze int
	soort         int
	// partij
	startFEN string
	zetten   []string
	spel     *chess.Game
	cursor   chess.Square
	gekozen  chess.Square // -1 niets
	jijWit   bool
	niveau   int // computer 1-8, puzzel 0-3
	denkt    bool
	melding  string
	tipVan   chess.Square
	tipNaar  chess.Square
	letters  bool
	// puzzel
	puzzel    *schaakPuzzel
	oplossing []string
	stap      int // index in oplossing van de zet die het kind moet doen
	hints     int
	opgelost  bool
	motor     *motor
	volgID    int
}

func nieuwSchaak() *schaakStaat {
	return &schaakStaat{gekozen: -1, tipVan: -1, tipNaar: -1, jijWit: true, niveau: 2}
}

type bewaardePartij struct {
	Soort    int      `json:"soort"`
	StartFEN string   `json:"start"`
	Zetten   []string `json:"zetten"`
	JijWit   bool     `json:"jij_wit"`
	Niveau   int      `json:"niveau"`
	Letters  bool     `json:"letters"`
}

func schaakPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "schaak.json") }

func (s *schaakStaat) bewaar() {
	if s.soort == soortPuzzel {
		return
	}
	data, _ := json.Marshal(bewaardePartij{s.soort, s.startFEN, s.zetten, s.jijWit, s.niveau, s.letters})
	_ = os.MkdirAll(filepath.Dir(schaakPad()), 0o755)
	_ = os.WriteFile(schaakPad(), data, 0o644)
}

// opnieuwOpbouwen speelt alle zetten opnieuw vanaf de begin­stand (voor terugnemen en laden).
func (s *schaakStaat) opnieuwOpbouwen() error {
	opties := []func(*chess.Game){chess.UseNotation(chess.UCINotation{})}
	if s.startFEN != "" {
		f, err := chess.FEN(s.startFEN)
		if err != nil {
			return err
		}
		opties = append(opties, f)
	}
	s.spel = chess.NewGame(opties...)
	for _, z := range s.zetten {
		if err := s.spel.MoveStr(z); err != nil {
			return err
		}
	}
	return nil
}

func (s *schaakStaat) doeZet(z string) error {
	if err := s.spel.MoveStr(z); err != nil {
		return err
	}
	s.zetten = append(s.zetten, z)
	s.tipVan, s.tipNaar = -1, -1
	return nil
}

func (s *schaakStaat) aanZetIsKind() bool {
	if s.soort == soortSamen {
		return true
	}
	return (s.spel.Position().Turn() == chess.White) == s.jijWit
}

/* ---------------- berichten ---------------- */

type motorZetMsg struct {
	id  int
	zet string
	tip bool
	err error
}

func (m *model) motorZet(tip bool) tea.Cmd {
	s := m.dk.sk
	s.volgID++
	id, fen := s.volgID, s.spel.Position().String()
	niveau := s.niveau
	var geldig []string
	for _, z := range s.spel.ValidMoves() {
		geldig = append(geldig, z.String())
	}
	slag := map[string]bool{}
	for _, z := range s.spel.ValidMoves() {
		if z.HasTag(chess.Capture) {
			slag[z.String()] = true
		}
	}
	if s.motor == nil {
		e, err := startMotor()
		if err != nil {
			if !tip && niveau == 1 {
				e = nil // niveau 1 kan ook zonder Stockfish
			} else {
				return func() tea.Msg { return motorZetMsg{id: id, tip: tip, err: err} }
			}
		}
		s.motor = e
	}
	e := s.motor
	s.denkt = !tip
	return func() tea.Msg {
		if !tip && niveau == 1 && rand.Intn(2) == 0 || e == nil {
			// beginner: een willekeurige zet, maar slaan gaat voor
			var kies []string
			for z := range slag {
				kies = append(kies, z)
			}
			if len(kies) == 0 || rand.Intn(2) == 0 {
				kies = geldig
			}
			time.Sleep(400 * time.Millisecond)
			return motorZetMsg{id: id, zet: kies[rand.Intn(len(kies))]}
		}
		skill, zoek := 20, "movetime 400"
		if !tip {
			skill, zoek = motorNiveaus[niveau-1].skill, motorNiveaus[niveau-1].zoek
		}
		z, err := e.besteZet(fen, skill, zoek)
		return motorZetMsg{id: id, zet: z, tip: tip, err: err}
	}
}

func (m *model) schaakBericht(msg motorZetMsg) tea.Cmd {
	s := m.dk.sk
	if msg.id != s.volgID {
		return nil
	}
	s.denkt = false
	if msg.err != nil {
		if errors.Is(msg.err, errGeenMotor) {
			s.melding = m.t("chess.nostockfish")
		} else {
			s.melding = msg.err.Error()
		}
		return nil
	}
	if msg.tip {
		if z, err := (chess.UCINotation{}).Decode(s.spel.Position(), msg.zet); err == nil {
			s.tipVan, s.tipNaar = z.S1(), z.S2()
			s.melding = m.t("chess.hintshown")
		}
		return nil
	}
	if err := s.doeZet(msg.zet); err == nil {
		s.melding = m.uitslag()
		s.bewaar()
	}
	return nil
}

// uitslag: tekst als de partij voorbij is (of schaak staat).
func (m *model) uitslag() string {
	s := m.dk.sk
	switch s.spel.Outcome() {
	case chess.NoOutcome:
		if laatste := s.spel.Moves(); len(laatste) > 0 && laatste[len(laatste)-1].HasTag(chess.Check) {
			return m.t("chess.check")
		}
		return ""
	case chess.Draw:
		return m.t("chess.draw")
	}
	witWint := s.spel.Outcome() == chess.WhiteWon
	if s.soort == soortSamen {
		if witWint {
			return m.t("chess.whitewins")
		}
		return m.t("chess.blackwins")
	}
	if witWint == s.jijWit {
		return m.t("chess.youwin")
	}
	return m.t("chess.computerwins")
}

/* ---------------- puzzels ---------------- */

func (m *model) nieuwePuzzel() {
	s := m.dk.sk
	var kandidaten []schaakPuzzel
	for _, p := range allePuzzels() {
		if p.Niveau == s.niveau {
			kandidaten = append(kandidaten, p)
		}
	}
	if len(kandidaten) == 0 {
		s.melding = m.t("chess.nopuzzles")
		return
	}
	p := kandidaten[rand.Intn(len(kandidaten))]
	s.puzzel, s.soort, s.startFEN = &p, soortPuzzel, p.FEN
	s.oplossing = strings.Fields(p.Zetten)
	s.zetten, s.hints, s.opgelost, s.gekozen, s.tipVan, s.tipNaar = nil, 0, false, -1, -1, -1
	_ = s.opnieuwOpbouwen()
	// de eerste zet is van de tegenstander; daarna is het kind aan zet
	_ = s.doeZet(s.oplossing[0])
	s.stap = 1
	s.jijWit = s.spel.Position().Turn() == chess.White
	s.cursor = chess.E2
	if !s.jijWit {
		s.cursor = chess.E7
	}
	s.melding = m.puzzelOpdracht()
}

func (m *model) puzzelOpdracht() string {
	s := m.dk.sk
	kleur := m.t("chess.white")
	if !s.jijWit {
		kleur = m.t("chess.black")
	}
	for _, t := range s.puzzel.Themas {
		switch t {
		case "mateIn1", "mateIn2", "mateIn3":
			return m.t("chess.findmate", kleur, t[len(t)-1:])
		}
	}
	return m.t("chess.findbest", kleur)
}

// puzzelZet controleert een zet van het kind in een puzzel.
func (m *model) puzzelZet(z string) tea.Cmd {
	s := m.dk.sk
	goed := z == s.oplossing[s.stap]
	if !goed {
		// een andere zet die ook mat geeft is ook goed
		proef := s.spel.Clone()
		if proef.MoveStr(z) == nil && proef.Method() == chess.Checkmate {
			goed = true
		}
	}
	if !goed {
		s.melding = m.t("chess.notbest")
		return nil
	}
	_ = s.doeZet(z)
	s.stap++
	if s.stap >= len(s.oplossing) || s.spel.Outcome() != chess.NoOutcome {
		s.opgelost = true
		s.melding = m.t("chess.solved")
		return nil
	}
	// het antwoord van de tegenstander, en dan weer het kind
	_ = s.doeZet(s.oplossing[s.stap])
	s.stap++
	s.melding = m.t("chess.goodcontinue")
	return nil
}

/* ---------------- toetsen ---------------- */

func (m *model) toetsSchaak(msg tea.KeyMsg) (tea.Cmd, bool) {
	s := m.dk.sk
	k := msg.String()
	switch s.scherm {
	case schaakMenu:
		switch k {
		case "up", "k":
			s.keuze = (s.keuze + 2) % 3
		case "down", "j":
			s.keuze = (s.keuze + 1) % 3
		case "enter":
			switch s.keuze {
			case 0:
				s.scherm, s.niveau = schaakPuzzelKies, min(s.niveau, 3)
			case 1:
				s.scherm, s.niveau = schaakNiveauKies, max(s.niveau, 1)
			default:
				m.nieuwePartij(soortSamen)
			}
		case "esc":
			m.schaakStop()
			m.dk.scherm = denkMenu
		default:
			return nil, false
		}
		return nil, true
	case schaakPuzzelKies:
		switch k {
		case "up", "k":
			s.niveau = max(s.niveau-1, 0)
		case "down", "j":
			s.niveau = min(s.niveau+1, 3)
		case "enter":
			m.nieuwePuzzel()
			s.scherm = schaakBord
		case "esc":
			s.scherm = schaakMenu
		}
		return nil, true
	case schaakNiveauKies:
		switch k {
		case "up", "k":
			s.niveau = max(s.niveau-1, 1)
		case "down", "j":
			s.niveau = min(s.niveau+1, 8)
		case "left", "right", "w", "z", "b":
			s.jijWit = !s.jijWit
		case "enter":
			return m.nieuwePartij(soortComputer), true
		case "esc":
			s.scherm = schaakMenu
		}
		return nil, true
	}

	// het bord
	draai := !s.jijWit && s.soort != soortSamen // zwart onderaan
	stap := func(df, dr int) {
		if draai {
			df, dr = -df, -dr
		}
		f, r := int(s.cursor)%8+df, int(s.cursor)/8+dr
		if f >= 0 && f < 8 && r >= 0 && r < 8 {
			s.cursor = chess.Square(r*8 + f)
		}
	}
	switch k {
	case "esc":
		if s.gekozen >= 0 {
			s.gekozen = -1
			return nil, true
		}
		s.bewaar()
		s.scherm = schaakMenu
	case "up", "k":
		stap(0, 1)
	case "down", "j":
		stap(0, -1)
	case "left":
		stap(-1, 0)
	case "right":
		stap(1, 0)
	case "l":
		s.letters = !s.letters
		s.bewaar()
	case "enter", " ":
		return m.schaakKies(), true
	case "u":
		if s.soort != soortPuzzel && !s.denkt && len(s.zetten) > 0 {
			terug := 1
			if s.soort == soortComputer && len(s.zetten) >= 2 && s.aanZetIsKind() {
				terug = 2 // je eigen zet en die van de computer
			}
			s.zetten = s.zetten[:len(s.zetten)-terug]
			_ = s.opnieuwOpbouwen()
			s.gekozen, s.tipVan, s.tipNaar, s.melding = -1, -1, -1, m.t("chess.undone")
			s.volgID++ // een zet van de computer die nog onderweg is, telt niet meer
			s.denkt = false
			s.bewaar()
			if !s.aanZetIsKind() {
				return m.motorZet(false), true
			}
		}
	case "h":
		if s.soort == soortPuzzel && !s.opgelost {
			z, err := (chess.UCINotation{}).Decode(s.spel.Position(), s.oplossing[s.stap])
			if err == nil {
				s.hints++
				s.tipVan = z.S1()
				if s.hints >= 2 {
					s.tipNaar = z.S2()
				}
				s.melding = m.t("chess.hintpiece")
			}
		} else if s.spel.Outcome() == chess.NoOutcome && s.aanZetIsKind() {
			s.melding = m.t("chess.thinkinghint")
			return m.motorZet(true), true
		}
	case "n":
		switch s.soort {
		case soortPuzzel:
			m.nieuwePuzzel()
		default:
			s.scherm = schaakMenu
		}
	default:
		return nil, false
	}
	return nil, true
}

func (m *model) nieuwePartij(soort int) tea.Cmd {
	s := m.dk.sk
	s.soort, s.startFEN, s.zetten, s.puzzel = soort, "", nil, nil
	s.gekozen, s.tipVan, s.tipNaar, s.melding, s.denkt = -1, -1, -1, "", false
	_ = s.opnieuwOpbouwen()
	s.scherm = schaakBord
	s.cursor = chess.E2
	if soort == soortComputer && !s.jijWit {
		s.cursor = chess.E7
		s.bewaar()
		return m.motorZet(false)
	}
	s.bewaar()
	return nil
}

// schaakKies: eerst een eigen stuk kiezen, dan het veld waar het heen moet.
func (m *model) schaakKies() tea.Cmd {
	s := m.dk.sk
	if s.denkt || s.spel.Outcome() != chess.NoOutcome || (s.soort == soortPuzzel && s.opgelost) || !s.aanZetIsKind() {
		return nil
	}
	stuk := s.spel.Position().Board().Piece(s.cursor)
	if stuk != chess.NoPiece && stuk.Color() == s.spel.Position().Turn() {
		s.gekozen = s.cursor
		return nil
	}
	if s.gekozen < 0 {
		return nil
	}
	var zet string
	for _, z := range s.spel.ValidMoves() {
		if z.S1() == s.gekozen && z.S2() == s.cursor && (z.Promo() == chess.NoPieceType || z.Promo() == chess.Queen) {
			zet = z.String()
		}
	}
	if zet == "" {
		s.melding = m.t("chess.illegal")
		return nil
	}
	s.gekozen = -1
	if s.soort == soortPuzzel {
		return m.puzzelZet(zet)
	}
	_ = s.doeZet(zet)
	s.melding = m.uitslag()
	s.bewaar()
	if s.soort == soortComputer && s.spel.Outcome() == chess.NoOutcome {
		return m.motorZet(false)
	}
	return nil
}

func (m *model) schaakOpen() {
	s := m.dk.sk
	if s.spel != nil {
		return
	}
	var b bewaardePartij
	if data, err := os.ReadFile(schaakPad()); err == nil && json.Unmarshal(data, &b) == nil {
		s.soort, s.startFEN, s.zetten, s.jijWit, s.niveau, s.letters = b.Soort, b.StartFEN, b.Zetten, b.JijWit, b.Niveau, b.Letters
		if s.opnieuwOpbouwen() != nil {
			s.zetten = nil
			_ = s.opnieuwOpbouwen()
		}
	}
}

func (m *model) schaakStop() {
	m.dk.sk.motor.stop()
	m.dk.sk.motor = nil
}

/* ---------------- weergave ---------------- */

var stukTekens = map[chess.PieceType]string{chess.King: "♚", chess.Queen: "♛", chess.Rook: "♜", chess.Bishop: "♝", chess.Knight: "♞", chess.Pawn: "♟"}
var stukLettersNL = map[chess.PieceType]string{chess.King: "K", chess.Queen: "D", chess.Rook: "T", chess.Bishop: "L", chess.Knight: "P", chess.Pawn: "o"}
var stukLettersEN = map[chess.PieceType]string{chess.King: "K", chess.Queen: "Q", chess.Rook: "R", chess.Bishop: "B", chess.Knight: "N", chess.Pawn: "o"}

func (m model) viewSchaak(p palet) string {
	s := m.dk.sk
	dim := lipgloss.NewStyle().Foreground(p.dim)
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.puzzles")) + "  " + dim.Render(m.t("chess.title"))
	kiesLijst := func(titel string, items []string, sel int, uitleg string) string {
		var r []string
		for i, it := range items {
			if i == sel {
				r = append(r, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸ "+it+" "))
			} else {
				r = append(r, "  "+it)
			}
		}
		if uitleg != "" {
			r = append(r, "", dim.Render(uitleg))
		}
		return kop + "\n\n" + paneel(p, titel, strings.Join(r, "\n"), min(m.breed-8, 70), true)
	}
	switch s.scherm {
	case schaakMenu:
		return kiesLijst(m.t("buddy.choose"), []string{m.t("chess.puzzles"), m.t("chess.computer"), m.t("chess.together")}, s.keuze, m.t("chess.menuhint"))
	case schaakPuzzelKies:
		return kiesLijst(m.t("chess.puzzlelevel"), []string{m.t("chess.p0"), m.t("chess.p1"), m.t("chess.p2"), m.t("chess.p3")}, s.niveau, m.t("chess.lichess"))
	case schaakNiveauKies:
		var items []string
		for i := 1; i <= 8; i++ {
			items = append(items, m.t("chess.level", i)+"  "+dim.Render(m.t(fmt.Sprintf("chess.l%d", i))))
		}
		kleur := m.t("chess.white")
		if !s.jijWit {
			kleur = m.t("chess.black")
		}
		return kiesLijst(m.t("chess.computer"), items, s.niveau-1, m.t("chess.youplay", kleur)+"   "+m.t("chess.levelkeys"))
	}

	// bord
	draai := !s.jijWit && s.soort != soortSamen
	bord := s.spel.Position().Board()
	doelen := map[chess.Square]bool{}
	if s.gekozen >= 0 {
		for _, z := range s.spel.ValidMoves() {
			if z.S1() == s.gekozen {
				doelen[z.S2()] = true
			}
		}
	}
	var laatsteVan, laatsteNaar chess.Square = -1, -1
	if zz := s.spel.Moves(); len(zz) > 0 {
		laatsteVan, laatsteNaar = zz[len(zz)-1].S1(), zz[len(zz)-1].S2()
	}
	letters := stukLettersEN
	if m.inst.Taal == "nl" {
		letters = stukLettersNL
	}
	var b strings.Builder
	for ri := 0; ri < 8; ri++ {
		rij := 7 - ri
		if draai {
			rij = ri
		}
		b.WriteString(dim.Render(fmt.Sprintf(" %d ", rij+1)))
		for fi := 0; fi < 8; fi++ {
			lijn := fi
			if draai {
				lijn = 7 - fi
			}
			sq := chess.Square(rij*8 + lijn)
			achter := lipgloss.Color("#b58863")
			if (rij+lijn)%2 == 1 {
				achter = lipgloss.Color("#f0d9b5")
			}
			switch {
			case sq == s.cursor:
				achter = p.acc
			case sq == s.gekozen:
				achter = lipgloss.Color("#7fa650")
			case sq == s.tipVan || sq == s.tipNaar:
				achter = lipgloss.Color("#6fa8dc")
			case sq == laatsteVan || sq == laatsteNaar:
				achter = lipgloss.Color("#cdd26a")
			}
			tekst := "   "
			stijl := lipgloss.NewStyle().Background(achter)
			if stuk := bord.Piece(sq); stuk != chess.NoPiece {
				teken := stukTekens[stuk.Type()]
				if s.letters {
					teken = letters[stuk.Type()]
				}
				tekst = " " + teken + " "
				if stuk.Color() == chess.White {
					stijl = stijl.Foreground(lipgloss.Color("#ffffff")).Bold(true)
				} else {
					stijl = stijl.Foreground(lipgloss.Color("#000000")).Bold(true)
				}
			} else if doelen[sq] {
				tekst = " • "
				stijl = stijl.Foreground(lipgloss.Color("#3d6b21"))
			}
			if doelen[sq] && bord.Piece(sq) != chess.NoPiece {
				stijl = stijl.Underline(true)
			}
			b.WriteString(stijl.Render(tekst))
		}
		b.WriteString("\n")
	}
	kolommen := "abcdefgh"
	if draai {
		kolommen = "hgfedcba"
	}
	b.WriteString("   ")
	for _, c := range kolommen {
		b.WriteString(dim.Render(" " + string(c) + " "))
	}

	// rechts: wie is aan zet, melding, toetsen
	var info []string
	switch s.soort {
	case soortPuzzel:
		info = append(info, lipgloss.NewStyle().Bold(true).Render(m.t("chess.puzzles")), dim.Render(m.t([]string{"chess.p0", "chess.p1", "chess.p2", "chess.p3"}[s.niveau])))
	case soortComputer:
		info = append(info, lipgloss.NewStyle().Bold(true).Render(m.t("chess.computer")), dim.Render(m.t("chess.level", s.niveau)))
	default:
		info = append(info, lipgloss.NewStyle().Bold(true).Render(m.t("chess.together")))
	}
	info = append(info, "")
	if s.spel.Outcome() == chess.NoOutcome && !s.opgelost {
		beurt := m.t("chess.whitetomove")
		if s.spel.Position().Turn() == chess.Black {
			beurt = m.t("chess.blacktomove")
		}
		if s.denkt {
			beurt = m.t("chess.thinking")
		}
		info = append(info, beurt)
	}
	if s.melding != "" {
		kleur := p.acc
		if s.opgelost || s.spel.Outcome() != chess.NoOutcome {
			kleur = p.groen
		}
		info = append(info, "", lipgloss.NewStyle().Foreground(kleur).Width(30).Render(s.melding))
	}
	toetsen := m.t("chess.keys")
	switch {
	case s.soort == soortPuzzel:
		toetsen = m.t("chess.puzzlekeys")
	case s.soort == soortComputer:
		toetsen = m.t("chess.computerkeys")
	}
	return kop + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, b.String(), "   ", strings.Join(info, "\n")) + "\n\n" + dim.Render(toetsen)
}
