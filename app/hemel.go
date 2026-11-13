package main

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	satellite "github.com/joshuaferrara/go-satellite"
)

// Sterrenhemel: welke planeten en sterrenbeelden je vanavond ziet, en wanneer het ISS overkomt.
// Planeten en sterrenbeelden rekent de app zelf uit (formules van Paul Schlyter); voor het ISS
// haalt hij één keer per dag de baangegevens bij Celestrak en rekent hij de overkomsten uit.

const graad = math.Pi / 180

func sinG(x float64) float64 { return math.Sin(x * graad) }
func cosG(x float64) float64 { return math.Cos(x * graad) }
func norm360(x float64) float64 {
	x = math.Mod(x, 360)
	if x < 0 {
		x += 360
	}
	return x
}

func juliaans(t time.Time) float64 {
	return float64(t.UTC().UnixNano())/86400e9 + 2440587.5
}

type baan struct{ N, Nd, i, id, w, wd, a, e, ed, M, Md float64 }

// Baanelementen (Schlyter): waarde + verandering per dag sinds 31 december 1999.
var planeetBanen = map[string]baan{
	"mercury": {48.3313, 3.24587e-5, 7.0047, 5.00e-8, 29.1241, 1.01444e-5, 0.387098, 0.205635, 5.59e-10, 168.6562, 4.0923344368},
	"venus":   {76.6799, 2.46590e-5, 3.3946, 2.75e-8, 54.8910, 1.38374e-5, 0.723330, 0.006773, -1.302e-9, 48.0052, 1.6021302244},
	"mars":    {49.5574, 2.11081e-5, 1.8497, -1.78e-8, 286.5016, 2.92961e-5, 1.523688, 0.093405, 2.516e-9, 18.6021, 0.5240207766},
	"jupiter": {100.4542, 2.76854e-5, 1.3030, -1.557e-7, 273.8777, 1.64505e-5, 5.20256, 0.048498, 4.469e-9, 19.8950, 0.0830853001},
	"saturn":  {113.6634, 2.38980e-5, 2.4886, -1.081e-7, 339.3939, 2.97661e-5, 9.55475, 0.055546, -9.499e-9, 316.9670, 0.0334442282},
}

var planeetVolgorde = []string{"venus", "jupiter", "mars", "saturn", "mercury"}

func keplerE(M, e float64) float64 {
	E := M + e/graad*sinG(M)*(1+e*cosG(M))
	for i := 0; i < 5; i++ {
		E = E - (E-e/graad*sinG(E)-M)/(1-e*cosG(E))
	}
	return E
}

// zonEcliptisch: de plek van de zon gezien vanaf de aarde (ecliptische x, y in AE).
func zonEcliptisch(d float64) (float64, float64) {
	w := 282.9404 + 4.70935e-5*d
	e := 0.016709 - 1.151e-9*d
	M := norm360(356.0470 + 0.9856002585*d)
	E := keplerE(M, e)
	xv, yv := cosG(E)-e, math.Sqrt(1-e*e)*sinG(E)
	v, r := math.Atan2(yv, xv)/graad, math.Hypot(xv, yv)
	lon := v + w
	return r * cosG(lon), r * sinG(lon)
}

// naarEquatoriaal: ecliptisch (x, y, z) naar rechte klimming en declinatie (graden).
func naarEquatoriaal(x, y, z, d float64) (ra, dec float64) {
	ecl := 23.4393 - 3.563e-7*d
	ye := y*cosG(ecl) - z*sinG(ecl)
	ze := y*sinG(ecl) + z*cosG(ecl)
	return norm360(math.Atan2(ye, x) / graad), math.Atan2(ze, math.Hypot(x, ye)) / graad
}

func zonRADec(t time.Time) (float64, float64) {
	d := juliaans(t) - 2451543.5
	xs, ys := zonEcliptisch(d)
	return naarEquatoriaal(xs, ys, 0, d)
}

func planeetRADec(naam string, t time.Time) (float64, float64) {
	d := juliaans(t) - 2451543.5
	b := planeetBanen[naam]
	N, i, w := b.N+b.Nd*d, b.i+b.id*d, b.w+b.wd*d
	e, M := b.e+b.ed*d, norm360(b.M+b.Md*d)
	E := keplerE(M, e)
	xv, yv := b.a*(cosG(E)-e), b.a*math.Sqrt(1-e*e)*sinG(E)
	v, r := math.Atan2(yv, xv)/graad, math.Hypot(xv, yv)
	xh := r * (cosG(N)*cosG(v+w) - sinG(N)*sinG(v+w)*cosG(i))
	yh := r * (sinG(N)*cosG(v+w) + cosG(N)*sinG(v+w)*cosG(i))
	zh := r * sinG(v+w) * sinG(i)
	xs, ys := zonEcliptisch(d)
	return naarEquatoriaal(xh+xs, yh+ys, zh, d)
}

