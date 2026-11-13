package main

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Een rekenmachine met knoppen, zoals een echte: basis en wetenschappelijk (Tab).
// Knoppen kies je met de pijltjes en Enter, of je typt gewoon. De som wordt als tekst opgebouwd
// en bij = uitgerekend: in de basisstand met exacte breuken, wetenschappelijk met kommagetallen.

var knoppenBasis = [][]string{
	{"C", "CE", "%", "÷"},
	{"7", "8", "9", "×"},
	{"4", "5", "6", "−"},
	{"1", "2", "3", "+"},
	{"±", "0", ",", "="},
}

var knoppenWet = [][]string{
	{"sin", "cos", "tan", "π"},
	{"x²", "xʸ", "√", "1/x"},
	{"log", "ln", "(", ")"},
	{"M+", "M−", "MR", "MC"},
	{"e", "Ans", "10ˣ", "eˣ"},
}

type rmStaat struct {
	wet      bool
	rij, kol int // cursor; in de wetenschappelijke stand komen de extra knoppen links
	som      string
	uit      string // laatste uitkomst (zoals getoond)
	fout     string
	ans      float64
	ansTekst string
	geheugen float64
	heeftM   bool
	nieuw    bool // na = begint een cijfer een nieuwe som
	laatste  string
}

func nieuweRekenmachine() *rmStaat { return &rmStaat{rij: 4, kol: 3} } // cursor op =

func (r *rmStaat) raster() [][]string {
	if !r.wet {
		return knoppenBasis
	}
	var uit [][]string
	for i := range knoppenBasis {
		uit = append(uit, append(append([]string{}, knoppenWet[i]...), knoppenBasis[i]...))
	}
	return uit
}

func (r *rmStaat) naarIs() {
	g := r.raster()
	r.rij, r.kol = len(g)-1, len(g[0])-1
}

/* ---------------- rekenen ---------------- */

var (
	errWortelMin = errors.New("wortelmin")
	errLogNul    = errors.New("lognul")
	impliciet    = regexp.MustCompile(`(\d|\)|π)\s*(π|e\b|sqrt|sin|cos|tan|log|ln|\()`)
)

// rekenWet rekent met kommagetallen en kent π, e, √ en sin/cos/tan (graden), log, ln.
func rekenWet(som string) (float64, error) {
	s := strings.TrimSpace(som)
	s = procentVan.ReplaceAllString(s, "($1/100)*")
	s = procentLos.ReplaceAllString(s, "($1/100)")
	s = strings.NewReplacer("×", "*", "÷", "/", ":", "/", "−", "-", "²", "^2", "³", "^3", ",", ".", "√", "sqrt").Replace(s)
	for i := 0; i < 3; i++ {
		s = impliciet.ReplaceAllString(s, "$1*$2")
	}
	p := &wetParser{t: []rune(s)}
	v, err := p.plus()
	if err != nil {
		return 0, err
	}
	p.spatie()
	if p.pos != len(p.t) {
		return 0, errSom
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, errTeGroot
	}
	return v, nil
}

type wetParser struct {
	t   []rune
	pos int
}

func (p *wetParser) spatie() {
	for p.pos < len(p.t) && unicode.IsSpace(p.t[p.pos]) {
		p.pos++
	}
}

func (p *wetParser) kijk() rune {
	p.spatie()
	if p.pos < len(p.t) {
		return p.t[p.pos]
	}
	return 0
}

func (p *wetParser) plus() (float64, error) {
	v, err := p.maal()
	for err == nil && (p.kijk() == '+' || p.kijk() == '-') {
		op := p.t[p.pos]
		p.pos++
		var w float64
		if w, err = p.maal(); op == '+' {
			v += w
		} else {
			v -= w
		}
	}
	return v, err
}

func (p *wetParser) maal() (float64, error) {
	v, err := p.macht()
	for err == nil && (p.kijk() == '*' || p.kijk() == '/') {
		op := p.t[p.pos]
		p.pos++
		var w float64
		if w, err = p.macht(); err != nil {
			break
		}
		if op == '*' {
			v *= w
		} else if w == 0 {
			return 0, errDeelNul
		} else {
			v /= w
		}
	}
	return v, err
}

