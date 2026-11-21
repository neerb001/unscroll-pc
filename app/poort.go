package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Eerst oefenen, dan de computer. Ouders kunnen instellen dat het kind elke dag eerst een
// aantal sommen goed moet hebben (uit een tafel naar keuze) voordat de rest van de computer
// opengaat. Alleen goede antwoorden tellen; de voortgang staat groot in beeld.
//
// Eén keer per dag, of (als ouders dat instellen) elke zoveel minuten opnieuw, zodat het kind
// blijft oefenen. Na een ronde gaat het kind terug naar waar het was. Bewaard, zodat een
// herstart niets kost. Niet tijdens het avondslot
// (dan mag alleen nog lezen en luisteren, en dat moet zonder sommen kunnen). Ouders kunnen er
// altijd langs met o en de pincode.

type poortVoortgang struct {
	Datum     string    `json:"datum"`
	Goed      int       `json:"goed"`       // goede sommen in deze ronde
	OpenSinds time.Time `json:"open_sinds"` // wanneer de laatste ronde klaar was
}

// verlopen: de vorige ronde is klaar, maar het is tijd voor een nieuwe.
func (v poortVoortgang) verlopen(e eerstOefenen, nu time.Time) bool {
	return e.ElkeMinuten > 0 && v.Goed >= e.Sommen && !v.OpenSinds.IsZero() &&
		nu.Sub(v.OpenSinds) >= time.Duration(e.ElkeMinuten)*time.Minute
}

// stand: de voortgang van de ronde die nu loopt (een nieuwe ronde begint bij 0).
func (m *model) poortStand() poortVoortgang {
	v := laadPoort(m.nu)
	if v.verlopen(m.ouders.EerstOefenen, m.nu) {
		v.Goed = 0
	}
	return v
}

func poortPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "eerst-oefenen.json") }

func laadPoort(nu time.Time) poortVoortgang {
	var v poortVoortgang
	if data, err := os.ReadFile(poortPad()); err == nil {
		_ = json.Unmarshal(data, &v)
	}
	if d := nu.Format("2006-01-02"); v.Datum != d {
		v = poortVoortgang{Datum: d}
	}
	return v
}

func (v poortVoortgang) bewaar() {
	_ = os.MkdirAll(filepath.Dir(poortPad()), 0o755)
	data, _ := json.Marshal(v)
	_ = veiligSchrijven(poortPad(), data, 0o644)
}

type poortStaat struct {
	som      tafelSom
	antwoord string
	terug    string // wat er net gebeurde: goed, of het goede antwoord
	vanTab   int    // waar het kind was toen een nieuwe ronde begon: daar gaat het daarna heen
	goed     bool
	klaar    time.Time // net klaar: even het feestje laten zien
}

// poortDicht: moet het kind nu eerst oefenen?
func (m *model) poortDicht() bool {
	e := m.ouders.EerstOefenen
	if e.Sommen <= 0 || m.ouderModus || m.dicht() {
		return false
	}
	return m.poortStand().Goed < e.Sommen
}

func (m *model) poortNieuweSom() {
	pt := m.poort
	tafel := m.ouders.EerstOefenen.Tafel
	for {
		t := tafel
		if t < 1 || t > maxTafel {
			t = 1 + rand.IntN(10) // tafels 1 tot en met 10 door elkaar
		}
		s := tafelSom{a: 1 + rand.IntN(10), b: t}
		if s != pt.som {
			pt.som, pt.antwoord = s, ""
			return
		}
	}
}

