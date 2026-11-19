package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Topografie: Europa (landen en hoofdsteden, groep 7), Nederland (provincies en hun
// hoofdsteden, steden, rivieren, wateren en Waddeneilanden, groep 6 en 7) en de wereld
// (werelddelen, landen en hoofdsteden, oceanen, zeeën, gebergten en rivieren, groep 8). Een kaart getekend
// met braillepuntjes waarop één ding oplicht: een land of provincie ingekleurd, een stad als
// stip, een rivier als lijn. Eerst rondkijken, dan overhoren; de leermotor laat terugkomen wat
// lastig is.
//
// De kaarten komen uit Natural Earth (publiek domein), vereenvoudigd met tools/maak_topo.py.

//go:embed topo/*.json
var topoBestanden embed.FS

type topoStad struct {
	NL   string     `json:"nl"`
	EN   string     `json:"en"`
	Ook  []string   `json:"ook"`
	Punt [2]float64 `json:"punt"`
}

type topoDing struct {
	Code  string         `json:"code"`
	Groep string         `json:"groep"` // land, provincie, stad, water
	NL    string         `json:"nl"`
	EN    string         `json:"en"`
	Ook   []string       `json:"ook"`
	Vlak  [][][2]float64 `json:"vlak"`
	Punt  *[2]float64    `json:"punt"`
	Lijn  [][][2]float64 `json:"lijn"`
	Stad  *topoStad      `json:"stad"`
}

type topoKaart struct {
	Kader       [4]float64     `json:"kader"`
	Achtergrond [][][2]float64 `json:"achtergrond"` // land dat meedoet op de kaart, maar niet gevraagd wordt
	Grijs       [][][2]float64 `json:"grijs"`       // land eromheen (buiten Europa, buurlanden van Nederland)
	Lijnen      [][][2]float64 `json:"lijnen"`
	Items       []topoDing     `json:"items"`
}

var (
	topoMu      sync.Mutex
	topoKaarten = map[string]*topoKaart{}
)

func topoKaartVan(naam string) *topoKaart {
	topoMu.Lock()
	defer topoMu.Unlock()
	if k, ok := topoKaarten[naam]; ok {
		return k
	}
	k := &topoKaart{}
	if data, err := topoBestanden.ReadFile("topo/" + naam + ".json"); err == nil {
		_ = json.Unmarshal(data, k)
	}
	topoKaarten[naam] = k
	return k
}

// topoOefening: wat je oefent, op welke kaart, en of je de naam of de hoofdstad noemt.
type topoOefening struct {
	sleutel, kaart, groep string
	vraag                 string // "kijk", "naam" of "stad"
}

var topoOefeningen = []topoOefening{
	{"topo.eu.look", "europa", "land", "kijk"},
	{"topo.eu.countries", "europa", "land", "naam"},
	{"topo.eu.capitals", "europa", "land", "stad"},
	{"topo.nl.look", "nederland", "", "kijk"},
	{"topo.nl.provinces", "nederland", "provincie", "naam"},
	{"topo.nl.capitals", "nederland", "provincie", "stad"},
	{"topo.nl.cities", "nederland", "stad", "naam"},
	{"topo.nl.water", "nederland", "water", "naam"},
	{"topo.w.look", "wereld", "", "kijk"},
	{"topo.w.continents", "wereld", "werelddeel", "naam"},
	{"topo.w.countries", "wereld", "land", "naam"},
	{"topo.w.capitals", "wereld", "land", "stad"},
	{"topo.w.nature", "wereld", "natuur", "naam"},
}

func (o topoOefening) set() string { return "topo:" + o.kaart + ":" + o.groep + ":" + o.vraag }

// dingen: alles wat bij deze oefening hoort ("" is alles op de kaart).
func (o topoOefening) dingen() []*topoDing {
	k := topoKaartVan(o.kaart)
	var uit []*topoDing
	for i := range k.Items {
		if o.groep == "" || k.Items[i].Groep == o.groep {
			uit = append(uit, &k.Items[i])
		}
	}
	return uit
}

func (d *topoDing) naam(taal string) string {
	if taal == "nl" {
		return d.NL
	}
	return d.EN
}

func (s *topoStad) naam(taal string) string {
	if taal == "nl" {
		return s.NL
	}
	return s.EN
}

// gewoon: zonder hoofdletters, accenten, streepjes en spaties, zodat "oekraine" en
// "Oekraïne" allebei goed zijn.
func gewoon(s string) string {
	return strings.NewReplacer("-", "", " ", "", "'", "", ".", "").Replace(zoekVorm(strings.TrimSpace(s)))
}

func topoGoed(antwoord string, opties ...string) bool {
	a := gewoon(antwoord)
	if a == "" {
		return false
	}
	for _, o := range opties {
		if gewoon(o) == a {
			return true
		}
	}
	return false
}

/* ---------------- tekenen ---------------- */

// topoVerhouding: breedte gedeeld door hoogte van de kaart.
func topoVerhouding(kaart string) float64 {
	k := topoKaartVan(kaart)
	cosL := math.Cos((k.Kader[1] + k.Kader[3]) / 2 * math.Pi / 180)
	return (k.Kader[2] - k.Kader[0]) * cosL / (k.Kader[3] - k.Kader[1])
}

// topoMaat: hoeveel tekens breed een kaart van h regels hoog is (zodat hij niet uitgerekt wordt).
func topoMaat(kaart string, h int) int {
	return int(math.Ceil(float64(h*4) * topoVerhouding(kaart) / 2))
}

// topoZoom: een uitsnede rond een ding, met de buurt eromheen, die precies b × h tekens vult.
// Zo heeft een provincie of een klein land veel meer puntjes dan op de hele kaart.
func topoZoom(kaart string, d *topoDing, b, h int) [4]float64 {
	k := topoKaartVan(kaart)
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	neem := func(pt [2]float64) {
		minX, maxX = math.Min(minX, pt[0]), math.Max(maxX, pt[0])
		minY, maxY = math.Min(minY, pt[1]), math.Max(maxY, pt[1])
	}
	for _, r := range d.Vlak {
		for _, pt := range r {
			neem(pt)
		}
	}
	for _, l := range d.Lijn {
		for _, pt := range l {
			neem(pt)
		}
	}
	if d.Punt != nil {
		neem(*d.Punt)
	}
	if d.Stad != nil {
		neem(d.Stad.Punt)
	}
	if math.IsInf(minX, 1) {
		return k.Kader
	}
	// hoe ver je minstens uitzoomt, zodat de buurt herkenbaar blijft (in graden breedte)
	minBreedte := map[string]float64{"nederland": 1.1, "europa": 10, "wereld": 40}[kaart]
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	cosL := math.Cos(cy * math.Pi / 180)
	r := float64(b*2) / float64(h*4) // breedte gedeeld door hoogte, in puntjes
	lat := math.Max(minBreedte, math.Max((maxY-minY)*2.2, (maxX-minX)*cosL*2.2/r))
	lon := lat * r / cosL
	heelLon, heelLat := k.Kader[2]-k.Kader[0], k.Kader[3]-k.Kader[1]
	if lat >= heelLat && lon >= heelLon {
		return k.Kader
	}
	// binnen de kaart blijven: liever opschuiven dan buiten de gegevens kijken
	schuif := func(c, half, lo, hi float64) float64 {
		if 2*half >= hi-lo {
			return (lo + hi) / 2
		}
		return math.Min(math.Max(c, lo+half), hi-half)
	}
	cx = schuif(cx, lon/2, k.Kader[0], k.Kader[2])
	cy = schuif(cy, lat/2, k.Kader[1], k.Kader[3])
	return [4]float64{cx - lon/2, cy - lat/2, cx + lon/2, cy + lat/2}
}

// topoPassend: breedte en hoogte voor een kaart die in b × h tekens moet passen.
func topoPassend(kaart string, b, h int) (int, int) {
	if w := topoMaat(kaart, h); w <= b {
		return w, h
	}
	return b, max(int(math.Ceil(float64(b*2)/(4*topoVerhouding(kaart)))), 6)
}

var (
	topoCacheMu sync.Mutex
	topoCache   = map[string]string{}
)

// de kleuren van de kaart: zee en het ding dat gevraagd wordt
const (
	topoZee  = "#2f6db5"
	topoDoel = "#ff8c1a"
)

// isWater: meren en zeeën die als vlak op de kaart staan (het IJsselmeer).
func (d *topoDing) isWater() bool { return strings.HasPrefix(d.Code, "water:") }

// topoTekst tekent een kaart van b × h tekens, zoals een atlas in de terminal: de zee blauw
// ingekleurd, het land donker met heldere grenzen, rivieren blauw, en het ding dat gevraagd
// wordt fel ingekleurd. Met namen zet hij bij rondkijken de namen op de kaart. Met braille-
// puntjes: 2 bij 4 per teken, zodat kusten en grenzen fijn genoeg zijn.
// Onthouden per tekening: tekenen is werk voor een netbook.
func topoTekst(kaart, code string, stad, namen bool, b, h int, p palet, taal string, kader [4]float64) string {
	sleutel := fmt.Sprintf("%s|%s|%v|%v|%d|%d|%s|%s|%v|%.3f", kaart, code, stad, namen, b, h, p.acc, taal, alleenLetters(), kader)
	topoCacheMu.Lock()
	if s, ok := topoCache[sleutel]; ok {
		topoCacheMu.Unlock()
		return s
	}
	topoCacheMu.Unlock()

	k := topoKaartVan(kaart)
	lonMin, latMin, lonMax, latMax := kader[0], kader[1], kader[2], kader[3]
	cosL := math.Cos((latMin + latMax) / 2 * math.Pi / 180) // een graad lengte is korter dan een graad breedte
	W, H := b*2, h*4
	schaal := math.Min(float64(W)/((lonMax-lonMin)*cosL), float64(H)/(latMax-latMin))
	naar := func(pt [2]float64) (float64, float64) {
		return (pt[0] - lonMin) * cosL * schaal, (latMax - pt[1]) * schaal
	}
	naarRing := func(r [][2]float64) [][2]float64 {
		uit := make([][2]float64, len(r))
		for i, pt := range r {
			x, y := naar(pt)
			uit[i] = [2]float64{x, y}
		}
		return uit
	}
	nieuw := func() [][]bool {
		v := make([][]bool, H)
		for y := range v {
			v[y] = make([]bool, W)
		}
		return v
	}
	land, water, grens, grijsGrens, rivier, doel := nieuw(), nieuw(), nieuw(), nieuw(), nieuw(), nieuw()
	vul := func(v [][]bool, r [][2]float64) { vulRing(v, naarRing(r)) }
	lijn := func(v [][]bool, r [][2]float64, dicht bool) { tekenLijn(v, naarRing(r), dicht) }

	for _, r := range k.Grijs {
		vul(land, r)
		lijn(grijsGrens, r, true)
	}
	for _, r := range k.Achtergrond {
		vul(land, r)
		lijn(grens, r, true)
	}
	var gekozen *topoDing
	for i := range k.Items {
		d := &k.Items[i]
		if d.Code == code {
			gekozen = d
		}
		if d.Groep == "werelddeel" {
			continue // werelddelen liggen over de landen heen: alleen inkleuren als ze gevraagd worden
		}
		for _, r := range d.Vlak {
			if d.isWater() {
				vul(water, r)
			} else {
				vul(land, r)
			}
			lijn(grens, r, true)
		}
	}
	for _, l := range k.Lijnen {
		lijn(rivier, l, false)
	}
	doelX, doelY := -1, -1
	if gekozen != nil {
		for _, r := range gekozen.Vlak {
			vul(doel, r)
		}
		gevuld := 0
		for y := range doel {
			for x := range doel[y] {
				if doel[y][x] {
					gevuld++
				}
			}
		}
		if gevuld < 6 && len(gekozen.Vlak) > 0 { // heel klein (Luxemburg, Schiermonnikoog): een stip erbij
			var sx, sy, n float64
			for _, r := range gekozen.Vlak {
				for _, pt := range r {
					sx, sy, n = sx+pt[0], sy+pt[1], n+1
				}
			}
			x, y := naar([2]float64{sx / n, sy / n})
			doelX, doelY = int(x/2), int(y/4)
		}
		for _, l := range gekozen.Lijn { // een rivier: dik
			for _, d := range [][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				rr := naarRing(l)
				for i := range rr {
					rr[i][0] += d[0]
					rr[i][1] += d[1]
				}
				tekenLijn(doel, rr, false)
			}
		}
		if gekozen.Punt != nil {
			x, y := naar(*gekozen.Punt)
			doelX, doelY = int(x/2), int(y/4)
		}
	}
	stadX, stadY := -1, -1
	if stad && gekozen != nil && gekozen.Stad != nil {
		x, y := naar(gekozen.Stad.Punt)
		stadX, stadY = int(x/2), int(y/4)
	}

	// per teken: de puntjes van één laag, in de kleur van die laag
	type teken struct {
		r      rune
		kleur  string
		achter string // achtergrond (alleen op de tekstconsole)
	}
	const (
		zeeKleur    = topoZee
		grensKleur  = "#e8e8e8"
		grijsKleur  = "#6b6b6b"
		rivierKleur = "#5aa9ff"
		doelKleur   = topoDoel
		stadKleur   = "#ffd43b"
		naamKleur   = "#ffd43b"
		landKleur   = "#101010" // alleen op de tekstconsole: zwart land
	)
	bits := [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	vak := make([][]teken, h)
	for cy := 0; cy < h; cy++ {
		vak[cy] = make([]teken, b)
		for cx := 0; cx < b; cx++ {
			var nD, nG, nGr, nR, nZ int
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					x, y := cx*2+dx, cy*4+dy
					bit := bits[dy][dx]
					zee := !land[y][x] || water[y][x]
					if doel[y][x] {
						nD |= bit
					}
					if grens[y][x] {
						nG |= bit
					}
					if grijsGrens[y][x] {
						nGr |= bit
					}
					if rivier[y][x] && !zee {
						nR |= bit
					}
					if zee {
						nZ |= bit
					}
				}
			}
			t := teken{r: ' '}
			switch {
			case nD != 0:
				t = teken{rune(0x2800 + (nD | nG)), doelKleur, ""}
			case nG != 0:
				t = teken{rune(0x2800 + nG), grensKleur, ""}
			case nR != 0:
				t = teken{rune(0x2800 + nR), rivierKleur, ""}
			case nGr != 0:
				t = teken{rune(0x2800 + nGr), grijsKleur, ""}
			case nZ != 0:
				t = teken{rune(0x2800 + nZ), zeeKleur, ""}
			}
			if alleenLetters() {
				// De Linux-console kent geen braille, wel halve blokjes: een teken wordt twee
				// vlakjes boven elkaar, elk in de kleur van wat daar ligt (zee, land, grens...).
				soort := func(dy0 int) string {
					var d, g, gr, r, z int
					for dy := dy0; dy < dy0+2; dy++ {
						for dx := 0; dx < 2; dx++ {
							x, y := cx*2+dx, cy*4+dy
							zee := !land[y][x] || water[y][x]
							switch {
							case doel[y][x]:
								d++
							case grens[y][x]:
								g++
							case rivier[y][x] && !zee:
								r++
							case grijsGrens[y][x]:
								gr++
							case zee:
								z++
							}
						}
					}
					switch {
					case d > 0:
						return doelKleur
					case g > 0:
						return grensKleur
					case r > 0:
						return rivierKleur
					case gr > 0:
						return grijsKleur
					case z >= 2:
						return zeeKleur
					}
					return landKleur
				}
				// alleen de 8 basiskleuren werken op de console overal als achtergrond
				console := map[string]string{zeeKleur: "#000080", landKleur: "#000000", grensKleur: "#c0c0c0",
					grijsKleur: "#c0c0c0", rivierKleur: "#008080", doelKleur: "#808000"}
				boven, onder := console[soort(0)], console[soort(2)]
				t = teken{'▀', boven, onder}
				if boven == onder {
					t = teken{' ', "", onder}
				}
			}
			vak[cy][cx] = t
		}
	}
	zetTekst := func(x, y int, tekst, kleur string) {
		for i, r := range []rune(tekst) {
			if y >= 0 && y < h && x+i >= 0 && x+i < b {
				if r == '●' && alleenLetters() {
					r = '*' // niet elk consolelettertype heeft een stip
				}
				vak[y][x+i] = teken{r, kleur, ""}
			}
		}
	}
	if doelX >= 0 {
		zetTekst(doelX, doelY, "●", doelKleur)
	}
	if stadX >= 0 {
		zetTekst(stadX, stadY, "●", stadKleur)
	}
	if namen {
		// namen op de kaart, zoals in een atlas: eerst het gekozen ding, dan de rest als er plek is
		bezet := make([][]bool, h)
		for y := range bezet {
			bezet[y] = make([]bool, b)
		}
		zetNaam := func(d *topoDing, kleur string) {
			var x, y float64
			switch {
			case d.Punt != nil:
				x, y = naar(*d.Punt)
			case len(d.Vlak) > 0:
				x, y = naar(middenVan(grootsteRing(d.Vlak)))
			case len(d.Lijn) > 0:
				l := d.Lijn[0]
				x, y = naar(l[len(l)/2])
			default:
				return
			}
			if x < 0 || y < 0 || x >= float64(W) || y >= float64(H) {
				return // buiten beeld (ingezoomd)
			}
			naam := d.naam(taal)
			cx, cy := int(x/2)-len([]rune(naam))/2, int(y/4)
			if d.Punt != nil {
				cx, cy = int(x/2)+1, int(y/4) // naast de stip
			}
			cx = max(min(cx, b-len([]rune(naam))), 0)
			if cy < 0 || cy >= h {
				return
			}
			for i := -1; i <= len([]rune(naam)); i++ {
				if cx+i >= 0 && cx+i < b && bezet[cy][cx+i] {
					return // past niet: liever geen naam dan namen door elkaar
				}
			}
			zetTekst(cx, cy, naam, kleur)
			for i := range []rune(naam) {
				bezet[cy][cx+i] = true
			}
		}
		if gekozen != nil {
			zetNaam(gekozen, doelKleur)
		}
		for i := range k.Items {
			d := &k.Items[i]
			if d != gekozen && gekozen != nil && d.Groep == gekozen.Groep && d.Punt == nil {
				zetNaam(d, naamKleur)
			}
		}
	}
	var regels []string
	for y := 0; y < h; y++ {
		var r kleurRegel
		for x := 0; x < b; x++ {
			t := vak[y][x]
			r.zet(t.kleur, t.achter, t.kleur == doelKleur || t.kleur == stadKleur, string(t.r))
		}
		regels = append(regels, r.klaar())
	}
	uit := strings.Join(regels, "\n")
	topoCacheMu.Lock()
	topoCache[sleutel] = uit
	topoCacheMu.Unlock()
	return uit
}

