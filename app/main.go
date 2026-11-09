// Unscroll: een rustige luister- en ontdekcomputer voor kinderen.
// Draait als volledig-scherm terminal-app op een oude 32-bit netbook.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var versie = "dev" // wordt bij het bouwen gezet met -ldflags

const (
	tabStart = iota
	tabLuisteren
	tabLezen
	tabMaatje
	tabOntdekken
	tabDenksport
	tabSchrijfhoek
	tabVandaag
	tabSysteem
)

var tabNamen = []string{"start", "luister", "lees", "maatje", "ontdek", "denk", "schrijf", "vandaag", "systeem"}

type menuItem struct {
	tab        int
	naam, info string
}

func menu(maatje string) []menuItem {
	return []menuItem{
		{tabLuisteren, "Luisteren", "muziek, Spotify, luisterboeken, podcasts, radio"},
		{tabLezen, "Lezen", "e-boeken uit je eigen bibliotheek"},
		{tabMaatje, "Vraag " + maatje, "je slimme maatje"},
		{tabOntdekken, "Ontdekken", "computerlab, programmeren, dieren, natuur"},
		{tabDenksport, "Denksport", "sudoku, schaakpuzzel, raadsel"},
		{tabSchrijfhoek, "Schrijfhoek", "privé dagboek en verhalen"},
		{tabVandaag, "Vandaag", "weer, wekker, maan, seizoen"},
		{tabSysteem, "Systeem", "naam, kleuren, wifi, volume, hulp"},
	}
}

// wat er per tabblad nog komt (fase-overzicht uit de README)
var komtNog = map[int][]string{
	tabLuisteren:   {"fase 1", "Navidrome: door artiesten scrollen, typen om te zoeken", "Radio met draaiknop langs goedgekeurde zenders", "Luisterboeken uit Audiobookshelf, voortgang gaat mee", "Podcasts die jullie als ouders kiezen", "Slaaptimer en maximaal volume", "Spotify (fase 3): alleen goedgekeurde lijsten, geen autoplay"},
	tabLezen:       {"fase 2", "E-boeken uit Audiobookshelf, ook offline", "Rustige leesweergave met bladwijzers", "Letters groter en kleiner"},
	tabMaatje:      {"fase 4", "Vragen in gewone taal via de Anthropic-API", "Modi: hoe werkt het, huiswerkcoach, overhoren, samen een verhaal, dieren, mythen", "Grenzen: dagbudget, niet na het avondslot, persoonsgegevens worden weggefilterd", "Weekoverzicht voor de ouders"},
	tabOntdekken:   {"fase 5", "Computerlab: Linux leren met missies in een oefenomgeving", "Python, met alleen je eigen kamerlamp als echte uitvoer", "Dier van de dag (Wikipedia) en natuurdagboek"},
	tabDenksport:   {"fase 5", "Sudoku, schaakpuzzels en raadsels", "Geen timer, geen score, een paar per dag"},
	tabSchrijfhoek: {"fase 5", "Privé dagboek en verhalen", "Alleen jij kunt het lezen"},
	tabVandaag:     {"fase 2", "Weer met regen per uur en een praktische tip", "Wekker, planning, maanfase en ISS", "Seizoen en het volgende jaarfeest"},
}

type tickMsg time.Time
type statusMsg struct {
	wifi, accu string
	vol        int
}

type model struct {
	inst       Instellingen
	ouders     Ouders
	oudersFout error

	tab, sel     int
	breed, hoog  int
	nu           time.Time
	velden       []textinput.Model // computer, naam, maatje
	focus        int               // -1 niets, 0..2 tekstveld, 3 gezicht, 4 thema
	pin          textinput.Model
	pinOpen      bool
	ouderModus   bool
	ontgrendeld  time.Time
	melding      string
	meldingTot   time.Time
	wifi, accu   string
	vol          int
	laatsteKleur string
}

func nieuwModel() model {
	o, err := laadOuders()
	m := model{inst: laadInstellingen(), ouders: o, oudersFout: err, nu: time.Now(), focus: -1, vol: -1, wifi: "…", accu: "…"}
	plekken := []struct {
		waarde, hint string
		max          int
	}{{m.inst.Computer, "naam van je computer", 9}, {m.inst.Naam, "zodat ik je kan begroeten", 14}, {m.inst.Maatje, "naam van je maatje", 12}}
	for _, p := range plekken {
		ti := textinput.New()
		ti.SetValue(p.waarde)
		ti.Placeholder = p.hint
		ti.CharLimit = p.max
		ti.Prompt = ""
		m.velden = append(m.velden, ti)
	}
	m.pin = textinput.New()
	m.pin.Prompt = "pincode: "
	m.pin.EchoMode = textinput.EchoPassword
	m.pin.CharLimit = 8
	return m
}