func (p *wetParser) macht() (float64, error) {
	v, err := p.enkel()
	if err == nil && p.kijk() == '^' {
		p.pos++
		e, err := p.macht()
		if err != nil {
			return 0, err
		}
		return math.Pow(v, e), nil
	}
	return v, err
}

func (p *wetParser) enkel() (float64, error) {
	c := p.kijk()
	switch {
	case c == '-':
		p.pos++
		v, err := p.enkel()
		return -v, err
	case c == '+':
		p.pos++
		return p.enkel()
	case c == '(':
		p.pos++
		v, err := p.plus()
		if err != nil {
			return 0, err
		}
		if p.kijk() != ')' {
			return 0, errSom
		}
		p.pos++
		return v, nil
	case c == 'π':
		p.pos++
		return math.Pi, nil
	case c >= '0' && c <= '9' || c == '.':
		begin := p.pos
		for p.pos < len(p.t) && (p.t[p.pos] >= '0' && p.t[p.pos] <= '9' || p.t[p.pos] == '.') {
			p.pos++
		}
		return strconv.ParseFloat(string(p.t[begin:p.pos]), 64)
	case unicode.IsLetter(c):
		begin := p.pos
		for p.pos < len(p.t) && unicode.IsLetter(p.t[p.pos]) {
			p.pos++
		}
		naam := string(p.t[begin:p.pos])
		switch naam {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		}
		if p.kijk() != '(' {
			return 0, errSom
		}
		v, err := p.enkel() // het deel tussen haakjes
		if err != nil {
			return 0, err
		}
		graden := v * math.Pi / 180
		switch naam {
		case "sqrt":
			if v < 0 {
				return 0, errWortelMin
			}
			return math.Sqrt(v), nil
		case "sin":
			return math.Sin(graden), nil
		case "cos":
			return math.Cos(graden), nil
		case "tan":
			if math.Abs(math.Mod(v, 180)) == 90 {
				return 0, errTeGroot
			}
			return math.Tan(graden), nil
		case "log":
			if v <= 0 {
				return 0, errLogNul
			}
			return math.Log10(v), nil
		case "ln":
			if v <= 0 {
				return 0, errLogNul
			}
			return math.Log(v), nil
		}
	}
	return 0, errSom
}

// toonKomma: een kommagetal zonder rekenruis (sin 180 is 0, niet 1,2e-16).
func toonKomma(v float64, komma string) string {
	v = math.Round(v*1e10) / 1e10
	if v == 0 {
		v = 0 // geen -0
	}
	s := strconv.FormatFloat(v, 'g', 12, 64)
	if strings.ContainsAny(s, "e") && math.Abs(v) < 1e12 && math.Abs(v) >= 1e-6 {
		s = strings.TrimRight(strings.TrimRight(strconv.FormatFloat(v, 'f', 10, 64), "0"), ".")
	}
	return strings.Replace(s, ".", komma, 1)
}

// sluitHaakjes: ontbrekende ) achteraan aanvullen, zoals een echte rekenmachine doet.
func sluitHaakjes(s string) string {
	open := strings.Count(s, "(") - strings.Count(s, ")")
	return s + strings.Repeat(")", max(open, 0))
}

/* ---------------- knoppen ---------------- */