// vulRing kleurt de puntjes waarvan het midden in de ring ligt (per regel de snijpunten).
func vulRing(v [][]bool, r [][2]float64) {
	if len(r) < 3 {
		return
	}
	minY, maxY := r[0][1], r[0][1]
	for _, pt := range r {
		minY, maxY = math.Min(minY, pt[1]), math.Max(maxY, pt[1])
	}
	var xs []float64
	for y := max(int(minY), 0); y <= min(int(maxY)+1, len(v)-1); y++ {
		fy := float64(y) + 0.5
		xs = xs[:0]
		for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
			if (r[i][1] > fy) != (r[j][1] > fy) {
				xs = append(xs, r[i][0]+(fy-r[i][1])*(r[j][0]-r[i][0])/(r[j][1]-r[i][1]))
			}
		}
		sort.Float64s(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			for x := max(int(math.Ceil(xs[k]-0.5)), 0); x < min(int(math.Ceil(xs[k+1]-0.5)), len(v[y])); x++ {
				v[y][x] = true
			}
		}
	}
}

// tekenLijn zet de puntjes langs een lijn (of de rand van een ring, als dicht).
func tekenLijn(v [][]bool, r [][2]float64, dicht bool) {
	zet := func(x, y int) {
		if y >= 0 && y < len(v) && x >= 0 && x < len(v[y]) {
			v[y][x] = true
		}
	}
	stuk := func(a, b [2]float64) {
		n := int(math.Max(math.Abs(b[0]-a[0]), math.Abs(b[1]-a[1]))) + 1
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(n)
			zet(int(a[0]+(b[0]-a[0])*t), int(a[1]+(b[1]-a[1])*t))
		}
	}
	for i := 1; i < len(r); i++ {
		stuk(r[i-1], r[i])
	}
	if dicht && len(r) > 2 {
		stuk(r[len(r)-1], r[0])
	}
}

