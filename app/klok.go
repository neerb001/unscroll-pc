package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Klokkijken: een echte wijzerklok en de vraag "hoe laat is het?". Je mag antwoorden zoals je
// het zegt (kwart over drie, vijf voor half vier) of met cijfers (3:15). Geen klok die meeloopt,
// geen punten. Vier niveaus: hele en halve uren, kwartieren, per vijf minuten, per minuut.

const klokNiveaus = 4

type klokStaat struct {
	niveau    int
	uur       int // 1..12
	minuut    int
	invoer    textinput.Model
	uitslag   string // na het nakijken
	goed      bool
	nagekeken bool
	terug     [][2]int // tijden die fout gingen: die komen nog eens
}

func nieuweKlok() *klokStaat {
	k := &klokStaat{niveau: 1}
	k.invoer = textinput.New()
	k.invoer.CharLimit = 40
	k.invoer.Prompt = "⏲ "
	return k
}

func (k *klokStaat) nieuweTijd() {
	k.invoer.SetValue("")
	k.uitslag, k.nagekeken = "", false
	if len(k.terug) > 0 && rand.IntN(3) == 0 { // wat fout ging komt terug
		t := k.terug[0]
		k.terug = k.terug[1:]
		k.uur, k.minuut = t[0], t[1]
		return
	}
	stap := []int{30, 15, 5, 1}[k.niveau]
	for {
		u, mi := 1+rand.IntN(12), rand.IntN(60/stap)*stap
		if u != k.uur || mi != k.minuut {
			k.uur, k.minuut = u, mi
			return
		}
	}
}

/* ---------------- tijd in woorden ---------------- */

var nlGetallen = []string{"nul", "een", "twee", "drie", "vier", "vijf", "zes", "zeven", "acht", "negen", "tien", "elf", "twaalf", "dertien", "veertien"}
var enGetallen = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen", "twenty"}

func uurNa(u int) int { return u%12 + 1 }

// tijdInWoorden: zoals je het zegt, in het Nederlands of Engels.
func tijdInWoorden(u, mi int, taal string) string {
	if taal == "nl" {
		n, v := nlGetallen[u], nlGetallen[uurNa(u)]
		switch {
		case mi == 0:
			return n + " uur"
		case mi == 15:
			return "kwart over " + n
		case mi == 30:
			return "half " + v
		case mi == 45:
			return "kwart voor " + v
		case mi < 15:
			return nlGetallen[mi] + " over " + n
		case mi < 30:
			return nlGetallen[30-mi] + " voor half " + v
		case mi < 45:
			return nlGetallen[mi-30] + " over half " + v
		default:
			return nlGetallen[60-mi] + " voor " + v
		}
	}
	en := func(x int) string {
		if x <= 20 {
			return enGetallen[x]
		}
		return "twenty-" + enGetallen[x-20]
	}
	n, v := enGetallen[u], enGetallen[uurNa(u)]
	switch {
	case mi == 0:
		return n + " o'clock"
	case mi == 15:
		return "quarter past " + n
	case mi == 30:
		return "half past " + n
	case mi == 45:
		return "quarter to " + v
	case mi < 30:
		return en(mi) + " past " + n
	default:
		return en(60-mi) + " to " + v
	}
}

var digitaal = regexp.MustCompile(`^(\d{1,2})\s*[:.u]\s*(\d{2})$`)

// leesTijd begrijpt "3:15", "15.15", "kwart over drie", "vijf voor half vier", "quarter past three".
// Geeft uur (1..12) en minuut, of ok=false. "half drie" is 2:30, maar Engels "half three" is 3:30.
func leesTijd(tekst, taal string) (int, int, bool) {
	t := strings.ToLower(strings.TrimSpace(tekst))
	t = strings.NewReplacer("-", " ", ",", " ", "minuten", "", "minuut", "", "minutes", "", "minute", "").Replace(t)
	if g := digitaal.FindStringSubmatch(t); g != nil {
		u, _ := strconv.Atoi(g[1])
		mi, _ := strconv.Atoi(g[2])
		if u > 24 || mi > 59 {
			return 0, 0, false
		}
		return twaalf(u), mi, true
	}
	var w []int // getallen als getal, andere woorden als -1
	var woorden []string
	for _, s := range strings.Fields(t) {
		if n, ok := getalWoord(s); ok {
			// twenty five: optellen
			if len(w) > 0 && w[len(w)-1] == 20 && n < 10 {
				w[len(w)-1] += n
				continue
			}
			w = append(w, n)
			woorden = append(woorden, "#")
			continue
		}
		w = append(w, -1)
		woorden = append(woorden, s)
	}
	patroon := strings.Join(woorden, " ")
	getal := func(i int) int { // het i-de getal in de zin
		for _, x := range w {
			if x >= 0 {
				if i == 0 {
					return x
				}
				i--
			}
		}
		return -1
	}
	a, b := getal(0), getal(1)
	geldig := func(u, mi int) (int, int, bool) {
		if u < 1 || u > 24 || mi < 0 || mi > 59 {
			return 0, 0, false
		}
		return twaalf(u), mi, true
	}
	switch patroon {
	case "# uur", "#", "# o'clock":
		return geldig(a, 0)
	case "half #":
		if taal != "nl" {
			return geldig(a, 30)
		}
		return geldig(vorigUur(a), 30)
	case "kwart over #", "quarter past #", "a quarter past #":
		return geldig(a, 15)
	case "kwart voor #", "quarter to #", "a quarter to #":
		return geldig(vorigUur(a), 45)
	case "# over half #":
		return geldig(vorigUur(b), 30+a)
	case "# voor half #":
		return geldig(vorigUur(b), 30-a)
	case "# over #", "# past #":
		return geldig(b, a)
	case "# voor #", "# to #":
		return geldig(vorigUur(b), 60-a)
	case "half past #":
		return geldig(a, 30)
	case "# #": // "three fifteen", "drie vijftien" kan ook
		return geldig(a, b)
	}
	return 0, 0, false
}

