package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/notnil/chess"
)

// Het schaakbord tekenen.
//
// Snelheid: het bord bestaat uit honderden gekleurde vakjes. Die elk apart door lipgloss laten
// opmaken kost op een oude netbook een tiende seconde per toetsaanslag. Daarom schrijven we de
// kleurcodes hier zelf, en alleen als de kleur verandert.
//
// Vorm: elk stuk is een klein plaatje van puntjes. Een terminalteken wordt verdeeld in 2 bij 3
// blokjes (de "sextant"-tekens, die de foot-terminal zelf tekent), zodat een veld van 7 bij 3
// tekens 14 bij 9 puntjes heeft: genoeg voor ronde koppen en schuine kanten.

// stukPlaatjes: groot (14x9 puntjes, veld 7x3 tekens) en klein (10x6 puntjes, veld 5x2 tekens).
var stukPlaatjes = map[int]map[chess.PieceType][]string{
	3: {
		chess.King: {
			"......##......",
			".....####.....",
			"......##......",
			"....######....",
			"...########...",
			"...########...",
			"....######....",
			"..##########..",
			"..............",
		},
		chess.Queen: {
			"..#...##...#..",
			"..##.####.##..",
			"...########...",
			"...########...",
			"....######....",
			"....######....",
			"...########...",
			"..##########..",
			"..............",
		},
		chess.Rook: {
			"..............",
			"...##.##.##...",
			"...########...",
			"....######....",
			"....######....",
			"....######....",
			"...########...",
			"..##########..",
			"..............",
		},
		chess.Bishop: {
			"......##......",
			".....####.....",
			"....###.##....",
			"....##.###....",
			".....####.....",
			"......##......",
			"....######....",
			"..##########..",
			"..............",
		},
		chess.Knight: {
			"......###.....",
			"....######....",
			"...###.####...",
			"..#########...",
			"..###..####...",
			"......#####...",
			"....#######...",
			"..##########..",
			"..............",
		},
		chess.Pawn: {
			"..............",
			"..............",
			".....####.....",
			"....######....",
			".....####.....",
			"......##......",
			"....######....",
			"...########...",
			"..............",
		},
	},
	2: {
		chess.King: {
			"....##....",
			"..######..",
			"....##....",
			"..######..",
			"..######..",
			".########.",
		},
		chess.Queen: {
			".#..##..#.",
			".##.##.##.",
			"..######..",
			"...####...",
			"..######..",
			".########.",
		},
		chess.Rook: {
			"..........",
			".##.##.##.",
			".########.",
			"..######..",
			"..######..",
			".########.",
		},
		chess.Bishop: {
			"....##....",
			"...####...",
			"..######..",
			"...####...",
			"....##....",
			".########.",
		},
		chess.Knight: {
			"...####...",
			"..######..",
			".###.###..",
			".#..####..",
			"...#####..",
			".########.",
		},
		chess.Pawn: {
			"..........",
			"....##....",
			"...####...",
			"....##....",
			"..######..",
			"..........",
		},
	},
}

// sextant geeft het teken voor een blokje van 2 bij 3 puntjes.
// bits: 1 linksboven, 2 rechtsboven, 4 linksmidden, 8 rechtsmidden, 16 linksonder, 32 rechtsonder.
func sextant(bits int) rune {
	switch bits {
	case 0:
		return ' '
	case 21:
		return '▌'
	case 42:
		return '▐'
	case 63:
		return '█'
	}
	n := bits - 1
	if bits > 21 {
		n--
	}
	if bits > 42 {
		n--
	}
	return rune(0x1FB00 + n)
}

var (
	stukRegelsMu sync.Mutex
	stukRegels   = map[[2]int][]string{}
)

// stukTekst zet het plaatje van een stuk om in regels terminaltekens (één keer, daarna onthouden).
func stukTekst(h int, stuk chess.PieceType) []string {
	sleutel := [2]int{h, int(stuk)}
	stukRegelsMu.Lock()
	defer stukRegelsMu.Unlock()
	if r, ok := stukRegels[sleutel]; ok {
		return r
	}
	plaatje := stukPlaatjes[h][stuk]
	aan := func(x, y int) bool { return y < len(plaatje) && x < len(plaatje[y]) && plaatje[y][x] == '#' }
	var uit []string
	for rij := 0; rij < h; rij++ {
		var b strings.Builder
		for kol := 0; kol < len(plaatje[0])/2; kol++ {
			bits := 0
			for i := 0; i < 6; i++ {
				if aan(kol*2+i%2, rij*3+i/2) {
					bits |= 1 << i
				}
			}
			b.WriteRune(sextant(bits))
		}
		uit = append(uit, b.String())
	}
	stukRegels[sleutel] = uit
	return uit
}