func grootsteRing(rr [][][2]float64) [][2]float64 {
	var beste [][2]float64
	opp := -1.0
	for _, r := range rr {
		a := 0.0
		for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
			a += r[j][0]*r[i][1] - r[i][0]*r[j][1]
		}
		if math.Abs(a) > opp {
			opp, beste = math.Abs(a), r
		}
	}
	return beste
}

// middenVan: het zwaartepunt van een ring (een goede plek voor de naam).
func middenVan(r [][2]float64) [2]float64 {
	var a, cx, cy float64
	for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
		f := r[j][0]*r[i][1] - r[i][0]*r[j][1]
		a += f
		cx += (r[j][0] + r[i][0]) * f
		cy += (r[j][1] + r[i][1]) * f
	}
	if a == 0 {
		return r[0]
	}
	return [2]float64{cx / (3 * a), cy / (3 * a)}
}

/* ---------------- oefenen ---------------- */

type topoStaat struct {
	bezig      bool // false: kiezen wat je oefent
	heel       bool // de hele kaart in plaats van ingezoomd (Tab wisselt)
	keuze      int  // in topoOefeningen
	kijk       int  // bij rondkijken: welk ding
	ronde      []*topoDing
	nr         int
	invoer     textinput.Model
	uitslag    string
	goed       bool
	klaar      bool // nagekeken: Enter gaat verder
	gehad      map[string]bool
	lastig     []string
	rondeKlaar bool
}