func twaalf(u int) int {
	u %= 12
	if u == 0 {
		return 12
	}
	return u
}

func vorigUur(u int) int {
	if u <= 1 {
		return 12
	}
	return u - 1
}

func getalWoord(s string) (int, bool) {
	if n, err := strconv.Atoi(s); err == nil {
		return n, true
	}
	for i, w := range nlGetallen {
		if s == w {
			return i, true
		}
	}
	if s == "één" {
		return 1, true
	}
	for i, w := range enGetallen {
		if s == w {
			return i, true
		}
	}
	extra := map[string]int{"vijftien": 15, "twintig": 20, "vijfentwintig": 25, "dertig": 30, "thirty": 30}
	n, ok := extra[s]
	return n, ok
}

func (m *model) klokNakijken() {
	k := m.lr.klok
	u, mi, ok := leesTijd(k.invoer.Value(), m.inst.Taal)
	if !ok {
		k.uitslag, k.goed = m.t("klok.notunderstood"), false
		return
	}
	juist := tijdInWoorden(k.uur, k.minuut, m.inst.Taal)
	cijfers := fmt.Sprintf("%d:%02d", k.uur, k.minuut)
	k.nagekeken = true
	if u == k.uur && mi == k.minuut {
		k.goed, k.uitslag = true, m.t("klok.right", juist, cijfers)
		return
	}
	k.goed, k.uitslag = false, m.t("klok.wrong", juist, cijfers)
	k.terug = append(k.terug, [2]int{k.uur, k.minuut})
}

func (m *model) toetsKlok(msg tea.KeyMsg) (tea.Cmd, bool) {
	k := m.lr.klok
	switch msg.String() {
	case "esc":
		k.invoer.Blur()
		m.lr.scherm = lerenMenu
		return nil, true
	case "enter":
		if k.nagekeken {
			k.nieuweTijd()
		} else if strings.TrimSpace(k.invoer.Value()) != "" {
			m.klokNakijken()
		}
		return nil, true
	case "left", "right":
		if k.invoer.Value() == "" || k.nagekeken {
			if msg.String() == "left" {
				k.niveau = max(k.niveau-1, 0)
			} else {
				k.niveau = min(k.niveau+1, klokNiveaus-1)
			}
			k.terug = nil
			k.nieuweTijd()
			return nil, true
		}
	}
	if k.nagekeken {
		return nil, true
	}
	var cmd tea.Cmd
	k.invoer, cmd = k.invoer.Update(msg)
	return cmd, true
}

/* ---------------- de klok tekenen ---------------- */

// brailleVlak: tekenen met braillepuntjes, 2 bij 4 puntjes per teken. Puntjes zijn dan ongeveer
// vierkant, dus een cirkel blijft rond.
type brailleVlak struct {
	b, h int // in tekens
	punt [][]bool
}

func nieuwBrailleVlak(b, h int) *brailleVlak {
	v := &brailleVlak{b: b, h: h, punt: make([][]bool, h*4)}
	for y := range v.punt {
		v.punt[y] = make([]bool, b*2)
	}
	return v
}

func (v *brailleVlak) zet(x, y int) {
	if y >= 0 && y < len(v.punt) && x >= 0 && x < len(v.punt[0]) {
		v.punt[y][x] = true
	}
}

func (v *brailleVlak) lijn(x0, y0, x1, y1 float64) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		v.zet(int(math.Round(x0+(x1-x0)*t)), int(math.Round(y0+(y1-y0)*t)))
	}
}

// tekens: elke regel als runen. Op de Linux-console (zonder braille) een eenvoudige stip.
func (v *brailleVlak) tekens() [][]rune {
	bits := [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	uit := make([][]rune, v.h)
	for cy := 0; cy < v.h; cy++ {
		uit[cy] = make([]rune, v.b)
		for cx := 0; cx < v.b; cx++ {
			n := 0
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					if v.punt[cy*4+dy][cx*2+dx] {
						n |= bits[dy][dx]
					}
				}
			}
			switch {
			case n == 0:
				uit[cy][cx] = ' '
			case alleenLetters():
				uit[cy][cx] = '•'
			default:
				uit[cy][cx] = rune(0x2800 + n)
			}
		}
	}
	return uit
}

