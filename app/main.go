// Unscroll: een rustige luister- en ontdekcomputer voor kinderen.
// Draait als volledig-scherm terminal-app op oude laptops, netbooks en Raspberry Pi's.
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

// sleutels in de vertalingen per tabblad
var tabSleutels = []string{"tab.start", "tab.listen", "tab.read", "", "tab.discover", "tab.puzzles", "tab.write", "tab.today", "tab.system"}
var menuSleutels = []string{"", "menu.listen", "menu.read", "menu.buddy", "menu.discover", "menu.puzzles", "menu.write", "menu.today", "menu.system"}
var straksSleutels = map[int]string{}

type tickMsg time.Time
type statusMsg struct {
	wifi, accu string
	vol        int
}

const (
	veldComputer = iota
	veldNaam
	veldMaatje
	veldGezicht
	veldThema
	veldTaal
	aantalVelden
)

type model struct {
	inst       Instellingen
	ouders     Ouders
	oudersFout error

	tab, sel     int
	breed, hoog  int
	nu           time.Time
	velden       []textinput.Model // computer, naam, maatje
	focus        int               // -1 niets, anders een veld* constante
	pin          textinput.Model
	pinOpen      bool
	ouderModus   bool
	ontgrendeld  time.Time
	laatsteToets time.Time
	melding      string
	bewaarFout   string // blijft staan zolang bewaren niet lukt
	meldingTot   time.Time
	wifi, accu   string
	vol          int
	l            *luister
	mj           *maatjeStaat
	ls           *leesStaat
	od           *ontdekStaat
	vd           *vandaagStaat
	sh           *schrijfStaat
	oi           *ouderInstStaat
	ouderPin     string // alleen zolang de ouderstand aan is, voor het wijzigen van instellingen
	dk           *denkStaat
}

func nieuwModel() model {
	o, err := laadOuders()
	m := model{inst: laadInstellingen(), ouders: o, oudersFout: err, nu: time.Now(), focus: -1, vol: -1, wifi: "…", accu: "…", ontgrendeld: laadPauze()}
	for _, w := range []struct {
		waarde string
		max    int
	}{{m.inst.Computer, 9}, {m.inst.Naam, 14}, {m.inst.Maatje, 12}} {
		ti := textinput.New()
		ti.SetValue(w.waarde)
		ti.CharLimit = w.max
		ti.Prompt = ""
		m.velden = append(m.velden, ti)
	}
	geheim := laadGeheimen()
	m.l = nieuwLuister(geheim)
	m.mj = nieuwMaatjeStaat(geheim, o)
	m.ls = &leesStaat{}
	m.od = nieuweOntdek()
	m.vd = &vandaagStaat{}
	m.sh = nieuweSchrijfhoek()
	m.oi = nieuweOuderInst()
	m.dk = nieuweDenksport()
	m.pin = textinput.New()
	m.pin.EchoMode = textinput.EchoPassword
	m.pin.CharLimit = 8
	m.zetTaalteksten()
	return m
}

func (m *model) t(sleutel string, args ...any) string { return tr(m.inst.Taal, sleutel, args...) }

func (m *model) zetTaalteksten() {
	m.velden[veldComputer].Placeholder = m.t("sys.ph.computer")
	m.velden[veldNaam].Placeholder = m.t("sys.ph.name")
	m.velden[veldMaatje].Placeholder = m.t("sys.ph.buddy")
	m.pin.Prompt = m.t("sys.pin")
}