func nieuweTopo() *topoStaat {
	t := &topoStaat{}
	t.invoer = textinput.New()
	t.invoer.CharLimit = 40
	t.invoer.Prompt = "➜ "
	return t
}

func (t *topoStaat) oefening() topoOefening { return topoOefeningen[t.keuze] }

func (m *model) topoStart() tea.Cmd {
	t := m.lr.topo
	t.bezig, t.kijk = true, 0
	o := t.oefening()
	if o.vraag == "kijk" {
		return nil
	}
	per := map[string]*topoDing{}
	var codes []string
	for _, d := range o.dingen() {
		per[d.Code] = d
		codes = append(codes, d.Code)
	}
	t.ronde = nil
	for _, c := range laadLeren().kies(o.set(), codes, 10, time.Now()) {
		t.ronde = append(t.ronde, per[c])
	}
	t.nr, t.gehad, t.lastig, t.rondeKlaar = 0, map[string]bool{}, nil, false
	t.uitslag, t.klaar = "", false
	t.invoer.SetValue("")
	return t.invoer.Focus()
}

func (m *model) topoNakijken() {
	t := m.lr.topo
	o := t.oefening()
	d := t.ronde[t.nr]
	taal := m.inst.Taal
	var goed bool
	var juist string
	if o.vraag == "stad" {
		goed = topoGoed(t.invoer.Value(), append([]string{d.Stad.NL, d.Stad.EN}, d.Stad.Ook...)...)
		juist = d.Stad.naam(taal)
	} else {
		goed = topoGoed(t.invoer.Value(), append([]string{d.NL, d.EN}, d.Ook...)...)
		juist = d.naam(taal)
	}
	if !t.gehad[d.Code] {
		t.gehad[d.Code] = true
		ls := laadLeren()
		ls.zet(o.set(), d.Code, goed, time.Now())
		ls.bewaar()
	}
	t.goed, t.klaar = goed, true
	if goed {
		t.uitslag = m.t("topo.right", juist)
		return
	}
	t.uitslag = m.t("topo.wrong", juist)
	t.lastig = append(t.lastig, juist)
	if len(t.ronde) < 14 {
		t.ronde = append(t.ronde, d)
	}
}

