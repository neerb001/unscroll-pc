package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Vandaag: het weer met een praktische tip, zon en maan, het seizoen en het volgende jaarfeest.
// Het weer komt van Open-Meteo (gratis, geen sleutel, geen account). Al het andere rekent de app zelf uit.

/* ---------------- weer ---------------- */

type weerData struct {
	Opgehaald time.Time `json:"opgehaald"`
	Current   struct {
		Temperature float64 `json:"temperature_2m"`
		Apparent    float64 `json:"apparent_temperature"`
		WeatherCode int     `json:"weather_code"`
		Wind        float64 `json:"wind_speed_10m"`
		IsDay       int     `json:"is_day"`
	} `json:"current"`
	Hourly struct {
		Time        []string  `json:"time"`
		Temperature []float64 `json:"temperature_2m"`
		Neerslag    []float64 `json:"precipitation"`
		Kans        []float64 `json:"precipitation_probability"`
	} `json:"hourly"`
	Daily struct {
		Time        []string  `json:"time"`
		WeatherCode []int     `json:"weather_code"`
		Max         []float64 `json:"temperature_2m_max"`
		Min         []float64 `json:"temperature_2m_min"`
		Neerslag    []float64 `json:"precipitation_sum"`
		Kans        []float64 `json:"precipitation_probability_max"`
		Op          []string  `json:"sunrise"`
		Onder       []string  `json:"sunset"`
		Daglicht    []float64 `json:"daylight_duration"`
	} `json:"daily"`
}

// weerBasis is los te zetten voor tests.
var weerBasis = "https://api.open-meteo.com/v1/forecast"

func weerCache() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "unscroll", "weer.json")
}

func laadWeerCache() *weerData {
	var w weerData
	if data, err := os.ReadFile(weerCache()); err == nil && json.Unmarshal(data, &w) == nil && len(w.Daily.Time) > 0 {
		return &w
	}
	return nil
}

type weerMsg struct {
	w   *weerData
	err error
}

func haalWeer(o Ouders) tea.Cmd {
	lat, lon := o.plek()
	return func() tea.Msg {
		q := url.Values{
			"latitude": {fmt.Sprint(lat)}, "longitude": {fmt.Sprint(lon)}, "timezone": {"auto"}, "forecast_days": {"2"},
			"current": {"temperature_2m,apparent_temperature,weather_code,wind_speed_10m,is_day"},
			"hourly":  {"temperature_2m,precipitation,precipitation_probability"},
			"daily":   {"weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,precipitation_probability_max,sunrise,sunset,daylight_duration"},
		}
		c := http.Client{Timeout: 15 * time.Second}
		resp, err := c.Get(weerBasis + "?" + q.Encode())
		if err != nil {
			return weerMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return weerMsg{err: fmt.Errorf("open-meteo: %s", resp.Status)}
		}
		var w weerData
		if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
			return weerMsg{err: err}
		}
		w.Opgehaald = time.Now()
		_ = os.MkdirAll(filepath.Dir(weerCache()), 0o755)
		if data, err := json.Marshal(w); err == nil {
			_ = veiligSchrijven(weerCache(), data, 0o644)
		}
		return weerMsg{w: &w}
	}
}

// plek: de coördinaten uit ouders.yaml, anders De Bilt (midden van Nederland).
func (o Ouders) plek() (float64, float64) {
	if o.Vandaag.Breedtegraad == 0 && o.Vandaag.Lengtegraad == 0 {
		return 52.10, 5.18
	}
	return o.Vandaag.Breedtegraad, o.Vandaag.Lengtegraad
}

func (o Ouders) plaatsNaam() string {
	if o.Vandaag.Plaats != "" {
		return o.Vandaag.Plaats
	}
	if o.Vandaag.Breedtegraad == 0 && o.Vandaag.Lengtegraad == 0 {
		return "De Bilt"
	}
	return ""
}

// weerSoort zet een WMO-weercode om naar een sleutel voor de vertaling.
func weerSoort(code int) string {
	switch {
	case code == 0:
		return "clear"
	case code == 1:
		return "mostlyclear"
	case code == 2:
		return "partly"
	case code == 3:
		return "cloudy"
	case code == 45 || code == 48:
		return "fog"
	case code >= 51 && code <= 57:
		return "drizzle"
	case code >= 61 && code <= 67:
		return "rain"
	case code >= 71 && code <= 77:
		return "snow"
	case code >= 80 && code <= 82:
		return "showers"
	case code == 85 || code == 86:
		return "snowshowers"
	case code >= 95:
		return "thunder"
	}
	return "unknown"
}