// hoogteRichting: hoe hoog (graden) en in welke richting (graden vanaf noord) iets aan de hemel staat.
func hoogteRichting(ra, dec, lat, lon float64, t time.Time) (float64, float64) {
	gmst := norm360(280.46061837 + 360.98564736629*(juliaans(t)-2451545))
	ha := norm360(gmst + lon - ra)
	alt := math.Asin(sinG(lat)*sinG(dec)+cosG(lat)*cosG(dec)*cosG(ha)) / graad
	az := norm360(math.Atan2(sinG(ha), cosG(ha)*sinG(lat)-sinG(dec)/cosG(dec)*cosG(lat))/graad + 180)
	return alt, az
}

func zonHoogte(t time.Time, lat, lon float64) float64 {
	ra, dec := zonRADec(t)
	h, _ := hoogteRichting(ra, dec, lat, lon, t)
	return h
}

// avondMoment: het eerste moment na 16:00 dat het echt donker is (zon 8° onder de horizon).
func avondMoment(dag time.Time, lat, lon float64) time.Time {
	t := time.Date(dag.Year(), dag.Month(), dag.Day(), 16, 0, 0, 0, dag.Location())
	for ; t.Hour() < 23 || t.Minute() < 30; t = t.Add(10 * time.Minute) {
		if zonHoogte(t, lat, lon) < -8 {
			return t
		}
	}
	return t
}

// ochtendMoment: het laatste donkere moment voor de ochtendschemering.
func ochtendMoment(dag time.Time, lat, lon float64) time.Time {
	t := time.Date(dag.Year(), dag.Month(), dag.Day(), 9, 0, 0, 0, dag.Location()).AddDate(0, 0, 1)
	for ; t.Hour() > 2; t = t.Add(-10 * time.Minute) {
		if zonHoogte(t, lat, lon) < -8 {
			return t.Add(-20 * time.Minute)
		}
	}
	return t
}

type hemelObject struct {
	sleutel          string
	hoogte, richting float64
}

func zichtbarePlaneten(t time.Time, lat, lon float64) []hemelObject {
	var uit []hemelObject
	for _, p := range planeetVolgorde {
		ra, dec := planeetRADec(p, t)
		h, r := hoogteRichting(ra, dec, lat, lon, t)
		if h > 8 {
			uit = append(uit, hemelObject{p, h, r})
		}
	}
	return uit
}

// Sterrenbeelden: midden (rechte klimming in uren, declinatie in graden).
var sterrenbeelden = []struct {
	sleutel string
	ra, dec float64
}{
	{"ursamajor", 11, 55}, {"ursaminor", 15, 77}, {"cassiopeia", 1, 60}, {"orion", 5.6, 3}, {"taurus", 4.5, 16},
	{"gemini", 7.1, 23}, {"leo", 10.6, 15}, {"virgo", 13.3, -3}, {"bootes", 14.7, 31}, {"cygnus", 20.6, 43},
	{"lyra", 18.8, 37}, {"aquila", 19.8, 5}, {"scorpius", 16.9, -30}, {"pegasus", 22.7, 20}, {"andromeda", 0.8, 38},
	{"perseus", 3.3, 45}, {"auriga", 5.9, 42}, {"canismajor", 6.8, -22}, {"cancer", 8.6, 20}, {"draco", 17, 65},
}

func hogeSterrenbeelden(t time.Time, lat, lon float64, max int) []hemelObject {
	var uit []hemelObject
	for _, s := range sterrenbeelden {
		h, r := hoogteRichting(s.ra*15, s.dec, lat, lon, t)
		if h > 30 {
			uit = append(uit, hemelObject{s.sleutel, h, r})
		}
	}
	sort.Slice(uit, func(i, j int) bool { return uit[i].hoogte > uit[j].hoogte })
	if len(uit) > max {
		uit = uit[:max]
	}
	return uit
}

/* ---------------- ISS ---------------- */

type issOverkomst struct {
	begin, top, eind time.Time
	vanuit, naar     float64
	maxHoogte        float64
}

var celestrakISS = "https://celestrak.org/NORAD/elements/gp.php?CATNR=25544&FORMAT=tle"