// klokTekst tekent een wijzerklok van h regels hoog.
func klokTekst(u, mi, h int, p palet) string {
	b := h * 2 // tekens breed: een teken is ongeveer half zo breed als hoog
	rand_ := nieuwBrailleVlak(b, h)
	wijzers := nieuwBrailleVlak(b, h)
	cx, cy := float64(b*2)/2-0.5, float64(h*4)/2-0.5
	r := math.Min(cx, cy) - 1
	for a := 0.0; a < 360; a += 1.5 {
		rand_.zet(int(math.Round(cx+r*math.Sin(a*math.Pi/180))), int(math.Round(cy-r*math.Cos(a*math.Pi/180))))
	}
	for i := 0; i < 60; i += 5 { // streepjes bij de uren
		a := float64(i) * 6 * math.Pi / 180
		binnen := 0.88
		if i%15 == 0 {
			binnen = 0.8
		}
		rand_.lijn(cx+r*binnen*math.Sin(a), cy-r*binnen*math.Cos(a), cx+r*math.Sin(a), cy-r*math.Cos(a))
	}
	// de kleine wijzer schuift mee met de minuten, zoals op een echte klok
	ua := (float64(u%12) + float64(mi)/60) * 30 * math.Pi / 180
	ma := float64(mi) * 6 * math.Pi / 180
	for _, d := range []float64{-0.6, 0, 0.6} { // de kleine wijzer dikker
		wijzers.lijn(cx+d*math.Cos(ua), cy+d*math.Sin(ua), cx+r*0.5*math.Sin(ua)+d*math.Cos(ua), cy-r*0.5*math.Cos(ua)+d*math.Sin(ua))
	}
	wijzers.lijn(cx, cy, cx+r*0.82*math.Sin(ma), cy-r*0.82*math.Cos(ma))

	randT, wijzT := rand_.tekens(), wijzers.tekens()
	stijlRand := lipgloss.NewStyle().Foreground(p.dim)
	stijlWijzer := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	stijlCijfer := lipgloss.NewStyle().Foreground(p.fg).Bold(true)
	// cijfers 12, 3, 6, 9 net binnen de rand
	cijfers := map[[2]int]string{}
	for _, c := range []struct {
		uur   int
		tekst string
	}{{12, "12"}, {3, "3"}, {6, "6"}, {9, "9"}} {
		a := float64(c.uur) * 30 * math.Pi / 180
		x := int(math.Round((cx + r*0.64*math.Sin(a)) / 2))
		y := int(math.Round((cy - r*0.64*math.Cos(a)) / 4))
		if c.uur == 12 {
			x-- // twee cijfers: midden boven
		}
		cijfers[[2]int{x, y}] = c.tekst
	}
	var regels []string
	for y := 0; y < h; y++ {
		var s strings.Builder
		for x := 0; x < b; x++ {
			if c, ok := cijfers[[2]int{x, y}]; ok {
				s.WriteString(stijlCijfer.Render(c))
				x += len(c) - 1
				continue
			}
			switch {
			case wijzT[y][x] != ' ':
				s.WriteString(stijlWijzer.Render(string(wijzT[y][x])))
			case randT[y][x] != ' ':
				s.WriteString(stijlRand.Render(string(randT[y][x])))
			default:
				s.WriteString(" ")
			}
		}
		regels = append(regels, s.String())
	}
	return strings.Join(regels, "\n")
}

func (m model) viewKlok(p palet, kop string, breed int) string {
	k := m.lr.klok
	dim := lipgloss.NewStyle().Foreground(p.dim)
	h := min(max(m.hoog-14, 7), 15)
	klok := klokTekst(k.uur, k.minuut, h, p)
	var rechts []string
	rechts = append(rechts, lipgloss.NewStyle().Bold(true).Render(m.t("klok.ask")), "")
	rechts = append(rechts, k.invoer.View(), "")
	if k.uitslag != "" {
		kleur := p.groen
		if !k.goed {
			kleur = p.geel
		}
		rechts = append(rechts, lipgloss.NewStyle().Foreground(kleur).Width(max(breed-h*2-12, 20)).Render(k.uitslag), "")
	}
	if k.nagekeken {
		rechts = append(rechts, dim.Render(m.t("klok.next")))
	} else {
		rechts = append(rechts, dim.Width(max(breed-h*2-12, 20)).Render(m.t("klok.hint")))
	}
	rechts = append(rechts, "", dim.Render(m.t("klok.level", k.niveau+1, m.t(fmt.Sprintf("klok.level.%d", k.niveau)))))
	inhoud := lipgloss.JoinHorizontal(lipgloss.Top, klok, "    ", strings.Join(rechts, "\n"))
	return kop + "\n\n" + paneel(p, m.t("learn.clock"), inhoud, breed, true) + "\n" + dim.Render(m.t("klok.keys"))
}
