package main

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Geschiedenis: de tien tijdvakken van de basisschool, als tijdlijn om doorheen te bladeren, en
// drie manieren van oefenen: in welk tijdvak hoort dit, welk jaartal hoort hierbij, en wat
// was eerder. De leermotor laat terugkomen wat lastig is.

type gebeurtenis struct {
	tekst string
	jaar  int  // negatief: voor Christus
	rond  bool // ongeveer: dan niet vragen naar het precieze jaartal
}

type tijdvak struct {
	naam, naamEN string
	van, tot     int
	uitleg       string
	gebeurd      []gebeurtenis
}

var tijdvakken = []tijdvak{
	{"jagers en boeren", "hunters and farmers", -10000, -3000,
		"Na de laatste ijstijd trokken mensen rond om te jagen en te verzamelen. Later gingen ze op één plek wonen en werden ze boer: ze hielden vee en verbouwden graan. In Drenthe bouwden ze hunebedden van enorme zwerfstenen.",
		[]gebeurtenis{{"Het einde van de laatste ijstijd", -10000, true}, {"De eerste boeren wonen in Limburg", -5300, true}, {"In Drenthe worden hunebedden gebouwd", -3400, true}}},
	{"Grieken en Romeinen", "Greeks and Romans", -3000, 500,
		"De Grieken bedachten veel over wetenschap, toneel en democratie. De Romeinen bouwden een enorm rijk, met wegen, steden en badhuizen. De Rijn werd de grens van hun rijk, dwars door Nederland.",
		[]gebeurtenis{{"De eerste Olympische Spelen in Griekenland", -776, false}, {"Julius Caesar verovert Gallië", -50, true}, {"De Rijn wordt de grens van het Romeinse Rijk", 47, true}, {"Nijmegen wordt een Romeinse stad", 100, true}, {"Het West-Romeinse Rijk houdt op te bestaan", 476, false}}},
	{"monniken en ridders", "monks and knights", 500, 1000,
		"Monniken brachten het christendom naar onze streken en schreven boeken over met de hand. Karel de Grote regeerde over een groot rijk. Rijke heren lieten boeren voor zich werken, in ruil voor bescherming.",
		[]gebeurtenis{{"Willibrord wordt bisschop van Utrecht", 695, false}, {"Bonifatius wordt vermoord bij Dokkum", 754, false}, {"Karel de Grote wordt keizer", 800, false}, {"Vikingen overvallen Dorestad", 834, false}}},
	{"steden en staten", "towns and states", 1000, 1500,
		"Steden groeiden, met markten, gilden en stadsmuren. Kooplieden uit verschillende steden werkten samen in de Hanze. Graven en hertogen kregen steeds meer macht.",
		[]gebeurtenis{{"Amsterdam krijgt stadsrechten", 1300, true}, {"Graaf Floris V wordt vermoord", 1296, false}, {"Steden werken samen in de Hanze", 1350, true}, {"De Sint-Elisabethsvloed", 1421, false}, {"De boekdrukkunst wordt uitgevonden", 1450, true}}},
	{"ontdekkers en hervormers", "explorers and reformers", 1500, 1600,
		"Europeanen voeren over de oceanen naar nieuwe werelddelen. Luther wilde de kerk veranderen en de kerk splitste in katholieken en protestanten. In de Nederlanden kwamen mensen in opstand tegen de Spaanse koning.",
		[]gebeurtenis{{"Luther begint de Reformatie", 1517, false}, {"Het eerste schip vaart rond de wereld", 1522, false}, {"De Beeldenstorm", 1566, false}, {"Het begin van de Tachtigjarige Oorlog", 1568, false}, {"Leiden wordt ontzet", 1574, false}, {"Willem van Oranje wordt vermoord in Delft", 1584, false}}},
	{"regenten en vorsten", "regents and monarchs", 1600, 1700,
		"Dit was de Gouden Eeuw van de Republiek: rijke handel, grote schilders en mooie grachtenpanden. Maar de rijkdom kwam ook uit oorlog, kolonies en slavenhandel. Rijke regenten bestuurden de steden.",
		[]gebeurtenis{{"De VOC wordt opgericht", 1602, false}, {"De WIC wordt opgericht", 1621, false}, {"Rembrandt schildert De Nachtwacht", 1642, false}, {"De Vrede van Münster: einde van de Tachtigjarige Oorlog", 1648, false}, {"Het Rampjaar", 1672, false}}},
	{"pruiken en revoluties", "wigs and revolutions", 1700, 1800,
		"Rijke mensen droegen pruiken en volgden de mode uit Frankrijk. Steeds meer mensen vonden dat het volk mee moest beslissen. In Amerika en Frankrijk kwamen revoluties.",
		[]gebeurtenis{{"De Verenigde Staten verklaren zich onafhankelijk", 1776, false}, {"De patriotten willen meer inspraak", 1785, true}, {"De Franse Revolutie begint", 1789, false}, {"De Bataafse Republiek", 1795, false}}},
	{"burgers en stoommachines", "citizens and steam engines", 1800, 1900,
		"Machines op stoom veranderden het werk: er kwamen fabrieken en treinen. Nederland werd een koninkrijk met een grondwet. Kinderarbeid werd beperkt en de slavernij werd afgeschaft.",
		[]gebeurtenis{{"Nederland wordt een koninkrijk", 1815, false}, {"De eerste trein rijdt van Amsterdam naar Haarlem", 1839, false}, {"De nieuwe grondwet van Thorbecke", 1848, false}, {"De slavernij in Suriname en op de Antillen wordt afgeschaft", 1863, false}, {"Het Kinderwetje van Van Houten", 1874, false}}},
	{"wereldoorlogen", "world wars", 1900, 1950,
		"Twee keer was er een wereldoorlog. In de Tweede Wereldoorlog werd Nederland bezet door Duitsland en werden miljoenen Joden vermoord. Na de oorlog moest het land weer worden opgebouwd.",
		[]gebeurtenis{{"De Eerste Wereldoorlog begint", 1914, false}, {"Vrouwen krijgen stemrecht", 1919, false}, {"De Afsluitdijk is klaar", 1932, false}, {"Duitsland valt Nederland binnen", 1940, false}, {"Anne Frank duikt onder", 1942, false}, {"Nederland wordt bevrijd", 1945, false}, {"Indonesië wordt onafhankelijk", 1949, false}}},
	{"televisie en computer", "television and computer", 1950, 2026,
		"Bijna iedereen kreeg een televisie, en later een computer en een telefoon met internet. Na de Watersnoodramp werden de Deltawerken gebouwd. Europese landen gingen steeds meer samenwerken.",
		[]gebeurtenis{{"De Watersnoodramp", 1953, false}, {"De eerste mens loopt op de maan", 1969, false}, {"Suriname wordt onafhankelijk", 1975, false}, {"De Berlijnse Muur valt", 1989, false}, {"De Deltawerken zijn klaar", 1997, false}, {"De euro wordt ons geld", 2002, false}}},
}