func issTLE() (string, string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	pad := filepath.Join(dir, "unscroll", "iss.tle")
	if info, err := os.Stat(pad); err != nil || time.Since(info.ModTime()) > 24*time.Hour {
		c := http.Client{Timeout: 15 * time.Second}
		req, _ := http.NewRequest("GET", celestrakISS, nil)
		req.Header.Set("User-Agent", "Unscroll/"+versie)
		if resp, err := c.Do(req); err == nil {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && strings.Contains(string(data), "\n1 25544") {
				_ = os.MkdirAll(filepath.Dir(pad), 0o755)
				_ = os.WriteFile(pad, data, 0o644)
			}
		}
	}
	data, err := os.ReadFile(pad)
	if err != nil {
		return "", "", err
	}
	var l1, l2 string
	for _, r := range strings.Split(string(data), "\n") {
		r = strings.TrimRight(r, "\r ")
		if strings.HasPrefix(r, "1 ") {
			l1 = r
		} else if strings.HasPrefix(r, "2 ") {
			l2 = r
		}
	}
	if l1 == "" || l2 == "" {
		return "", "", fmt.Errorf("no ISS data")
	}
	return l1, l2, nil
}

// issOverkomsten: de zichtbare overkomsten in de komende dagen (ISS in de zon, het bij jou donker).
func issOverkomsten(l1, l2 string, vanaf time.Time, dagen int, lat, lon float64) []issOverkomst {
	sat := satellite.TLEToSat(l1, l2, satellite.GravityWGS72)
	waar := satellite.LatLong{Latitude: lat * graad, Longitude: lon * graad}
	stand := func(t time.Time) (hoogte, richting float64, inZon bool) {
		u := t.UTC()
		pos, _ := satellite.Propagate(sat, u.Year(), int(u.Month()), u.Day(), u.Hour(), u.Minute(), u.Second())
		jd := satellite.JDay(u.Year(), int(u.Month()), u.Day(), u.Hour(), u.Minute(), u.Second())
		la := satellite.ECIToLookAngles(pos, waar, 0, jd)
		// in de schaduw van de aarde? (eenvoudige cilinder)
		ra, dec := zonRADec(t)
		zx, zy, zz := cosG(dec)*cosG(ra), cosG(dec)*sinG(ra), sinG(dec)
		langs := pos.X*zx + pos.Y*zy + pos.Z*zz
		loodrecht := math.Sqrt(math.Max(pos.X*pos.X+pos.Y*pos.Y+pos.Z*pos.Z-langs*langs, 0))
		inZon = langs > 0 || loodrecht > 6371
		return la.El / graad, la.Az / graad, inZon
	}
	var uit []issOverkomst
	var huidig *issOverkomst
	zichtbaar := false
	for t := vanaf; t.Before(vanaf.AddDate(0, 0, dagen)); t = t.Add(20 * time.Second) {
		h, r, zon := stand(t)
		if h > 10 {
			if huidig == nil {
				huidig = &issOverkomst{begin: t, vanuit: r}
				zichtbaar = false
			}
			if h > huidig.maxHoogte {
				huidig.maxHoogte, huidig.top = h, t
			}
			huidig.eind, huidig.naar = t, r
			if zon && zonHoogte(t, lat, lon) < -6 {
				zichtbaar = true
			}
		} else if huidig != nil {
			if zichtbaar && huidig.maxHoogte > 15 {
				uit = append(uit, *huidig)
			}
			huidig = nil
		}
	}
	return uit
}

/* ---------------- scherm ---------------- */

type hemelStaat struct {
	kaart   bool
	uitleg  bool
	keuze   int
	scroll  int
	iss     []issOverkomst
	issFout bool
	laden   bool
}

type issMsg struct {
	o   []issOverkomst
	err error
}

func (m *model) hemelOpen() tea.Cmd {
	h := m.od.hemel
	if h.laden {
		return nil
	}
	h.laden = true
	lat, lon := m.ouders.plek()
	return func() tea.Msg {
		l1, l2, err := issTLE()
		if err != nil {
			return issMsg{err: err}
		}
		return issMsg{o: issOverkomsten(l1, l2, time.Now(), 5, lat, lon)}
	}
}

func (m *model) hemelBericht(msg issMsg) {
	h := m.od.hemel
	h.laden = false
	h.iss, h.issFout = msg.o, msg.err != nil
}

var windRichtingen = map[string][]string{
	"nl": {"noorden", "noordoosten", "oosten", "zuidoosten", "zuiden", "zuidwesten", "westen", "noordwesten"},
	"en": {"north", "northeast", "east", "southeast", "south", "southwest", "west", "northwest"},
}