func (m *model) toetsTopo(msg tea.KeyMsg) (tea.Cmd, bool) {
	t := m.lr.topo
	k := msg.String()
	if !t.bezig {
		switch k {
		case "up", "k":
			t.keuze = max(t.keuze-1, 0)
		case "down", "j":
			t.keuze = min(t.keuze+1, len(topoOefeningen)-1)
		case "enter":
			return m.topoStart(), true
		case "esc":
			m.lr.scherm = lerenMenu
		default:
			return nil, false
		}
		return nil, true
	}
	if k == "tab" { // ingezoomd of de hele kaart
		t.heel = !t.heel
		return nil, true
	}
	o := t.oefening()
	if o.vraag == "kijk" {
		n := len(o.dingen())
		switch k {
		case "right", "down", "l", "j", " ":
			t.kijk = (t.kijk + 1) % n
		case "left", "up", "h", "k":
			t.kijk = (t.kijk + n - 1) % n
		case "esc":
			t.bezig = false
		default:
			return nil, false
		}
		return nil, true
	}
	if t.rondeKlaar {
		switch k {
		case "enter", " ":
			return m.topoStart(), true
		case "esc":
			t.invoer.Blur()
			t.bezig = false
		}
		return nil, true
	}
	switch k {
	case "esc":
		t.invoer.Blur()
		t.bezig = false
		return nil, true
	case "enter":
		switch {
		case t.klaar:
			t.nr++
			t.klaar, t.uitslag = false, ""
			t.invoer.SetValue("")
			if t.nr >= len(t.ronde) {
				t.rondeKlaar = true
			}
		case strings.TrimSpace(t.invoer.Value()) != "":
			m.topoNakijken()
		}
		return nil, true
	}
	if t.klaar {
		return nil, true
	}
	var cmd tea.Cmd
	t.invoer, cmd = t.invoer.Update(msg)
	return cmd, true
}