// jaarTekst: 1648, of 50 v.Chr., met "rond" ervoor als het ongeveer is.
func jaarTekst(j int, rond bool, taal string) string {
	s := strconv.Itoa(j)
	if j < 0 {
		s = strconv.Itoa(-j) + " v.Chr."
		if taal != "nl" {
			s = strconv.Itoa(-j) + " BC"
		}
	}
	if rond {
		if taal == "nl" {
			return "rond " + s
		}
		return "around " + s
	}
	return s
}

func (t tijdvak) jaren(taal string) string {
	tot := jaarTekst(t.tot, false, taal)
	if t.tot >= 2026 {
		tot = map[bool]string{true: "nu", false: "now"}[taal == "nl"]
	}
	return jaarTekst(t.van, false, taal) + " - " + tot
}

func (t tijdvak) naamIn(taal string) string {
	if taal == "nl" {
		return t.naam
	}
	return t.naamEN
}

type gebeurtenisMet struct {
	gebeurtenis
	vak int // 0..9
}

func alleGebeurtenissen() []gebeurtenisMet {
	var uit []gebeurtenisMet
	for i, t := range tijdvakken {
		for _, g := range t.gebeurd {
			uit = append(uit, gebeurtenisMet{g, i})
		}
	}
	return uit
}

/* ---------------- oefenen ---------------- */

const (
	geschTijdlijn = iota
	geschVak
	geschJaar
	geschEerder
)

type geschStaat struct {
	bezig      bool
	keuze      int // geschTijdlijn ..
	vak        int // tijdlijn: welk tijdvak
	ronde      []gebeurtenisMet
	paren      [][2]gebeurtenisMet
	nr         int
	invoer     textinput.Model
	nagekeken  bool
	goed       bool
	uitslag    string
	gehad      map[string]bool
	lastig     []string
	rondeKlaar bool
}

func nieuweGeschiedenis() *geschStaat {
	g := &geschStaat{}
	g.invoer = textinput.New()
	g.invoer.CharLimit = 12
	g.invoer.Prompt = "➜ "
	return g
}

func geschSet(keuze int) string { return fmt.Sprintf("gesch:%d", keuze) }