func (m model) richting(az float64) string {
	r := windRichtingen["en"]
	if m.inst.Taal == "nl" {
		r = windRichtingen["nl"]
	}
	return r[int(math.Round(norm360(az)/45))%8]
}

func (m model) hoogteWoord(h float64) string {
	switch {
	case h < 20:
		return m.t("sky.low")
	case h < 50:
		return m.t("sky.mid")
	}
	return m.t("sky.high")
}

func (m model) viewHemel(p palet) string {
	h := m.od.hemel
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	nu := time.Now()
	lat, lon := m.ouders.plek()
	breed := m.breed - 6
	links := breed * 11 / 20
	rechts := breed - links - 5
	tekst := func(w int, s string) string { return lipgloss.NewStyle().Width(w).Render(s) }

	avond := avondMoment(nu, lat, lon)
	var l []string
	l = append(l, acc.Render(m.t("sky.tonight", avond.Format("15:04"))))
	planeten := zichtbarePlaneten(avond, lat, lon)
	if len(planeten) == 0 {
		l = append(l, dim.Render(m.t("sky.noplanets")))
	}
	for _, pl := range planeten {
		l = append(l, "• "+lipgloss.NewStyle().Bold(true).Render(m.t("sky.p."+pl.sleutel))+dim.Render(" · "+m.t("sky.where", m.hoogteWoord(pl.hoogte), m.richting(pl.richting))),
			"  "+tekst(links-6, dim.Render(m.t("sky.p."+pl.sleutel+".d"))))
	}
	ochtend := ochtendMoment(nu, lat, lon)
	vroeg := zichtbarePlaneten(ochtend, lat, lon)
	if len(vroeg) > 0 {
		var namen []string
		for _, pl := range vroeg {
			namen = append(namen, m.t("sky.p."+pl.sleutel)+" ("+m.richting(pl.richting)+")")
		}
		l = append(l, "", dim.Render(m.t("sky.morning", ochtend.Format("15:04")))+" "+strings.Join(namen, ", "))
	}
	l = append(l, "", acc.Render(m.t("sky.constellations")))
	for _, s := range hogeSterrenbeelden(avond, lat, lon, 4) {
		l = append(l, "• "+lipgloss.NewStyle().Bold(true).Render(m.t("sky.c."+s.sleutel))+dim.Render(" · "+m.t("sky.where", m.hoogteWoord(s.hoogte), m.richting(s.richting))),
			"  "+tekst(links-6, dim.Render(m.t("sky.c."+s.sleutel+".d"))))
	}

	var r []string
	r = append(r, acc.Render(m.t("sky.iss")), tekst(rechts-4, m.t("sky.issexplain")), "")
	switch {
	case h.laden:
		r = append(r, dim.Render(m.t("listen.loading")))
	case h.issFout:
		r = append(r, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("sky.issoffline")))
	case len(h.iss) == 0:
		r = append(r, dim.Render(m.t("sky.issnone")))
	default:
		r = append(r, m.t("sky.isspasses"))
		for i, o := range h.iss {
			if i == 4 {
				break
			}
			dag := m.datumTekst(o.begin)
			r = append(r, "", lipgloss.NewStyle().Bold(true).Render(dag+" "+o.begin.Format("15:04")),
				tekst(rechts-4, dim.Render(m.t("sky.isspath", m.richting(o.vanuit), m.richting(o.naar), math.Round(o.maxHoogte), int(o.eind.Sub(o.begin).Minutes()+0.5)))))
		}
	}
	plaats := m.ouders.plaatsNaam()
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.discover")) + "  " + dim.Render(m.t("sky.title")+" · "+plaats)
	if h.uitleg {
		return m.viewHemelUitleg(p, kop, breed)
	}
	if h.kaart {
		hoogte := max(m.hoog-8, 12)
		kaartBreed := min(breed-4, hoogte*2+20)
		kaart := m.hemelKaart(p, avond, lat, lon, kaartBreed, hoogte)
		return kop + "\n\n" + paneel(p, m.t("sky.maptitle", avond.Format("15:04")), lipgloss.PlaceHorizontal(breed-4, lipgloss.Center, kaart), breed, true) +
			"\n" + dim.Render(m.t("sky.mapkeys"))
	}
	return kop + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top,
		paneel(p, m.t("sky.planets"), strings.Join(l, "\n"), links, true), " ",
		paneel(p, "ISS", strings.Join(r, "\n"), rechts, false)) + "\n" + dim.Render(m.t("sky.keys"))
}

/* ---------------- maan en hemelkaart ---------------- */