func tick() tea.Cmd {
	return tea.Tick(20*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func leesStatus(taal string) tea.Cmd {
	return func() tea.Msg { return statusMsg{wifiStatus(taal), accu(taal), volume()} }
}

func (m model) Init() tea.Cmd { return tea.Batch(tick(), leesStatus(m.inst.Taal)) }

func (m model) dicht() bool { return isDicht(m.ouders, m.nu) && m.nu.After(m.ontgrendeld) }

func (m *model) zeg(s string) { m.melding = s; m.meldingTot = time.Now().Add(4 * time.Second) }

func (m *model) naarTab(t int) tea.Cmd {
	if m.dicht() && t != tabStart && t != tabSysteem && !magTijdensSlot(m.ouders, t) {
		m.zeg(m.t("lock.only"))
		t = tabStart
	}
	m.tab = t
	m.focusOp(-1)
	m.pinOpen = false
	if t == tabLuisteren {
		return m.luisterOpen()
	}
	if t == tabLezen {
		return m.lezenOpen()
	}
	if t == tabVandaag {
		return m.vandaagOpen()
	}
	if t == tabSchrijfhoek && !m.sh.typen {
		m.sh.taken = laadTaken()
	}
	return nil
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

func (m *model) bewaar() {
	m.inst.Computer = strings.TrimSpace(m.velden[veldComputer].Value())
	if m.inst.Computer == "" {
		m.inst.Computer = standaardInstellingen().Computer
	}
	m.inst.Naam = strings.TrimSpace(m.velden[veldNaam].Value())
	m.inst.Maatje = strings.TrimSpace(m.velden[veldMaatje].Value())
	if m.inst.Maatje == "" {
		m.inst.Maatje = "Bliep"
	}
	if err := m.inst.bewaar(); err != nil {
		m.bewaarFout = m.t("msg.saveerr", err.Error())
		m.zeg(m.bewaarFout)
	} else {
		m.bewaarFout = ""
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.breed, m.hoog = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		m.nu = time.Time(msg)
		m.maatjeTik(m.nu)
		if m.ouderModus && m.nu.Sub(m.laatsteToets) > 10*time.Minute {
			m.ouderModus, m.ouderPin, m.oi.open = false, "", false
			m.zeg(m.t("msg.parentauto"))
		}
		if m.dicht() && m.tab != tabSysteem && !magTijdensSlot(m.ouders, m.tab) {
			m.tab = tabStart
		}
		var luister tea.Cmd
		if m.dicht() && m.tab == tabLuisteren && m.l.bron != bronBoeken {
			luister = m.luisterOpen()
		}
		var weer tea.Cmd
		if m.tab == tabVandaag {
			weer = m.vandaagOpen() // elk half uur vers weer zolang het scherm open is
		}
		slot := m.luisterSlot()
		return m, tea.Batch(tick(), leesStatus(m.inst.Taal), slot, luister, weer)
	case statusMsg:
		m.wifi, m.accu, m.vol = msg.wifi, msg.accu, msg.vol
		return m, nil
	case meldingMsg:
		m.zeg(string(msg))
		return m, nil
	case hedyUitMsg, hedyKlaarMsg:
		cmd := m.hedyBericht(msg)
		return m, cmd
	case issMsg:
		m.hemelBericht(msg)
		return m, nil
	case instMsg:
		cmd := m.instBericht(msg)
		return m, cmd
	case motorZetMsg:
		cmd := m.schaakBericht(msg)
		return m, cmd
	case sudokuMsg:
		m.sudokuBericht(msg)
		return m, nil
	case woordMsg:
		m.woordBericht(msg)
		return m, nil
	case weerMsg:
		m.vandaagBericht(msg)
		return m, nil
	case dierMsg:
		m.ontdekBericht(msg)
		return m, nil
	case eboekenMsg, boekMsg:
		cmd := m.lezenBericht(msg)
		return m, cmd
	case maatjeMsg:
		cmd := m.maatjeBericht(msg)
		return m, cmd
	case lijstMsg, startMsg, spelerMsg:
		cmd := m.luisterBericht(msg)
		return m, cmd
	case tea.KeyMsg:
		return m.toets(msg)
	}
	return m, nil
}

func (m model) toets(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	m.laatsteToets = time.Now()
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
			switch {
			case m.ouders.pinKlopt(m.pin.Value()):
				m.ouderModus, m.ouderPin = true, m.pin.Value()
				m.zeg(m.t("msg.parenton"))
			case m.ouders.PinSHA256 == "":
				m.zeg(m.t("msg.nopin"))
			default:
				m.zeg(m.t("msg.badpin"))
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
	if m.tab == tabSysteem && m.focus >= 0 && m.focus < len(m.velden) {
		switch k {
		case "esc", "enter":
			m.bewaar()
			m.focusOp(-1)
			return m, nil
		case "tab", "down":
			m.bewaar()
			m.focusOp(m.focus + 1)
			return m, nil
		case "shift+tab", "up":
			m.bewaar()
			if m.focus > 0 {
				m.focusOp(m.focus - 1)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.velden[m.focus], cmd = m.velden[m.focus].Update(msg)
		m.bewaar()
		return m, cmd
	}

	if m.tab == tabSysteem && m.oi.open && m.ouderModus {
		if cmd, ok := m.toetsOuderInst(msg); ok {
			return m, cmd
		}
	}
	if m.tab == tabLuisteren {
		if cmd, ok := m.toetsLuisteren(msg); ok {
			return m, cmd
		}
	}
	if m.tab == tabVandaag && k == "r" && !m.vd.laden {
		m.vd.laden = true
		return m, haalWeer(m.ouders)
	}
	if m.tab == tabDenksport {
		if cmd, ok := m.toetsDenk(msg); ok {
			return m, cmd
		}
	}
	if m.tab == tabSchrijfhoek {
		if cmd, ok := m.toetsSchrijf(msg); ok {
			return m, cmd
		}
	}
	if m.tab == tabOntdekken {
		if cmd, ok := m.toetsOntdek(msg); ok {
			return m, cmd
		}
	}
	if m.tab == tabLezen {
		if cmd, ok := m.toetsLezen(msg); ok {
			return m, cmd
		}
	}
	if m.tab == tabMaatje {
		if cmd, ok := m.toetsMaatje(msg); ok {
			return m, cmd
		}
	}
	if len(k) == 1 && k[0] >= '0' && k[0] <= '8' {
		cmd := m.naarTab(int(k[0] - '0'))
		return m, cmd
	}
	if k == "esc" {
		cmd := m.naarTab(tabStart)
		return m, cmd
	}

	switch m.tab {
	case tabStart:
		n := len(menuSleutels) - 1
		switch k {
		case "up", "k":
			m.sel = (m.sel - 1 + n) % n
		case "down", "j":
			m.sel = (m.sel + 1) % n
		case "enter":
			cmd := m.naarTab(m.sel + 1)
			return m, cmd
		}
	case tabSysteem:
		return m.toetsSysteem(k)
	}
	return m, nil
}

func draaiKeuze(lijst []string, huidig string, d int) string {
	i := 0
	for j, n := range lijst {
		if n == huidig {
			i = j
		}
	}
	return lijst[(i+d+len(lijst))%len(lijst)]
}

func (m model) toetsSysteem(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "tab", "down":
		m.focusOp((m.focus + 1) % aantalVelden)
	case "shift+tab", "up":
		if m.focus <= 0 {
			m.focusOp(aantalVelden - 1)
		} else {
			m.focusOp(m.focus - 1)
		}
	case "left", "right":
		d := 1
		if k == "left" {
			d = -1
		}
		switch m.focus {
		case veldGezicht:
			m.inst.Gezicht = (m.inst.Gezicht + d + len(gezichten)) % len(gezichten)
		case veldThema:
			m.inst.Thema = draaiKeuze(themaVolgorde, m.inst.Thema, d)
		case veldTaal:
			m.inst.Taal = draaiKeuze(talen, m.inst.Taal, d)
			m.zetTaalteksten()
		}
		m.bewaar()
	case "+", "=":
		m.vol = zetVolume(5, m.ouders.MaxVolume)
		if m.vol >= m.ouders.MaxVolume {
			m.zeg(m.t("msg.maxvol", m.ouders.MaxVolume))
		}
	case "-":
		m.vol = zetVolume(-5, m.ouders.MaxVolume)
	case "h":
		m.zeg(m.t("msg.sending"))
		return m, hulpVragen(m.ouders, m.inst.Naam, m.inst.Taal)
	case "o":
		if m.ouderModus {
			m.ouderModus, m.ouderPin, m.oi.open = false, "", false
			m.zeg(m.t("msg.parentoff"))
		} else {
			m.pinOpen = true
			m.pin.Focus()
		}
	case "t", "u", "m", "x", "a":
		if !m.ouderModus {
			break
		}
		nu := time.Now()
		switch k {
		case "t":
			m.ontgrendeld = nu.Add(30 * time.Minute)
		case "u":
			m.ontgrendeld = nu.Add(2 * time.Hour)
		case "m":
			m.ontgrendeld = volgendeOpening(m.ouders, nu)
		case "x":
			m.ontgrendeld = pauzeUit
		case "a":
			m.ontgrendeld = time.Time{}
		}
		bewaarPauze(m.ontgrendeld)
		m.nu = nu
		m.zeg(m.pauzeTekst())
	case "i":
		if m.ouderModus {
			m.oi.open, m.oi.melding = true, ""
		}
	case "Q":
		if m.ouderModus {
			return m, tea.Sequence(m.stopHuidig(true), tea.Quit)
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
	case m.tab == tabSysteem && m.oi.open && m.ouderModus:
		inhoud = m.viewOuderInst(p)
	case m.tab == tabSysteem:
		inhoud = m.viewSysteem(p)
	case m.tab == tabLuisteren:
		inhoud = m.viewLuisteren(p)
	case m.tab == tabMaatje:
		inhoud = m.viewMaatje(p)
	case m.tab == tabLezen:
		inhoud = m.viewLezen(p)
	case m.tab == tabOntdekken:
		inhoud = m.viewOntdek(p)
	case m.tab == tabVandaag:
		inhoud = m.viewVandaag(p)
	case m.tab == tabSchrijfhoek:
		inhoud = m.viewSchrijf(p)
	case m.tab == tabDenksport:
		inhoud = m.viewDenk(p)
	default:
		inhoud = m.viewStraks(p)
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
	g := m.t("greet.evening")
	switch {
	case u < 6:
		g = m.t("greet.night")
	case u < 12:
		g = m.t("greet.morning")
	case u < 18:
		g = m.t("greet.afternoon")
	}
	if m.inst.Naam != "" {
		g += ", " + m.inst.Naam
	}
	return g + "!"
}

func (m model) menuNaam(tab int) string {
	if tab == tabMaatje {
		return m.t(menuSleutels[tab], m.inst.Maatje)
	}
	return m.t(menuSleutels[tab])
}

func (m model) viewStart(p palet) string {
	dim := lipgloss.NewStyle().Foreground(p.dim)
	links := []string{
		lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render(blokletters(m.inst.Computer)),
		"",
		dim.Render(m.t("start.subtitle", versie)),
		lipgloss.NewStyle().Foreground(p.groen).Render(m.groet() + " " + m.t("start.hint")),
		"",
	}
	for tab := 1; tab < len(menuSleutels); tab++ {
		naam, info := m.menuNaam(tab), m.t(menuSleutels[tab]+".d")
		if tab-1 == m.sel {
			links = append(links, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render(fmt.Sprintf(" %d  %-16s ", tab, naam))+lipgloss.NewStyle().Background(p.sel).Foreground(p.dim).Render(info+" "))
		} else {
			links = append(links, fmt.Sprintf(" %d  %-16s %s ", tab, naam, dim.Render(info)))
		}
	}
	rbreed := 38
	if m.breed < 110 {
		rbreed = 30
	}
	slot := m.t("start.lock", bedtijd(m.ouders, m.nu), m.ouders.Avondslot.Open)
	if m.oudersFout != nil {
		slot += "\n" + lipgloss.NewStyle().Foreground(p.geel).Width(rbreed-4).Render(m.t("start.noparent"))
		if !os.IsNotExist(m.oudersFout) {
			slot += "\n" + lipgloss.NewStyle().Foreground(p.rood).Width(rbreed-4).Render(m.oudersFout.Error())
		}
	}
	rechts := lipgloss.JoinVertical(lipgloss.Left,
		paneel(p, m.t("start.today"), slot, rbreed, false),
		paneel(p, m.t("start.now"), dim.Render(m.nuKort()), rbreed, false),
		paneel(p, m.t("start.buddy"), lipgloss.NewStyle().Foreground(p.acc).Render(gezichten[m.inst.Gezicht])+"  "+m.inst.Maatje, rbreed, false),
		paneel(p, m.t("start.todo"), lipgloss.NewStyle().Width(rbreed-4).Render(m.takenKort(3)), rbreed, false),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(m.breed-rbreed-10).Render(strings.Join(links, "\n")), rechts)
}

func (m model) viewSlot(p palet) string {
	dim := lipgloss.NewStyle().Foreground(p.dim)
	regels := []string{
		lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render(gezichten[m.inst.Gezicht]),
		"",
		lipgloss.NewStyle().Bold(true).Render(m.t("lock.title", bedtijd(m.ouders, m.nu))),
		dim.Render(m.t("lock.sub", m.ouders.Avondslot.Open)),
		"",
	}
	if m.ouders.Avondslot.LezenMag {
		regels = append(regels, m.t("lock.read"))
	}
	if m.ouders.Avondslot.LuisterboekMag {
		regels = append(regels, m.t("lock.listen"))
	}
	regels = append(regels, dim.Render(m.t("lock.help")), "", dim.Render(m.t("lock.night")))
	if m.melding != "" && time.Now().Before(m.meldingTot) {
		regels = append(regels, "", lipgloss.NewStyle().Foreground(p.geel).Render(m.melding))
	}
	return lipgloss.Place(m.breed-4, m.hoog-3, lipgloss.Center, lipgloss.Center, lipgloss.JoinVertical(lipgloss.Center, regels...))
}

func (m model) viewStraks(p palet) string {
	regels := strings.Split(m.t(straksSleutels[m.tab]), "\n")
	var b strings.Builder
	for _, r := range regels[1:] {
		b.WriteString("• " + r + "\n")
	}
	kop := lipgloss.NewStyle().Bold(true).Render(m.menuNaam(m.tab))
	return kop + "\n\n" + paneel(p, m.t("soon.in", regels[0]), strings.TrimRight(b.String(), "\n"), min(m.breed-8, 90), true) +
		"\n\n" + lipgloss.NewStyle().Foreground(p.dim).Render(m.t("soon.building"))
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
	var gz, th, tl []string
	for i, g := range gezichten {
		gz = append(gz, keuze(i == m.inst.Gezicht, g))
	}
	for _, n := range themaVolgorde {
		th = append(th, keuze(n == m.inst.Thema, m.t("theme."+n)))
	}
	for _, c := range talen {
		tl = append(tl, keuze(c == m.inst.Taal, taalNamen[c]))
	}
	eigen := strings.Join([]string{
		label(veldComputer, m.t("sys.computer")) + m.velden[veldComputer].View(),
		label(veldNaam, m.t("sys.name")) + m.velden[veldNaam].View(),
		label(veldMaatje, m.t("sys.buddy")) + m.velden[veldMaatje].View(),
		label(veldGezicht, m.t("sys.face")) + strings.Join(gz, ""),
		label(veldThema, m.t("sys.colors")) + strings.Join(th, ""),
		label(veldTaal, m.t("sys.lang")) + strings.Join(tl, ""),
		"",
		dim.Render(m.t("sys.keys")),
		dim.Render(m.t("sys.saved")),
	}, "\n")
	if m.bewaarFout != "" {
		eigen += "\n\n" + lipgloss.NewStyle().Foreground(p.rood).Width((m.breed-10)/2).Render(m.bewaarFout+" "+m.t("msg.saveask"))
	}

	vol := m.t("sys.unknown")
	if m.vol >= 0 {
		balk := m.vol / 5
		vol = lipgloss.NewStyle().Foreground(p.acc).Render(strings.Repeat("█", balk)) + dim.Render(strings.Repeat("░", 20-balk)) + fmt.Sprintf(" %d%%", m.vol) + m.t("sys.maxvol", m.ouders.MaxVolume)
	}
	kol := func(s string) string { return lipgloss.NewStyle().Width(9).Render(s) }
	apparaat := strings.Join([]string{
		kol(m.t("sys.wifi")) + m.wifi,
		kol(m.t("sys.battery")) + m.accu,
		kol(m.t("sys.volume")) + vol,
		kol(m.t("sys.version")) + versie + dim.Render(m.t("sys.autoupdate")),
		"",
		m.t("sys.actions"),
		m.t("sys.parent"),
	}, "\n")
	if m.pinOpen {
		apparaat += "\n\n" + m.pin.View()
	}
	if m.ouderModus {
		a := m.ouders.Avondslot
		jn := func(b bool) string {
			if b {
				return m.t("yes")
			}
			return m.t("no")
		}
		apparaat += "\n\n" + lipgloss.NewStyle().Foreground(p.geel).Bold(true).Render(m.t("sys.parentmode")) + "\n" +
			m.t("sys.p.lock", a.Schooldagen, a.Weekend, a.Open) + "\n" +
			m.t("sys.p.after", jn(a.LezenMag), jn(a.LuisterboekMag)) + "\n" +
			lipgloss.NewStyle().Foreground(p.acc).Render(m.pauzeTekst()) + "\n" +
			dim.Render(m.t("sys.p.edit")) + "\n" + m.t("sys.p.keys")
	}
	breed := (m.breed - 10) / 2
	return lipgloss.JoinHorizontal(lipgloss.Top,
		paneel(p, m.t("sys.own"), eigen, breed+6, true),
		" ",
		paneel(p, m.t("sys.device"), apparaat, breed-6, false),
	)
}

func (m model) viewStatus(p palet) string {
	sessie := strings.ToLower(strings.TrimSpace(m.inst.Computer))
	if sessie == "" {
		sessie = "unscroll"
	}
	delen := []string{lipgloss.NewStyle().Background(p.groen).Foreground(p.bg).Bold(true).Render(" " + sessie + " ")}
	for i, s := range tabSleutels {
		n := strings.ToLower(m.inst.Maatje)
		if s != "" {
			n = m.t(s)
		}
		if len([]rune(n)) > 8 {
			n = string([]rune(n)[:8])
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
		rechts = lipgloss.NewStyle().Foreground(p.geel).Render(m.t("lock.status")) + " · " + rechts
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
		case "--spotify-login":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: unscroll --spotify-login CLIENT_ID")
				os.Exit(2)
			}
			if err := spotifyInloggen(os.Args[2], os.Stdin, os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, "Spotify login failed:", err)
				os.Exit(1)
			}
			return
		case "--ouder-zet":
			// als root, via sudo: wijzigt ouders.yaml na controle van de pincode (zie ouderinstellingen.go)
			if err := ouderZet(os.Stdin); err != nil {
				fmt.Fprintln(os.Stderr, "unscroll:", err)
				os.Exit(1)
			}
			return
		case "--pin-hash":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: unscroll --pin-hash 1234")
				os.Exit(2)
			}
			fmt.Println(pinHash(os.Args[2]))
			return
		}
	}
	// niet de terminal naar de achtergrondkleur vragen: niet elke console antwoordt
	lipgloss.SetHasDarkBackground(true)
	begin := nieuwModel()
	p := tea.NewProgram(begin, tea.WithAltScreen())
	_, err := p.Run()
	begin.l.sp.Afsluiten()                            // mpv stoppen
	begin.l.lspot.Stop()                              // librespot stoppen
	fmt.Fprint(os.Stdout, "\x1b]110\x07\x1b]111\x07") // terminalkleuren terug
	if err != nil {
		fmt.Fprintln(os.Stderr, "unscroll:", err)
		os.Exit(1)
	}
}

// pauzeTekst beschrijft de huidige stand van het avondslot.
func (m model) pauzeTekst() string {
	nu := time.Now()
	switch {
	case !m.ontgrendeld.Before(pauzeUit):
		return m.t("sys.p.state.off")
	case m.ontgrendeld.After(nu):
		klok := m.ontgrendeld.Format("15:04")
		if m.ontgrendeld.YearDay() != nu.YearDay() || m.ontgrendeld.Year() != nu.Year() {
			klok = m.t("time.tomorrow", klok)
		}
		return m.t("sys.p.state.paused", klok)
	}
	return m.t("sys.p.state.on", bedtijd(m.ouders, nu))
}