func (m *model) geschStart() tea.Cmd {
	g := m.lr.gesch
	g.bezig, g.nr, g.nagekeken, g.uitslag, g.rondeKlaar = true, 0, false, "", false
	g.gehad, g.lastig = map[string]bool{}, nil
	g.invoer.SetValue("")
	alle := alleGebeurtenissen()
	switch g.keuze {
	case geschTijdlijn:
		return nil
	case geschEerder:
		// twee gebeurtenissen met minstens tien jaar ertussen: wat was eerder?
		g.paren = nil
		for len(g.paren) < 10 {
			a, b := alle[rand.IntN(len(alle))], alle[rand.IntN(len(alle))]
			if a.jaar-b.jaar >= 10 || b.jaar-a.jaar >= 10 {
				g.paren = append(g.paren, [2]gebeurtenisMet{a, b})
			}
		}
		return nil
	}
	per := map[string]gebeurtenisMet{}
	var sleutels []string
	for _, x := range alle {
		if g.keuze == geschJaar && x.rond {
			continue // naar een jaartal dat niemand precies weet vragen we niet
		}
		per[x.tekst] = x
		sleutels = append(sleutels, x.tekst)
	}
	g.ronde = nil
	for _, s := range laadLeren().kies(geschSet(g.keuze), sleutels, 10, time.Now()) {
		g.ronde = append(g.ronde, per[s])
	}
	return g.invoer.Focus()
}

func (g *geschStaat) aantal() int {
	if g.keuze == geschEerder {
		return len(g.paren)
	}
	return len(g.ronde)
}

func (m *model) geschNakijken(antwoord string) {
	g := m.lr.gesch
	taal := m.inst.Taal
	var goed bool
	var sleutel, juist string
	switch g.keuze {
	case geschVak:
		x := g.ronde[g.nr]
		n, err := strconv.Atoi(strings.TrimSpace(antwoord))
		goed = err == nil && n == x.vak+1
		sleutel, juist = x.tekst, m.t("hist.invak", x.vak+1, tijdvakken[x.vak].naamIn(taal), jaarTekst(x.jaar, x.rond, taal))
	case geschJaar:
		x := g.ronde[g.nr]
		a := strings.ToLower(strings.TrimSpace(antwoord))
		voor := strings.Contains(a, "v") || strings.Contains(a, "bc") || strings.HasPrefix(a, "-")
		n, err := strconv.Atoi(strings.Trim(strings.Fields(a + " x")[0], "-"))
		if voor {
			n = -n
		}
		goed = err == nil && n == x.jaar
		sleutel, juist = x.tekst, m.t("hist.year", jaarTekst(x.jaar, false, taal), tijdvakken[x.vak].naamIn(taal))
	case geschEerder:
		p := g.paren[g.nr]
		eerste := 1
		if p[1].jaar < p[0].jaar {
			eerste = 2
		}
		goed = antwoord == strconv.Itoa(eerste)
		juist = m.t("hist.order", p[0].tekst, jaarTekst(p[0].jaar, p[0].rond, taal), p[1].tekst, jaarTekst(p[1].jaar, p[1].rond, taal))
	}
	if sleutel != "" && !g.gehad[sleutel] {
		g.gehad[sleutel] = true
		l := laadLeren()
		l.zet(geschSet(g.keuze), sleutel, goed, time.Now())
		l.bewaar()
	}
	g.nagekeken, g.goed = true, goed
	if goed {
		g.uitslag = m.t("hist.right", juist)
		return
	}
	g.uitslag = juist
	if sleutel != "" {
		g.lastig = append(g.lastig, sleutel)
		if len(g.ronde) < 14 {
			g.ronde = append(g.ronde, g.ronde[g.nr])
		}
	}
}

func (m *model) toetsGeschiedenis(msg tea.KeyMsg) (tea.Cmd, bool) {
	g := m.lr.gesch
	k := msg.String()
	if !g.bezig {
		switch k {
		case "up", "k":
			g.keuze = max(g.keuze-1, 0)
		case "down", "j":
			g.keuze = min(g.keuze+1, geschEerder)
		case "enter":
			return m.geschStart(), true
		case "esc":
			m.lr.scherm = lerenMenu
		default:
			return nil, false
		}
		return nil, true
	}
	if g.keuze == geschTijdlijn {
		switch k {
		case "right", "l", " ":
			g.vak = min(g.vak+1, len(tijdvakken)-1)
		case "left", "h":
			g.vak = max(g.vak-1, 0)
		case "esc":
			g.bezig = false
		default:
			return nil, false
		}
		return nil, true
	}
	if k == "esc" {
		g.invoer.Blur()
		g.bezig = false
		return nil, true
	}
	if g.rondeKlaar {
		if k == "enter" || k == " " {
			return m.geschStart(), true
		}
		return nil, true
	}
	if g.nagekeken {
		if k == "enter" || k == " " {
			g.nr++
			g.nagekeken, g.uitslag = false, ""
			g.invoer.SetValue("")
			if g.nr >= g.aantal() {
				g.rondeKlaar = true
			}
		}
		return nil, true
	}
	if g.keuze == geschEerder {
		if k == "1" || k == "2" {
			m.geschNakijken(k)
		}
		return nil, true
	}
	if k == "enter" {
		if strings.TrimSpace(g.invoer.Value()) != "" {
			m.geschNakijken(g.invoer.Value())
		}
		return nil, true
	}
	var cmd tea.Cmd
	g.invoer, cmd = g.invoer.Update(msg)
	return cmd, true
}