// komendeUren geeft de regen (mm) en kans (%) voor de komende uren vanaf nu.
func (w *weerData) komendeUren(nu time.Time, aantal int) (uren []string, mm, kans []float64) {
	h := w.Hourly
	for i, t := range h.Time {
		tijd, err := time.ParseInLocation("2006-01-02T15:04", t, nu.Location())
		if err != nil || tijd.Add(time.Hour).Before(nu) {
			continue
		}
		if len(uren) == aantal || i >= len(h.Neerslag) {
			break
		}
		uren = append(uren, tijd.Format("15"))
		mm = append(mm, h.Neerslag[i])
		k := 0.0
		if i < len(h.Kans) {
			k = h.Kans[i]
		}
		kans = append(kans, k)
	}
	return
}

// weerTips geeft praktische tips voor buiten, als vertaalsleutels.
func (w *weerData) weerTips(nu time.Time) []string {
	var tips []string
	_, mm, kans := w.komendeUren(nu, 12)
	regen := false
	for i := range mm {
		if mm[i] >= 0.3 || kans[i] >= 60 {
			regen = true
		}
	}
	soort := weerSoort(w.Current.WeatherCode)
	max := w.Current.Temperature
	if len(w.Daily.Max) > 0 {
		max = w.Daily.Max[0]
	}
	switch {
	case soort == "thunder":
		tips = append(tips, "today.tip.thunder")
	case soort == "snow" || soort == "snowshowers":
		tips = append(tips, "today.tip.snow")
	case regen:
		tips = append(tips, "today.tip.rain")
	}
	switch {
	case w.Current.Apparent <= 2:
		tips = append(tips, "today.tip.freezing")
	case w.Current.Apparent < 10:
		tips = append(tips, "today.tip.cold")
	case max >= 25:
		tips = append(tips, "today.tip.hot")
	}
	if w.Current.Wind >= 40 {
		tips = append(tips, "today.tip.wind")
	}
	if soort == "fog" {
		tips = append(tips, "today.tip.fog")
	}
	if len(tips) == 0 && !regen {
		tips = append(tips, "today.tip.nice")
	}
	return tips
}

/* ---------------- maan, seizoen, jaarfeesten ---------------- */

const maanMaand = 29.530588853

// maanStand geeft de leeftijd van de maan in dagen en het verlichte deel (0..1).
func maanStand(t time.Time) (leeftijd, verlicht float64) {
	nieuw := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC) // een bekende nieuwe maan
	dagen := t.Sub(nieuw).Hours() / 24
	leeftijd = math.Mod(dagen, maanMaand)
	if leeftijd < 0 {
		leeftijd += maanMaand
	}
	verlicht = (1 - math.Cos(2*math.Pi*leeftijd/maanMaand)) / 2
	return
}

var maanTekens = []string{"●", "◐", "◐", "◐", "○", "◑", "◑", "◑"}

func maanFase(t time.Time) int {
	leeftijd, _ := maanStand(t)
	return int(math.Floor(leeftijd/maanMaand*8+0.5)) % 8
}

func dagenTotVolleMaan(t time.Time) int {
	leeftijd, _ := maanStand(t)
	tot := maanMaand/2 - leeftijd
	if tot < 0 {
		tot += maanMaand
	}
	return int(math.Round(tot))
}

// pasen: de Gregoriaanse paasdatum (anonieme rekenregel).
func pasen(jaar int) time.Time {
	a, b, c := jaar%19, jaar/100, jaar%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	maand := (h + l - 7*m + 114) / 31
	dag := (h+l-7*m+114)%31 + 1
	return time.Date(jaar, time.Month(maand), dag, 0, 0, 0, 0, time.Local)
}

type jaarfeest struct {
	sleutel string
	datum   time.Time
}

func jaarfeesten(jaar int) []jaarfeest {
	d := func(m time.Month, dag int) time.Time { return time.Date(jaar, m, dag, 0, 0, 0, 0, time.Local) }
	p := pasen(jaar)
	kerst := d(time.December, 25)
	advent := kerst.AddDate(0, 0, -int(kerst.Weekday())-21) // vierde zondag voor Kerst
	if kerst.Weekday() == time.Sunday {
		advent = kerst.AddDate(0, 0, -28)
	}
	return []jaarfeest{
		{"epiphany", d(time.January, 6)}, {"candlemas", d(time.February, 2)}, {"easter", p},
		{"ascension", p.AddDate(0, 0, 39)}, {"pentecost", p.AddDate(0, 0, 49)}, {"stjohn", d(time.June, 24)},
		{"michaelmas", d(time.September, 29)}, {"martinmas", d(time.November, 11)}, {"advent", advent},
		{"stnicholas", d(time.December, 5)}, {"christmas", kerst},
	}
}