func (m *model) toetsPoort(msg tea.KeyMsg) tea.Cmd {
	pt := m.poort
	if pt.som.b == 0 {
		m.poortNieuweSom()
	}
	k := msg.String()
	switch {
	case k == "o": // ouders: naar Systeem voor de pincode
		m.tab = tabSysteem
		m.pinOpen = true
		return m.pin.Focus()
	case k == "backspace":
		if pt.antwoord != "" {
			pt.antwoord = pt.antwoord[:len(pt.antwoord)-1]
		}
	case msg.Type == tea.KeyRunes && strings.Trim(k, "0123456789") == "":
		pt.antwoord = (pt.antwoord + k)[:min(len(pt.antwoord+k), 3)]
	case k == "enter":
		if pt.antwoord == "" {
			return nil
		}
		n, _ := strconv.Atoi(pt.antwoord)
		v := m.poortStand()
		tv := laadTafels() // telt ook mee bij de tafels zelf
		if n == pt.som.uitkomst() {
			v.Goed++
			pt.goed, pt.terug = true, m.t("gate.right")
			tv.Reeks[pt.som.sleutel()]++
			missieGoed()
		} else {
			pt.goed, pt.terug = false, m.t("gate.wrong", pt.som.tekst(), pt.som.uitkomst())
			tv.Reeks[pt.som.sleutel()] = 0
		}
		v.bewaar()
		tv.bewaar()
		if v.Goed >= m.ouders.EerstOefenen.Sommen {
			pt.klaar = time.Now()
			v.OpenSinds = m.nu
			v.bewaar()
			m.zegLang(m.t("gate.open"))
			m.poortNieuweSom()
			if terug := pt.vanTab; terug != tabStart && terug != tabSysteem {
				pt.vanTab = tabStart
				return m.naarTab(terug) // terug naar waar het kind was
			}
			return nil
		}
		m.poortNieuweSom()
	}
	return nil
}

// voortgangsbalk: ●●●○○ en een balk, zodat het kind ziet hoe ver het is.
func voortgangsbalk(p palet, goed, totaal, breed int) string {
	vol := breed * goed / max(totaal, 1)
	balk := lipgloss.NewStyle().Foreground(p.groen).Render(strings.Repeat("█", vol)) +
		lipgloss.NewStyle().Foreground(p.lijn).Render(strings.Repeat("░", breed-vol))
	if totaal <= 20 {
		return lipgloss.NewStyle().Foreground(p.groen).Render(strings.Repeat("● ", goed)) +
			lipgloss.NewStyle().Foreground(p.lijn).Render(strings.Repeat("○ ", totaal-goed)) + "\n\n" + balk
	}
	return balk
}

func (m model) viewPoort(p palet) string {
	pt := m.poort
	if pt.som.b == 0 {
		mm := &m
		mm.poortNieuweSom()
	}
	e := m.ouders.EerstOefenen
	goed := m.poortStand().Goed
	dim := lipgloss.NewStyle().Foreground(p.dim)
	wat := m.t("tables.one", fmt.Sprint(e.Tafel))
	if e.Tafel < 1 || e.Tafel > maxTafel {
		wat = m.t("gate.mixed")
	}
	groot := lipgloss.NewStyle().Bold(true).Foreground(p.fg).Render(pt.som.tekst()+" = ") +
		lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(pt.antwoord+"▌")
	regels := []string{
		lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(m.t("gate.title", m.naamOfKind())),
		dim.Render(m.t("gate.explain", e.Sommen, wat)),
		"",
		"      " + groot,
		"",
	}
	if pt.terug != "" {
		kleur := p.groen
		if !pt.goed {
			kleur = p.geel
		}
		regels = append(regels, lipgloss.NewStyle().Foreground(kleur).Render(pt.terug))
	} else {
		regels = append(regels, "")
	}
	regels = append(regels, "", voortgangsbalk(p, goed, e.Sommen, min(40, m.breed-30)), "",
		m.t("gate.count", goed, e.Sommen))
	breed := min(m.breed-8, 70)
	return lipgloss.Place(m.breed-4, m.hoog-4, lipgloss.Center, lipgloss.Center,
		paneel(p, m.t("gate.panel"), strings.Join(regels, "\n"), breed, true)+"\n"+dim.Render(m.t("gate.keys")))
}