/* ---------------- weergave ---------------- */

// tijdbalk: de tien tijdvakken op een rij, het gekozen tijdvak licht op.
func (m model) tijdbalk(p palet, gekozen int) string {
	var delen []string
	for i := range tijdvakken {
		vak := fmt.Sprintf(" %d ", i+1)
		if i == gekozen {
			delen = append(delen, lipgloss.NewStyle().Background(p.acc).Foreground(p.bg).Bold(true).Render(vak))
		} else {
			delen = append(delen, lipgloss.NewStyle().Foreground(p.dim).Render(vak))
		}
	}
	return strings.Join(delen, lipgloss.NewStyle().Foreground(p.lijn).Render("─"))
}

func (m model) viewGeschiedenis(p palet, kop string, breed int) string {
	g := m.lr.gesch
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	tekst := lipgloss.NewStyle().Width(breed - 6)
	taal := m.inst.Taal
	if !g.bezig {
		l := laadLeren()
		var regels []string
		for i, sl := range []string{"hist.timeline", "hist.whichvak", "hist.whichyear", "hist.earlier"} {
			naam := fmt.Sprintf("%-30s", m.t(sl))
			extra := dim.Render("  " + m.t(sl+".d"))
			if i == geschVak || i == geschJaar {
				var s []string
				for _, x := range alleGebeurtenissen() {
					if i == geschVak || !x.rond {
						s = append(s, x.tekst)
					}
				}
				extra = dim.Render(fmt.Sprintf("  %d/%d", l.kent(geschSet(i), s), len(s)))
			}
			if i == g.keuze {
				regels = append(regels, acc.Render("▸ "+naam)+extra)
			} else {
				regels = append(regels, "  "+naam+extra)
			}
		}
		regels = append(regels, "", dim.Render(m.t("dictee.legend")))
		return kop + "\n\n" + paneel(p, m.t("learn.history"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.pickkeys"))
	}
	if g.keuze == geschTijdlijn {
		t := tijdvakken[g.vak]
		regels := []string{m.tijdbalk(p, g.vak), "", acc.Render(fmt.Sprintf("%d. %s", g.vak+1, m.t("hist.timeof", t.naamIn(taal)))) + "   " + dim.Render(t.jaren(taal)), "",
			tekst.Render(t.uitleg), ""}
		for _, x := range t.gebeurd {
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render(fmt.Sprintf("%14s", jaarTekst(x.jaar, x.rond, taal)))+"  "+x.tekst)
		}
		return kop + "\n\n" + paneel(p, m.t("hist.timeline"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("hist.timelinekeys"))
	}
	if g.rondeKlaar {
		regels := []string{acc.Render(m.t("dictee.done")), ""}
		if len(g.lastig) == 0 {
			regels = append(regels, m.t("hist.allright"))
		} else {
			regels = append(regels, m.t("dictee.tricky"))
			for _, x := range g.lastig {
				regels = append(regels, "  "+x)
			}
		}
		return kop + "\n\n" + paneel(p, m.t("learn.history"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.donekeys"))
	}
	regels := []string{dim.Render(m.t("learn.count", g.nr+1, g.aantal())), ""}
	toetsen := m.t("topo.keys")
	switch g.keuze {
	case geschVak:
		regels = append(regels, lipgloss.NewStyle().Bold(true).Render(g.ronde[g.nr].tekst), "", m.t("hist.askvak"), "")
		for i, t := range tijdvakken {
			regels = append(regels, dim.Render(fmt.Sprintf("  %2d  %s", i+1, t.naamIn(taal))))
		}
		regels = append(regels, "", g.invoer.View())
	case geschJaar:
		x := g.ronde[g.nr]
		regels = append(regels, lipgloss.NewStyle().Bold(true).Render(x.tekst), "", m.t("hist.askyear", tijdvakken[x.vak].naamIn(taal)), "", g.invoer.View())
	case geschEerder:
		pr := g.paren[g.nr]
		regels = append(regels, m.t("hist.askearlier"), "", "  1  "+pr[0].tekst, "  2  "+pr[1].tekst)
		toetsen = m.t("hist.earlierkeys")
	}
	if g.nagekeken {
		kleur := p.groen
		if !g.goed {
			kleur = p.geel
		}
		regels = append(regels, "", tekst.Foreground(kleur).Render(g.uitslag))
		toetsen = m.t("topo.nextkeys")
	}
	return kop + "\n\n" + paneel(p, m.t("learn.history"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(toetsen)
}