func tick() tea.Cmd {
	return tea.Tick(20*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func leesStatus() tea.Msg { return statusMsg{wifiStatus(), accu(), volume()} }

func (m model) Init() tea.Cmd { return tea.Batch(tick(), leesStatus) }

func (m model) dicht() bool { return isDicht(m.ouders, m.nu) && m.nu.After(m.ontgrendeld) }

func (m *model) zeg(s string) { m.melding = s; m.meldingTot = time.Now().Add(4 * time.Second) }

func (m *model) naarTab(t int) {
	if m.dicht() && t != tabStart && t != tabSysteem && !magTijdensSlot(m.ouders, t) {
		m.zeg("Avondstand: alleen lezen en je luisterboek.")
		t = tabStart
	}
	m.tab = t
	m.focusOp(-1)
	m.pinOpen = false
}

func (m *model) focusOp(f int) {
	m.focus = f
	for i := range m.velden {
		if i == f {
			m.velden[i].Focus()
		} else {
			m.velden[i].Blur()
		}
	}
}

func (m *model) bewaarVelden() {
	m.inst.Computer = strings.TrimSpace(m.velden[0].Value())
	if m.inst.Computer == "" {
		m.inst.Computer = standaardInstellingen().Computer
	}
	m.inst.Naam = strings.TrimSpace(m.velden[1].Value())
	m.inst.Maatje = strings.TrimSpace(m.velden[2].Value())
	if m.inst.Maatje == "" {
		m.inst.Maatje = "Bliep"
	}
	if err := m.inst.bewaar(); err != nil {
		m.zeg("Bewaren lukte niet: " + err.Error())
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.breed, m.hoog = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		m.nu = time.Time(msg)
		if m.dicht() && m.tab != tabSysteem && !magTijdensSlot(m.ouders, m.tab) {
			m.tab = tabStart
		}
		return m, tea.Batch(tick(), leesStatus)
	case statusMsg:
		m.wifi, m.accu, m.vol = msg.wifi, msg.accu, msg.vol
		return m, nil
	case meldingMsg:
		m.zeg(string(msg))
		return m, nil
	case tea.KeyMsg:
		return m.toets(msg)
	}
	return m, nil
}

func (m model) toets(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" || k == "ctrl+z" || k == "ctrl+\\" {
		return m, nil // in de kioskstand kun je de app niet zomaar afsluiten
	}

	// pincode voor de ouders
	if m.pinOpen {
		switch k {
		case "esc":
			m.pinOpen = false
			m.pin.Blur()
		case "enter":
			if m.ouders.pinKlopt(m.pin.Value()) {
				m.ouderModus = true
				m.zeg("Ouderstand aan.")
			} else if m.ouders.PinSHA256 == "" {
				m.zeg("Er is nog geen pincode ingesteld in ouders.yaml.")
			} else {
				m.zeg("Pincode klopt niet.")
			}
			m.pin.SetValue("")
			m.pinOpen = false
			m.pin.Blur()
		default:
			var cmd tea.Cmd
			m.pin, cmd = m.pin.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// typen in een tekstveld bij Systeem
	if m.tab == tabSysteem && m.focus >= 0 && m.focus <= 2 {
		switch k {
		case "esc", "enter":
			m.bewaarVelden()
			m.focusOp(-1)
			return m, nil
		case "tab", "down":
			m.bewaarVelden()
			m.focusOp(m.focus + 1)
			return m, nil
		case "shift+tab", "up":
			m.bewaarVelden()
			if m.focus > 0 {
				m.focusOp(m.focus - 1)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.velden[m.focus], cmd = m.velden[m.focus].Update(msg)
		m.bewaarVelden()
		return m, cmd
	}

	if len(k) == 1 && k[0] >= '0' && k[0] <= '8' {
		m.naarTab(int(k[0] - '0'))
		return m, nil
	}
	if k == "esc" {
		m.naarTab(tabStart)
		return m, nil
	}

	switch m.tab {
	case tabStart:
		items := menu(m.inst.Maatje)
		switch k {
		case "up", "k":
			m.sel = (m.sel - 1 + len(items)) % len(items)
		case "down", "j":
			m.sel = (m.sel + 1) % len(items)
		case "enter":
			m.naarTab(items[m.sel].tab)
		}
	case tabSysteem:
		return m.toetsSysteem(k)
	}
	return m, nil
}

func (m model) toetsSysteem(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "tab", "down":
		m.focusOp((m.focus + 1) % 5)
	case "shift+tab", "up":
		if m.focus <= 0 {
			m.focusOp(4)
		} else {
			m.focusOp(m.focus - 1)
		}
	case "left", "right":
		d := 1
		if k == "left" {
			d = -1
		}
		if m.focus == 3 {
			m.inst.Gezicht = (m.inst.Gezicht + d + len(gezichten)) % len(gezichten)
			m.bewaarVelden()
		}
		if m.focus == 4 {
			i := 0
			for j, n := range themaVolgorde {
				if n == m.inst.Thema {
					i = j
				}
			}
			m.inst.Thema = themaVolgorde[(i+d+len(themaVolgorde))%len(themaVolgorde)]
			m.bewaarVelden()
		}
	case "+", "=":
		m.vol = zetVolume(5, m.ouders.MaxVolume)
		if m.vol >= m.ouders.MaxVolume {
			m.zeg(fmt.Sprintf("Maximaal volume is %d%%.", m.ouders.MaxVolume))
		}
	case "-":
		m.vol = zetVolume(-5, m.ouders.MaxVolume)
	case "h":
		m.zeg("Berichtje versturen…")
		return m, hulpVragen(m.ouders, m.inst.Naam)
	case "o":
		if m.ouderModus {
			m.ouderModus = false
			m.zeg("Ouderstand uit.")
		} else {
			m.pinOpen = true
			m.pin.Focus()
		}
	case "t":
		if m.ouderModus {
			m.ontgrendeld = time.Now().Add(30 * time.Minute)
			m.zeg("Avondslot 30 minuten uit.")
		}
	case "Q":
		if m.ouderModus {
			return m, tea.Quit
		}
	}
	return m, nil
}

/* ---------------- weergave ---------------- */

func (m model) View() string {
	if m.breed == 0 {
		return ""
	}
	p := themas[m.inst.Thema]
	zetTerminalKleuren(p)

	var inhoud string
	switch {
	case m.dicht() && m.tab == tabStart:
		inhoud = m.viewSlot(p)
	case m.tab == tabStart:
		inhoud = m.viewStart(p)
	case m.tab == tabSysteem:
		inhoud = m.viewSysteem(p)
	default:
		inhoud = m.viewKomtNog(p)
	}
	h := m.hoog - 1
	body := lipgloss.NewStyle().Padding(1, 2).Width(m.breed).Height(h).MaxHeight(h).Render(inhoud)
	return body + "\n" + m.viewStatus(p)
}

var vorigeKleur string

// zetTerminalKleuren past de achter- en voorgrondkleur van de terminal aan (OSC 10/11).
func zetTerminalKleuren(p palet) {
	sleutel := string(p.bg) + string(p.fg)
	if sleutel == vorigeKleur {
		return
	}
	vorigeKleur = sleutel
	fmt.Fprintf(os.Stdout, "\x1b]11;%s\x07\x1b]10;%s\x07", p.bg, p.fg)
}

func paneel(p palet, titel, inhoud string, breed int, actief bool) string {
	rand := p.lijn
	if actief {
		rand = p.acc
	}
	kop := lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render(titel)
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(rand).Padding(0, 1).Width(breed).Render(kop + "\n" + inhoud)
}

func (m model) groet() string {
	u := m.nu.Hour()
	g := "Goedenavond"
	switch {
	case u < 6:
		g = "Goedenacht"
	case u < 12:
		g = "Goedemorgen"
	case u < 18:
		g = "Goedemiddag"
	}
	if m.inst.Naam != "" {
		g += ", " + m.inst.Naam
	}
	return g + "!"
}

func (m model) viewStart(p palet) string {
	dim := lipgloss.NewStyle().Foreground(p.dim)
	links := []string{
		lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render(blokletters(m.inst.Computer)),
		"",
		dim.Render("luister- en ontdekcomputer · " + versie),
		lipgloss.NewStyle().Foreground(p.groen).Render(m.groet() + " Kies met ↑ ↓ en Enter, of druk op een cijfer."),
		"",
	}
	for i, it := range menu(m.inst.Maatje) {
		regel := fmt.Sprintf(" %d  %-14s %s ", it.tab, it.naam, dim.Render(it.info))
		if i == m.sel {
			regel = lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render(fmt.Sprintf(" %d  %-14s ", it.tab, it.naam)) + lipgloss.NewStyle().Background(p.sel).Foreground(p.dim).Render(it.info+" ")
		}
		links = append(links, regel)
	}
	rbreed := 38
	if m.breed < 110 {
		rbreed = 30
	}
	slot := fmt.Sprintf("avondstand om %s\nweer open om %s", bedtijd(m.ouders, m.nu), m.ouders.Avondslot.Open)
	if m.oudersFout != nil {
		slot += "\n" + lipgloss.NewStyle().Foreground(p.geel).Render("ouders.yaml niet gevonden, standaardtijden")
	}
	rechts := lipgloss.JoinVertical(lipgloss.Left,
		paneel(p, "vandaag", slot, rbreed, false),
		paneel(p, "nu", dim.Render("luisteren komt in fase 1"), rbreed, false),
		paneel(p, "maatje", lipgloss.NewStyle().Foreground(p.acc).Render(gezichten[m.inst.Gezicht])+"  "+m.inst.Maatje, rbreed, false),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(m.breed-rbreed-10).Render(strings.Join(links, "\n")), rechts)
}

func (m model) viewSlot(p palet) string {
	dim := lipgloss.NewStyle().Foreground(p.dim)
	regels := []string{
		lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render(gezichten[m.inst.Gezicht]),
		"",
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Het is %s. Tijd om tot rust te komen.", bedtijd(m.ouders, m.nu))),
		dim.Render("Ga slapen of lees nog even. De rest is weer open om " + m.ouders.Avondslot.Open + "."),
		"",
	}
	if m.ouders.Avondslot.LezenMag {
		regels = append(regels, "[2] verder lezen")
	}
	if m.ouders.Avondslot.LuisterboekMag {
		regels = append(regels, "[1] luisterboek met slaaptimer")
	}
	regels = append(regels, dim.Render("[8] hulp vragen"), "", dim.Render("Welterusten!"))
	if m.melding != "" && time.Now().Before(m.meldingTot) {
		regels = append(regels, "", lipgloss.NewStyle().Foreground(p.geel).Render(m.melding))
	}
	return lipgloss.Place(m.breed-4, m.hoog-3, lipgloss.Center, lipgloss.Center, lipgloss.JoinVertical(lipgloss.Center, regels...))
}

func (m model) viewKomtNog(p palet) string {
	lijst := komtNog[m.tab]
	titel := tabNamen[m.tab]
	if m.tab == tabMaatje {
		titel = "Vraag " + m.inst.Maatje
	}
	var b strings.Builder
	for _, r := range lijst[1:] {
		b.WriteString("• " + r + "\n")
	}
	kop := lipgloss.NewStyle().Bold(true).Render(strings.ToUpper(titel[:1]) + titel[1:])
	return kop + "\n\n" + paneel(p, "komt in "+lijst[0], strings.TrimRight(b.String(), "\n"), min(m.breed-8, 90), true) +
		"\n\n" + lipgloss.NewStyle().Foreground(p.dim).Render("Dit onderdeel wordt nog gebouwd. Het komt vanzelf binnen met een update.")
}

func (m model) viewSysteem(p palet) string {
	dim := lipgloss.NewStyle().Foreground(p.dim)
	keuze := func(actief bool, s string) string {
		if actief {
			return lipgloss.NewStyle().Background(p.acc).Foreground(p.bg).Bold(true).Render(" " + s + " ")
		}
		return dim.Render(" " + s + " ")
	}
	label := func(i int, s string) string {
		st := lipgloss.NewStyle().Width(16)
		if m.focus == i {
			st = st.Foreground(p.acc).Bold(true)
			s = "▸ " + s
		} else {
			s = "  " + s
		}
		return st.Render(s)
	}
	var gz, th []string
	for i, g := range gezichten {
		gz = append(gz, keuze(i == m.inst.Gezicht, g))
	}
	for _, n := range themaVolgorde {
		th = append(th, keuze(n == m.inst.Thema, themas[n].naam))
	}
	eigen := strings.Join([]string{
		label(0, "naam computer") + m.velden[0].View(),
		label(1, "jouw naam") + m.velden[1].View(),
		label(2, "naam maatje") + m.velden[2].View(),
		label(3, "gezicht") + strings.Join(gz, ""),
		label(4, "kleuren") + strings.Join(th, ""),
		"",
		dim.Render("↑ ↓ kiezen · typen om te wijzigen · ← → bij gezicht en kleuren · esc klaar"),
		dim.Render("wijzigingen worden meteen bewaard"),
	}, "\n")

	vol := "onbekend"
	if m.vol >= 0 {
		balk := m.vol / 5
		vol = lipgloss.NewStyle().Foreground(p.acc).Render(strings.Repeat("█", balk)) + dim.Render(strings.Repeat("░", 20-balk)) + fmt.Sprintf(" %d%% · max %d%%", m.vol, m.ouders.MaxVolume)
	}
	apparaat := strings.Join([]string{
		"wifi    " + m.wifi,
		"accu    " + m.accu,
		"volume  " + vol,
		"versie  " + versie + dim.Render(" · bijwerken gaat vanzelf"),
		"",
		"[+] [-] volume · [h] hulp vragen aan je ouders",
		"[o] ouderinstellingen",
	}, "\n")
	if m.pinOpen {
		apparaat += "\n\n" + m.pin.View()
	}
	if m.ouderModus {
		a := m.ouders.Avondslot
		apparaat += "\n\n" + lipgloss.NewStyle().Foreground(p.geel).Bold(true).Render("ouderstand") + "\n" +
			fmt.Sprintf("avondslot  school %s · weekend %s · open %s\n", a.Schooldagen, a.Weekend, a.Open) +
			fmt.Sprintf("na het slot  lezen %s · luisterboek %s\n", jaNee(a.LezenMag), jaNee(a.LuisterboekMag)) +
			dim.Render("wijzigen in /etc/unscroll/ouders.yaml") + "\n" +
			"[t] avondslot 30 min uit · [Q] app afsluiten · [o] ouderstand uit"
	}
	breed := (m.breed - 10) / 2
	return lipgloss.JoinHorizontal(lipgloss.Top,
		paneel(p, "maak hem van jou", eigen, breed+6, true),
		" ",
		paneel(p, "apparaat", apparaat, breed-6, false),
	)
}

func jaNee(b bool) string {
	if b {
		return "ja"
	}
	return "nee"
}

func (m model) viewStatus(p palet) string {
	sessie := strings.ToLower(strings.TrimSpace(m.inst.Computer))
	if sessie == "" {
		sessie = "unscroll"
	}
	delen := []string{lipgloss.NewStyle().Background(p.groen).Foreground(p.bg).Bold(true).Render(" " + sessie + " ")}
	for i, n := range tabNamen {
		if i == tabMaatje {
			n = strings.ToLower(m.inst.Maatje)
			if len(n) > 8 {
				n = n[:8]
			}
		}
		t := fmt.Sprintf(" %d:%s ", i, n)
		if i == m.tab {
			delen = append(delen, lipgloss.NewStyle().Background(p.acc).Foreground(p.bg).Bold(true).Render(t))
		} else {
			delen = append(delen, lipgloss.NewStyle().Foreground(p.dim).Render(t))
		}
	}
	links := strings.Join(delen, "")
	rechts := m.nu.Format("15:04")
	if m.dicht() {
		rechts = lipgloss.NewStyle().Foreground(p.geel).Render("avondstand") + " · " + rechts
	}
	if m.melding != "" && time.Now().Before(m.meldingTot) {
		rechts = lipgloss.NewStyle().Foreground(p.geel).Render(m.melding) + "  " + rechts
	}
	ruimte := m.breed - lipgloss.Width(links) - lipgloss.Width(rechts) - 1
	if ruimte < 1 {
		ruimte = 1
	}
	return lipgloss.NewStyle().Background(p.bg2).Width(m.breed).Render(links + strings.Repeat(" ", ruimte) + rechts)
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(versie)
			return
		case "--pin-hash":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "gebruik: unscroll --pin-hash 1234")
				os.Exit(2)
			}
			fmt.Println(pinHash(os.Args[2]))
			return
		}
	}
	p := tea.NewProgram(nieuwModel(), tea.WithAltScreen())
	_, err := p.Run()
	fmt.Fprint(os.Stdout, "\x1b]110\x07\x1b]111\x07") // terminalkleuren terug
	if err != nil {
		fmt.Fprintln(os.Stderr, "unscroll:", err)
		os.Exit(1)
	}
}