func (m *model) rmDruk(knop string) {
	r := m.dk.rm
	r.laatste, r.fout = knop, ""
	komma := ","
	if m.inst.Taal != "nl" {
		komma = "."
	}
	isOp := func(s string) bool { return strings.ContainsAny(s, "+−×÷^%") || s == "xʸ" }
	if r.nieuw {
		// na = : een cijfer begint opnieuw, een bewerking gaat verder met de uitkomst
		if isOp(knop) || knop == "x²" || knop == "1/x" || knop == "±" {
			r.som = r.ansTekst
		} else if knop != "M+" && knop != "M−" && knop != "MC" {
			r.som = ""
		}
		r.nieuw = false
	}
	switch knop {
	case "C":
		r.som, r.uit = "", ""
	case "CE":
		if rs := []rune(r.som); len(rs) > 0 {
			// functienamen in één keer weg
			for _, f := range []string{"sin(", "cos(", "tan(", "log(", "ln(", "√(", "Ans"} {
				if strings.HasSuffix(r.som, f) {
					r.som = strings.TrimSuffix(r.som, f)
					return
				}
			}
			r.som = string(rs[:len(rs)-1])
		}
	case "=":
		m.rmUitrekenen(komma)
	case ",":
		r.som += komma
	case "±":
		if r.som != "" {
			r.som = "−(" + r.som + ")"
		}
	case "x²":
		r.som += "²"
	case "xʸ":
		r.som += "^"
	case "1/x":
		if r.som != "" {
			r.som = "1/(" + r.som + ")"
		}
	case "√", "sin", "cos", "tan", "log", "ln":
		r.som += knop + "("
	case "10ˣ":
		r.som += "10^("
	case "eˣ":
		r.som += "e^("
	case "Ans":
		r.som += "Ans"
	case "M+", "M−":
		if v, err := m.rmWaarde(); err == nil {
			if knop == "M+" {
				r.geheugen += v
			} else {
				r.geheugen -= v
			}
			r.heeftM = true
		}
	case "MR":
		if r.heeftM {
			r.som += toonKomma(r.geheugen, komma)
		}
	case "MC":
		r.geheugen, r.heeftM = 0, false
	default: // cijfers, + − × ÷ % ( ) π e
		if len([]rune(r.som)) < 60 {
			r.som += knop
		}
	}
}

// rmWaarde: de waarde van de som (of de laatste uitkomst) als kommagetal, voor het geheugen.
func (m *model) rmWaarde() (float64, error) {
	r := m.dk.rm
	if strings.TrimSpace(r.som) == "" {
		return r.ans, nil
	}
	return rekenWet(m.rmVoorbereid(r.som))
}

func (m *model) rmVoorbereid(som string) string {
	r := m.dk.rm
	ans := strings.Replace(toonKomma(r.ans, ","), ",", ".", 1)
	return strings.ReplaceAll(sluitHaakjes(som), "Ans", "("+ans+")")
}

func (m *model) rmUitrekenen(komma string) {
	r := m.dk.rm
	if strings.TrimSpace(r.som) == "" {
		return
	}
	som := m.rmVoorbereid(r.som)
	wetenschappelijk := regexp.MustCompile(`sin|cos|tan|log|ln|√|π|e\^|\be\b|\^\(?-?\d*[.,]\d`).MatchString(som)
	var err error
	if !wetenschappelijk {
		rat, e := reken(som)
		err = e
		if err == nil {
			r.uit = toonGetal(rat, komma)
			f, _ := rat.Float64()
			r.ans = f
			r.ansTekst = r.uit
			if !rat.IsInt() {
				r.ansTekst = toonKomma(f, komma) // verder rekenen met het kommagetal
			}
		}
	} else {
		var f float64
		f, err = rekenWet(som)
		if err == nil {
			r.uit = toonKomma(f, komma)
			r.ans, r.ansTekst = f, r.uit
		}
	}
	switch {
	case errors.Is(err, errDeelNul):
		r.fout = m.t("calc.divzero")
	case errors.Is(err, errWortelMin):
		r.fout = m.t("calc.sqrtneg")
	case errors.Is(err, errLogNul):
		r.fout = m.t("calc.logzero")
	case errors.Is(err, errTeGroot):
		r.fout = m.t("calc.toobig")
	case err != nil:
		r.fout = m.t("calc.wrong")
	default:
		r.som = sluitHaakjes(r.som)
		r.nieuw = true
	}
}