// Kleuren van het bord. Wit en zwart moeten op alle velden goed afsteken.
const (
	veldLicht   = "#b58f63"
	veldDonker  = "#8c6642"
	veldCursor  = "#3f7fc4"
	veldGekozen = "#5e8f35"
	veldTip     = "#9a5fb8"
	veldLaatste = "#9c8f2e"
	stukWit     = "#ffffff"
	stukZwart   = "#000000"
	doelKleur   = "#d4f5b0"
)

var (
	kleurCodeMu sync.Mutex
	kleurCodes  = map[string]string{}
)

// kleurCode geeft de terminalcode voor een kleur, passend bij wat deze terminal kan.
func kleurCode(hex string, achtergrond bool) string {
	prof := lipgloss.ColorProfile()
	sleutel := fmt.Sprintf("%d%t%s", prof, achtergrond, hex)
	kleurCodeMu.Lock()
	defer kleurCodeMu.Unlock()
	if c, ok := kleurCodes[sleutel]; ok {
		return c
	}
	c := ""
	if k := prof.Color(hex); k != nil {
		c = k.Sequence(achtergrond)
	}
	kleurCodes[sleutel] = c
	return c
}

// kleurRegel bouwt één regel op en schrijft alleen een kleurcode als de kleur verandert.
type kleurRegel struct {
	b      strings.Builder
	fg, bg string
	vet    bool
	open   bool
}

func (r *kleurRegel) zet(fg, bg string, vet bool, tekst string) {
	if !r.open || fg != r.fg || bg != r.bg || vet != r.vet {
		var delen []string
		if r.open {
			delen = append(delen, "0")
		}
		if vet {
			delen = append(delen, "1")
		}
		if c := kleurCode(fg, false); fg != "" && c != "" {
			delen = append(delen, c)
		}
		if c := kleurCode(bg, true); bg != "" && c != "" {
			delen = append(delen, c)
		}
		if len(delen) > 0 && !(len(delen) == 1 && delen[0] == "0" && !r.open) {
			r.b.WriteString("\x1b[" + strings.Join(delen, ";") + "m")
		}
		r.fg, r.bg, r.vet, r.open = fg, bg, vet, true
	}
	r.b.WriteString(tekst)
}

func (r *kleurRegel) klaar() string {
	if r.open && (r.fg != "" || r.bg != "" || r.vet) {
		r.b.WriteString("\x1b[0m")
	}
	return r.b.String()
}

// alleenLetters: op de gewone tekstconsole (zonder foot) bestaan de tekentjes voor de plaatjes niet.
func alleenLetters() bool { return os.Getenv("TERM") == "linux" }

func (m model) stukLetter(stuk chess.PieceType) string {
	letters := stukLettersEN
	if m.inst.Taal == "nl" {
		letters = stukLettersNL
	}
	if l := letters[stuk]; l != "o" {
		return l
	}
	return "i" // pion: een rechtopstaand poppetje leest beter dan een rondje
}

// schaakVeld geeft de h regels van één veld (elk w tekens breed) als losse stukjes (fg, tekst).
func (m model) schaakVeldRegels(h int, stuk chess.Piece, doel, metLetters bool) []string {
	w := 2*h + 1
	leeg := strings.Repeat(" ", w)
	regels := make([]string, h)
	for i := range regels {
		regels[i] = leeg
	}
	mid := w / 2
	switch {
	case stuk != chess.NoPiece && (metLetters || h == 1):
		regels[h/2] = strings.Repeat(" ", mid) + m.stukLetter(stuk.Type()) + strings.Repeat(" ", w-mid-1)
	case stuk != chess.NoPiece:
		copy(regels, stukTekst(h, stuk.Type()))
	case doel:
		regels[h/2] = strings.Repeat(" ", mid) + "●" + strings.Repeat(" ", w-mid-1)
	}
	return regels
}