// topoVraag: de vraag bij een ding, passend bij wat het is.
func (m model) topoVraag(o topoOefening, d *topoDing) string {
	taal := m.inst.Taal
	if o.vraag == "stad" {
		return m.t("topo.whichcapital", d.naam(taal))
	}
	switch {
	case d.Groep == "land":
		return m.t("topo.whichcountry")
	case d.Groep == "werelddeel":
		return m.t("topo.whichcontinent")
	case d.Groep == "natuur" && len(d.Lijn) == 0:
		return m.t("topo.whichnature")
	case d.Groep == "provincie":
		return m.t("topo.whichprovince")
	case d.Groep == "stad":
		return m.t("topo.whichcity")
	case len(d.Lijn) > 0:
		return m.t("topo.whichriver")
	case strings.HasPrefix(d.Code, "eiland:"):
		return m.t("topo.whichisland")
	}
	return m.t("topo.whichwater")
}

func (m model) viewTopo(p palet, kop string, breed int) string {
	t := m.lr.topo
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	taal := m.inst.Taal
	if !t.bezig {
		l := laadLeren()
		var regels []string
		vorige := ""
		for i, o := range topoOefeningen {
			if o.kaart != vorige {
				if vorige != "" {
					regels = append(regels, "")
				}
				regels = append(regels, lipgloss.NewStyle().Bold(true).Render(m.t("topo.map."+o.kaart)))
				vorige = o.kaart
			}
			naam := fmt.Sprintf("%-38s", m.t(o.sleutel))
			extra := dim.Render("  " + m.t("topo.look.d"))
			if o.vraag != "kijk" {
				var codes []string
				for _, d := range o.dingen() {
					codes = append(codes, d.Code)
				}
				extra = dim.Render(fmt.Sprintf("  %d/%d", l.kent(o.set(), codes), len(codes)))
			}
			if i == t.keuze {
				regels = append(regels, acc.Render("▸ "+naam)+extra)
			} else {
				regels = append(regels, "  "+naam+extra)
			}
		}
		regels = append(regels, "", dim.Render(m.t("dictee.legend")))
		return kop + "\n\n" + paneel(p, m.t("learn.topo"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.pickkeys"))
	}
	o := t.oefening()
	var ding *topoDing
	switch {
	case o.vraag == "kijk":
		alle := o.dingen()
		ding = alle[t.kijk%len(alle)]
	case !t.rondeKlaar:
		ding = t.ronde[t.nr]
	}
	// de hele breedte en hoogte: een regel voor de vraag, een voor het antwoord, een voor de toetsen
	W, H := m.breed-4, max(m.hoog-4, 8)
	var kaart string
	if ding != nil && !t.heel {
		kader := topoZoom(o.kaart, ding, W, H)
		kaart = topoTekst(o.kaart, ding.Code, o.vraag != "naam", o.vraag == "kijk", W, H, p, taal, kader)
	} else {
		b, h := topoPassend(o.kaart, W, H)
		code := ""
		if ding != nil {
			code = ding.Code
		}
		kaart = lipgloss.PlaceHorizontal(W, lipgloss.Center,
			topoTekst(o.kaart, code, o.vraag != "naam", o.vraag == "kijk", b, h, p, taal, topoKaartVan(o.kaart).Kader))
		kaart = lipgloss.PlaceVertical(H, lipgloss.Top, kaart)
	}
	kop = acc.Render(m.t("learn.topo")) + dim.Render(" · "+m.t("topo.map."+o.kaart)) + "   "
	onder := ""
	switch {
	case o.vraag == "kijk":
		kop += lipgloss.NewStyle().Foreground(lipgloss.Color(topoDoel)).Bold(true).Render(ding.naam(taal))
		if ding.Stad != nil {
			kop += "   " + m.t("topo.capital") + " " + lipgloss.NewStyle().Foreground(p.geel).Render(ding.Stad.naam(taal))
		}
		kop += dim.Render(fmt.Sprintf("   %d/%d", t.kijk+1, len(o.dingen())))
	case t.rondeKlaar:
		kop += acc.Render(m.t("dictee.done"))
		if len(t.lastig) == 0 {
			onder = m.t("dictee.allright")
		} else {
			onder = m.t("dictee.tricky") + " " + strings.Join(t.lastig, ", ")
		}
	default:
		kop += dim.Render(m.t("learn.count", t.nr+1, len(t.ronde))+"   ") + lipgloss.NewStyle().Bold(true).Render(m.topoVraag(o, ding))
		onder = t.invoer.View()
		if t.uitslag != "" {
			kleur := p.groen
			if !t.goed {
				kleur = p.geel
			}
			onder += "   " + lipgloss.NewStyle().Foreground(kleur).Render(t.uitslag)
		}
	}
	toetsen := m.t("topo.keys")
	switch {
	case o.vraag == "kijk":
		toetsen = m.t("topo.lookkeys")
	case t.rondeKlaar:
		toetsen = m.t("dictee.donekeys")
	case t.klaar:
		toetsen = m.t("topo.nextkeys")
	}
	zoom := m.t("topo.zoomout")
	if t.heel {
		zoom = m.t("topo.zoomin")
	}
	if !t.rondeKlaar {
		toetsen = zoom + " · " + toetsen
	}
	return kop + "\n" + kaart + "\n" + onder + "\n" + dim.Render(toetsen)
}