func (m *model) toetsRekenmachine(msg tea.KeyMsg) (tea.Cmd, bool) {
	r := m.dk.rm
	k := msg.String()
	g := r.raster()
	switch k {
	case "esc":
		m.dk.scherm = denkMenu
		return nil, true
	case "tab":
		// wisselen tussen basis en wetenschappelijk; de cursor blijft op dezelfde knop
		knop := g[r.rij][r.kol]
		r.wet = !r.wet
		g = r.raster()
		for i, rij := range g {
			for j, k := range rij {
				if k == knop {
					r.rij, r.kol = i, j
				}
			}
		}
		r.kol = min(r.kol, len(g[0])-1)
		return nil, true
	case "up":
		r.rij = (r.rij - 1 + len(g)) % len(g)
	case "down":
		r.rij = (r.rij + 1) % len(g)
	case "left":
		r.kol = (r.kol - 1 + len(g[0])) % len(g[0])
	case "right":
		r.kol = (r.kol + 1) % len(g[0])
	case "enter":
		m.rmDruk(g[r.rij][r.kol])
	case "backspace", "delete":
		m.rmDruk("CE")
	case "=":
		m.rmDruk("=")
		r.naarIs()
	default:
		// gewoon typen: cijfers en tekens, de cursor springt naar =
		vertaal := map[string]string{"*": "×", "x": "×", "/": "÷", ":": "÷", "-": "−", ".": ",", "^": "xʸ", "c": "C", "p": "π", "s": "√"}
		if msg.Type != tea.KeyRunes && k != " " {
			return nil, false
		}
		for _, t := range msg.Runes {
			knop := string(t)
			if v, ok := vertaal[knop]; ok {
				knop = v
			}
			if strings.Contains("0123456789+−×÷%(),C√π", knop) || knop == "xʸ" {
				m.rmDruk(knop)
			}
		}
		r.naarIs()
	}
	return nil, true
}

/* ---------------- weergave ---------------- */

func (m model) viewRekenmachine(p palet) string {
	r := m.dk.rm
	dim := lipgloss.NewStyle().Foreground(p.dim)
	g := r.raster()
	knopBreed := 7
	breedte := len(g[0])*(knopBreed+1) - 1

	// display
	somRegel := r.som
	if somRegel == "" {
		somRegel = " "
	}
	groot := r.uit
	if !r.nieuw {
		groot = r.som
	}
	if groot == "" {
		groot = "0"
	}
	hoofd, rest, _ := strings.Cut(groot, "  ")
	grootTekst := lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(hoofd)
	if len([]rune(hoofd)) <= 9 && strings.Trim(hoofd, "0123456789-−,./") == "" && hoofd != "" {
		grootTekst = lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(blokletters(strings.NewReplacer("−", "-").Replace(hoofd)))
	}
	if r.fout != "" {
		grootTekst = lipgloss.NewStyle().Foreground(p.geel).Render(r.fout)
	}
	m_ := " "
	if r.heeftM {
		m_ = "M"
	}
	stand := m.t("calc.basic")
	if r.wet {
		stand = m.t("calc.scientific")
	}
	kopje := dim.Render(m_+"  "+stand) + strings.Repeat(" ", max(breedte-lipgloss.Width(m_+"  "+stand)-lipgloss.Width(somRegel)-2, 1)) + dim.Render(somRegel)
	display := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.lijn).Width(breedte).Padding(0, 1).
		Render(kopje + "\n" + lipgloss.PlaceHorizontal(breedte-2, lipgloss.Right, grootTekst) + "\n" + lipgloss.PlaceHorizontal(breedte-2, lipgloss.Right, dim.Render(strings.TrimSpace(rest)+" ")))

	// knoppen
	var rijen []string
	for i, rij := range g {
		var vakken []string
		for j, knop := range rij {
			stijl := lipgloss.NewStyle().Width(knopBreed).Align(lipgloss.Center).Background(p.bg2).Foreground(p.fg)
			switch {
			case i == r.rij && j == r.kol:
				stijl = stijl.Background(p.acc).Foreground(p.bg).Bold(true)
			case knop == r.laatste:
				stijl = stijl.Background(p.sel).Foreground(p.acc)
			case knop == "=":
				stijl = stijl.Foreground(p.groen).Bold(true)
			case strings.ContainsAny(knop, "+−×÷%") || knop == "C" || knop == "CE":
				stijl = stijl.Foreground(p.acc)
			case !strings.ContainsAny(knop, "0123456789,±"):
				stijl = stijl.Foreground(p.acc2)
			}
			vakken = append(vakken, stijl.Render(knop))
		}
		rijen = append(rijen, strings.Join(vakken, " "), "")
	}
	inhoud := display + "\n\n" + strings.Join(rijen, "\n")
	return lipgloss.NewStyle().Bold(true).Render(m.t("menu.puzzles")) + "\n\n" +
		lipgloss.PlaceHorizontal(m.breed-6, lipgloss.Center, inhoud) + "\n" + dim.Render(m.t("calc.btnkeys"))
}