// schaakBord tekent het bord met velden van h regels hoog.
func (m model) schaakBord(p palet, h int) string {
	s := m.dk.sk
	w := 2*h + 1
	metLetters := s.letters || alleenLetters()
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
	dim := string(p.dim)

	var uit []string
	for ri := 0; ri < 8; ri++ {
		rij := 7 - ri
		if draai {
			rij = ri
		}
		regels := make([]kleurRegel, h)
		for i := range regels {
			label := "   "
			if i == h/2 {
				label = fmt.Sprintf(" %d ", rij+1)
			}
			regels[i].zet(dim, "", false, label)
		}
		for fi := 0; fi < 8; fi++ {
			lijn := fi
			if draai {
				lijn = 7 - fi
			}
			sq := chess.Square(rij*8 + lijn)
			achter := veldDonker
			if (rij+lijn)%2 == 1 {
				achter = veldLicht
			}
			stuk := bord.Piece(sq)
			slaan := doelen[sq] && stuk != chess.NoPiece
			switch {
			case sq == s.cursor:
				achter = veldCursor
			case sq == s.gekozen:
				achter = veldGekozen
			case slaan:
				achter = "#b0503c" // een stuk dat je kunt slaan
			case sq == s.tipVan || sq == s.tipNaar:
				achter = veldTip
			case sq == laatsteVan || sq == laatsteNaar:
				achter = veldLaatste
			}
			voor := doelKleur
			if stuk != chess.NoPiece {
				voor = stukWit
				if stuk.Color() == chess.Black {
					voor = stukZwart
				}
			}
			for i, r := range m.schaakVeldRegels(h, stuk, doelen[sq], metLetters) {
				regels[i].zet(voor, achter, stuk != chess.NoPiece, r)
			}
		}
		for i := range regels {
			uit = append(uit, regels[i].klaar())
		}
	}
	kolommen := "abcdefgh"
	if draai {
		kolommen = "hgfedcba"
	}
	var onder kleurRegel
	onder.zet(dim, "", false, "   ")
	for _, c := range kolommen {
		half := (w - 1) / 2
		onder.zet(dim, "", false, strings.Repeat(" ", half)+string(c)+strings.Repeat(" ", w-1-half))
	}
	uit = append(uit, onder.klaar())
	return strings.Join(uit, "\n")
}

var witteStukken = map[chess.PieceType]chess.Piece{chess.King: chess.WhiteKing, chess.Queen: chess.WhiteQueen, chess.Rook: chess.WhiteRook,
	chess.Bishop: chess.WhiteBishop, chess.Knight: chess.WhiteKnight, chess.Pawn: chess.WhitePawn}

// schaakBordBreedte: hoeveel tekens het bord breed is.
func schaakBordBreedte(h int) int { return 3 + 8*(2*h+1) }

// stukkenUitleg: welk plaatje welk stuk is, zodat je ze leert herkennen. Kiest de grootste vorm
// die in de beschikbare regels en breedte past: grote plaatjes, kleine plaatjes of letters.
func (m model) stukkenUitleg(h, breed, regelsMax int) string {
	s := m.dk.sk
	namen := []struct {
		stuk  chess.PieceType
		sleut string
	}{{chess.King, "chess.n.king"}, {chess.Queen, "chess.n.queen"}, {chess.Rook, "chess.n.rook"}, {chess.Bishop, "chess.n.bishop"}, {chess.Knight, "chess.n.knight"}, {chess.Pawn, "chess.n.pawn"}}
	const naamBreed = 8
	maak := func(th int, metLetters, dicht bool) (string, int) {
		w := 2*th + 1
		perRij := 2
		if breed < 2*(w+naamBreed) {
			perRij = 1
		}
		tussen := 1
		if th == 1 || dicht {
			tussen = 0
		}
		var blokken []string
		for i := 0; i < len(namen); i += perRij {
			regels := make([]kleurRegel, th)
			for j := i; j < i+perRij && j < len(namen); j++ {
				for k, r := range m.schaakVeldRegels(th, witteStukken[namen[j].stuk], false, metLetters) {
					achter := veldLicht
					if dicht && (i/perRij+j)%2 == 1 {
						achter = veldDonker // zonder witregels: om en om, net als op het bord
					}
					regels[k].zet(stukWit, achter, true, r)
					naam := ""
					if k == th/2 {
						naam = " " + m.t(namen[j].sleut)
					}
					regels[k].zet("", "", false, naam+strings.Repeat(" ", max(naamBreed-lipgloss.Width(naam), 1)))
				}
			}
			for k := range regels {
				blokken = append(blokken, regels[k].klaar())
			}
			if tussen == 1 && i+perRij < len(namen) {
				blokken = append(blokken, "")
			}
		}
		return strings.Join(blokken, "\n"), len(blokken)
	}
	if !(s.letters || alleenLetters()) {
		for th := min(h, 3); th >= 2; th-- {
			if tekst, n := maak(th, false, false); n <= regelsMax {
				return tekst
			}
		}
		if h >= 2 {
			if tekst, n := maak(2, false, true); n <= regelsMax {
				return tekst
			}
		}
	}
	if tekst, n := maak(1, true, false); n <= regelsMax {
		return tekst
	}
	return ""
}