// maanRADec: de plek van de maan (Schlyter, zonder storingen: nauwkeurig tot een paar graden).
func maanRADec(t time.Time) (float64, float64) {
	d := juliaans(t) - 2451543.5
	N := 125.1228 - 0.0529538083*d
	i, w := 5.1454, 318.0634+0.1643573223*d
	a, e, M := 60.2666, 0.054900, norm360(115.3654+13.0649929509*d)
	E := keplerE(M, e)
	xv, yv := a*(cosG(E)-e), a*math.Sqrt(1-e*e)*sinG(E)
	v, r := math.Atan2(yv, xv)/graad, math.Hypot(xv, yv)
	x := r * (cosG(N)*cosG(v+w) - sinG(N)*sinG(v+w)*cosG(i))
	y := r * (sinG(N)*cosG(v+w) + cosG(N)*sinG(v+w)*cosG(i))
	z := r * sinG(v+w) * sinG(i)
	return naarEquatoriaal(x, y, z, d)
}

var planeetTekens = map[string]string{"mercury": "☿", "venus": "♀", "mars": "♂", "jupiter": "♃", "saturn": "♄"}

// hemelKaart tekent de hemel als koepel: de rand is de horizon, het midden recht boven je.
// Noord boven, oost links (zoals je naar boven kijkt met je gezicht naar het zuiden: kaart boven je hoofd).
func (m model) hemelKaart(p palet, t time.Time, lat, lon float64, breed, hoog int) string {
	type punt struct {
		x, y  int
		tekst string
		kleur lipgloss.TerminalColor
		vet   bool
	}
	rx, ry := float64(breed/2-2), float64(hoog/2-1)
	plaats := func(h, az float64) (int, int) {
		r := (90 - h) / 90
		return breed/2 - int(math.Round(r*rx*sinG(az))), hoog/2 - int(math.Round(r*ry*cosG(az)))
	}
	vak := make([][]string, hoog)
	for y := range vak {
		vak[y] = make([]string, breed)
		for x := range vak[y] {
			vak[y][x] = " "
		}
	}
	zet := func(x, y int, s string, stijl lipgloss.Style) {
		for i, c := range []rune(s) {
			if y >= 0 && y < hoog && x+i >= 0 && x+i < breed {
				vak[y][x+i] = stijl.Render(string(c))
			}
		}
	}
	dim := lipgloss.NewStyle().Foreground(p.lijn)
	// horizon
	for a := 0.0; a < 360; a += 3 {
		x, y := plaats(0, a)
		zet(x, y, "·", dim)
	}
	wind := windRichtingen["en"]
	if m.inst.Taal == "nl" {
		wind = windRichtingen["nl"]
	}
	for i, a := range []float64{0, 90, 180, 270} {
		x, y := plaats(-6, a)
		letter := strings.ToUpper(wind[i*2][:1])
		zet(x, y, letter, lipgloss.NewStyle().Foreground(p.dim).Bold(true))
	}
	// sterrenbeelden (naam op hun plek)
	for _, s := range sterrenbeelden {
		h, az := hoogteRichting(s.ra*15, s.dec, lat, lon, t)
		if h > 5 {
			x, y := plaats(h, az)
			naam := m.t("sky.c." + s.sleutel)
			zet(x-len([]rune(naam))/2, y, naam, lipgloss.NewStyle().Foreground(p.dim))
		}
	}
	// maan
	ra, dec := maanRADec(t)
	if h, az := hoogteRichting(ra, dec, lat, lon, t); h > 0 {
		x, y := plaats(h, az)
		_, verlicht := maanStand(t)
		zet(x, y, maanTekens[maanFase(t)]+fmt.Sprintf(" %s %d%%", m.t("sky.moon"), int(verlicht*100)), lipgloss.NewStyle().Foreground(p.fg).Bold(true))
	}
	// planeten
	kleuren := map[string]lipgloss.Color{"venus": "#fff3b0", "jupiter": "#f0c27a", "mars": "#ff6b4a", "saturn": "#e6d27a", "mercury": "#c0c0c0"}
	for _, pl := range planeetVolgorde {
		ra, dec := planeetRADec(pl, t)
		if h, az := hoogteRichting(ra, dec, lat, lon, t); h > 0 {
			x, y := plaats(h, az)
			zet(x, y, planeetTekens[pl]+" "+m.t("sky.p."+pl), lipgloss.NewStyle().Foreground(kleuren[pl]).Bold(true))
		}
	}
	// recht boven je
	zet(breed/2, hoog/2, "+", dim)
	var b strings.Builder
	for _, rij := range vak {
		b.WriteString(strings.Join(rij, "") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