// volgendeFeesten: de eerstvolgende jaarfeesten vanaf vandaag (vandaag telt mee).
func volgendeFeesten(nu time.Time, aantal int) []jaarfeest {
	vandaag := time.Date(nu.Year(), nu.Month(), nu.Day(), 0, 0, 0, 0, time.Local)
	var uit []jaarfeest
	for _, j := range []int{nu.Year(), nu.Year() + 1} {
		for _, f := range jaarfeesten(j) {
			if !f.datum.Before(vandaag) && len(uit) < aantal {
				uit = append(uit, f)
			}
		}
	}
	return uit
}

func dagenTot(nu, dan time.Time) int {
	a := time.Date(nu.Year(), nu.Month(), nu.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(dan.Year(), dan.Month(), dan.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// seizoen: het huidige seizoen (astronomisch, op hele dagen) en het begin van het volgende.
func seizoen(nu time.Time) (string, string, time.Time) {
	j := nu.Year()
	grenzen := []struct {
		naam  string
		datum time.Time
	}{
		{"winter", time.Date(j, 1, 1, 0, 0, 0, 0, time.Local)},
		{"spring", time.Date(j, 3, 20, 0, 0, 0, 0, time.Local)},
		{"summer", time.Date(j, 6, 21, 0, 0, 0, 0, time.Local)},
		{"autumn", time.Date(j, 9, 22, 0, 0, 0, 0, time.Local)},
		{"winter", time.Date(j, 12, 21, 0, 0, 0, 0, time.Local)},
		{"spring", time.Date(j+1, 3, 20, 0, 0, 0, 0, time.Local)},
	}
	for i := len(grenzen) - 2; i >= 0; i-- {
		if !nu.Before(grenzen[i].datum) {
			return grenzen[i].naam, grenzen[i+1].naam, grenzen[i+1].datum
		}
	}
	return "winter", "spring", grenzen[1].datum
}

/* ---------------- staat en weergave ---------------- */

type vandaagStaat struct {
	weer  *weerData
	fout  string
	laden bool
}

func (m *model) vandaagOpen() tea.Cmd {
	v := m.vd
	if v.weer == nil {
		v.weer = laadWeerCache()
	}
	if v.laden || (v.weer != nil && time.Since(v.weer.Opgehaald) < 30*time.Minute) {
		return nil
	}
	v.laden = true
	return haalWeer(m.ouders)
}

func (m *model) vandaagBericht(msg weerMsg) {
	v := m.vd
	v.laden = false
	if msg.err != nil {
		v.fout = m.t("listen.err", msg.err.Error())
		return
	}
	v.weer, v.fout = msg.w, ""
}

var dagNamenNL = []string{"zondag", "maandag", "dinsdag", "woensdag", "donderdag", "vrijdag", "zaterdag"}
var maandNamenNL = []string{"", "januari", "februari", "maart", "april", "mei", "juni", "juli", "augustus", "september", "oktober", "november", "december"}

func (m model) datumTekst(t time.Time) string {
	if m.inst.Taal == "nl" {
		return fmt.Sprintf("%s %d %s", dagNamenNL[t.Weekday()], t.Day(), maandNamenNL[t.Month()])
	}
	return t.Format("Monday 2 January")
}

func (m model) viewVandaag(p palet) string {
	v := m.vd
	nu := m.nu
	if nu.IsZero() {
		nu = time.Now()
	}
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	breed := m.breed - 6
	links := breed * 11 / 20
	rechts := breed - links - 5

	// kop: datum, week en dag van het jaar
	_, week := nu.ISOWeek()
	kop := lipgloss.NewStyle().Bold(true).Render(strings.ToUpper(m.datumTekst(nu)[:1])+m.datumTekst(nu)[1:]) + "   " +
		dim.Render(m.t("today.week", week, nu.YearDay()))

	// links: het weer
	var w []string
	if v.weer == nil {
		switch {
		case v.laden:
			w = append(w, dim.Render(m.t("listen.loading")))
		case v.fout != "":
			w = append(w, lipgloss.NewStyle().Foreground(p.geel).Width(links-4).Render(v.fout))
		}
	} else {
		d := v.weer
		soort := m.t("today.w." + weerSoort(d.Current.WeatherCode))
		w = append(w, acc.Render(fmt.Sprintf("%.0f°", d.Current.Temperature))+"  "+soort,
			dim.Render(m.t("today.feels", math.Round(d.Current.Apparent), math.Round(d.Current.Wind))))
		if len(d.Daily.Max) > 0 {
			w = append(w, m.t("today.minmax", math.Round(d.Daily.Min[0]), math.Round(d.Daily.Max[0])))
		}
		// regen per uur als staafjes
		uren, mm, kans := d.komendeUren(nu, 12)
		if len(uren) > 0 {
			staaf := []rune(" ▁▂▃▄▅▆▇█")
			var balk, tijden strings.Builder
			for i := range uren {
				n := int(math.Min(mm[i]/0.25, 8))
				if mm[i] > 0 && n == 0 {
					n = 1
				}
				teken := string(staaf[n])
				if n == 0 {
					teken = "·"
				}
				kleur := p.dim
				if mm[i] > 0 || kans[i] >= 50 {
					kleur = p.acc
				}
				balk.WriteString(lipgloss.NewStyle().Foreground(kleur).Render(" " + teken + " "))
				tijden.WriteString(uren[i] + " ")
			}
			w = append(w, "", m.t("today.rain12"), balk.String(), dim.Render(tijden.String()))
		}
		// tips
		w = append(w, "")
		for _, tip := range d.weerTips(nu) {
			w = append(w, lipgloss.NewStyle().Foreground(p.groen).Width(links-4).Render("› "+m.t(tip)))
		}
		// morgen
		if len(d.Daily.Max) > 1 {
			w = append(w, "", dim.Render(m.t("today.tomorrow"))+" "+m.t("today.w."+weerSoort(d.Daily.WeatherCode[1]))+
				fmt.Sprintf(" · %.0f°/%.0f°", d.Daily.Min[1], d.Daily.Max[1])+m.t("today.rainchance", math.Round(d.Daily.Kans[1])))
		}
		bij := m.t("today.updated", d.Opgehaald.Format("15:04"))
		if v.fout != "" {
			bij = m.t("today.offline", d.Opgehaald.Format("15:04"))
		}
		w = append(w, "", dim.Render(bij+" · Open-Meteo"))
	}
	weerTitel := m.t("today.weather")
	if pl := m.ouders.plaatsNaam(); pl != "" {
		weerTitel += " · " + pl
	}

	// rechts: licht, maan, seizoen, jaarfeesten
	var r []string
	if v.weer != nil && len(v.weer.Daily.Op) > 0 {
		op, onder := v.weer.Daily.Op[0], v.weer.Daily.Onder[0]
		if len(op) >= 16 && len(onder) >= 16 {
			r = append(r, m.t("today.sun", op[11:16], onder[11:16]))
		}
		if dl := v.weer.Daily.Daglicht; len(dl) > 1 {
			verschil := math.Round((dl[1] - dl[0]) / 60)
			sleutel := "today.longer"
			if verschil < 0 {
				sleutel, verschil = "today.shorter", -verschil
			}
			r = append(r, dim.Render(m.t(sleutel, verschil)))
		}
		r = append(r, "")
	}
	fase := maanFase(nu)
	_, verlicht := maanStand(nu)
	r = append(r, acc.Render(maanTekens[fase])+"  "+m.t(fmt.Sprintf("today.moon.%d", fase))+dim.Render(fmt.Sprintf(" · %d%%", int(math.Round(verlicht*100)))))
	if fase != 4 {
		r = append(r, dim.Render(m.t("today.fullmoon", dagenTotVolleMaan(nu))))
	}
	nuS, volgendS, begin := seizoen(nu)
	r = append(r, "", m.t("today.season."+nuS)+dim.Render(" · "+m.t("today.nextseason", m.t("today.season."+volgendS), dagenTot(nu, begin))))
	r = append(r, "")
	for i, f := range volgendeFeesten(nu, 2) {
		dagen := dagenTot(nu, f.datum)
		wanneer := m.t("today.indays", dagen)
		switch dagen {
		case 0:
			wanneer = m.t("today.istoday")
		case 1:
			wanneer = m.t("today.istomorrow")
		}
		naam := m.t("today.f." + f.sleutel)
		if i == 0 {
			r = append(r, acc.Render(naam)+dim.Render(" · "+wanneer),
				lipgloss.NewStyle().Width(rechts-4).Render(m.t("today.f."+f.sleutel+".d")))
		} else {
			r = append(r, "", dim.Render(m.t("today.after", naam, wanneer)))
		}
	}

	return kop + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top,
		paneel(p, weerTitel, strings.Join(w, "\n"), links, true),
		" ",
		paneel(p, m.t("today.skyyear"), strings.Join(r, "\n"), rechts, false)) + "\n" + dim.Render(m.t("today.keys"))
}
