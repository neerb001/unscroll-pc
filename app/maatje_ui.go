package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* ---------------- gebruik per dag ---------------- */

// maatjeGebruik telt per dag hoeveel het maatje gebruikt is, voor de grenzen van de ouders.
type maatjeGebruik struct {
	Datum        string    `json:"datum"`
	Vragen       int       `json:"vragen"`
	Huiswerk     int       `json:"huiswerk"`
	Verhalen     int       `json:"verhalen"`
	Seconden     int       `json:"seconden"`
	Blok         int       `json:"blok"` // seconden achter elkaar, voor de pauze
	LaatstActief time.Time `json:"laatst_actief"`
	PauzeTot     time.Time `json:"pauze_tot"`
}

func maatjeMap() string { return filepath.Dir(instellingenPad()) }

func laadGebruik(nu time.Time) maatjeGebruik {
	var g maatjeGebruik
	if data, err := os.ReadFile(filepath.Join(maatjeMap(), "maatje.json")); err == nil {
		_ = json.Unmarshal(data, &g)
	}
	if d := nu.Format("2006-01-02"); g.Datum != d {
		g = maatjeGebruik{Datum: d, PauzeTot: g.PauzeTot}
	}
	return g
}

func (g maatjeGebruik) bewaar() {
	_ = os.MkdirAll(maatjeMap(), 0o755)
	data, _ := json.Marshal(g)
	_ = os.WriteFile(filepath.Join(maatjeMap(), "maatje.json"), data, 0o644)
}

// logRegel is wat ouders in het weekoverzicht zien. Het kind weet dat.
type logRegel struct {
	Tijd    time.Time `json:"tijd"`
	Modus   int       `json:"modus"`
	Vraag   string    `json:"vraag"`
	Oordeel string    `json:"oordeel"`
}

func logPad() string { return filepath.Join(maatjeMap(), "maatje-log.jsonl") }

func schrijfLog(r logRegel) {
	_ = os.MkdirAll(maatjeMap(), 0o755)
	f, err := os.OpenFile(logPad(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	data, _ := json.Marshal(r)
	_, _ = f.Write(append(data, '\n'))
}

func leesLog(vanaf time.Time) []logRegel {
	f, err := os.Open(logPad())
	if err != nil {
		return nil
	}
	defer f.Close()
	var uit []logRegel
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r logRegel
		if json.Unmarshal(sc.Bytes(), &r) == nil && r.Tijd.After(vanaf) {
			uit = append(uit, r)
		}
	}
	return uit
}

/* ---------------- staat ---------------- */

type regel struct {
	wie   int // 0 kind, 1 maatje, 2 melding
	tekst string
}

type sessie struct {
	id, modus, beurten int
	gesprek            []anthropic.BetaMessageParam
	regels             []regel
	wacht, geteld      bool
	verhaal            string
	eerder             string // het gesprek of verhaal tot nu toe, bij het hervatten
	alleenLezen        bool   // ouders lezen een huiswerksessie terug
	scroll             int    // regels omhoog gescrold
}

type maatjeStaat struct {
	ai      *MaatjeAI
	gebruik maatjeGebruik
	sessie  *sessie
	invoer  textinput.Model
	keuze   int
	week    bool
	volgID  int

	kiesVerhaal bool // kiest tussen een nieuwe en een eerdere sessie
	kiesModus   int  // voor welke modus; -1: ouders kiezen huiswerk om terug te lezen
	verhalen    []verhaalBestand
	verhaalSel  int
}

func nieuwMaatjeStaat(g Geheimen, o Ouders) *maatjeStaat {
	s := &maatjeStaat{ai: nieuwMaatjeAI(g, o), gebruik: laadGebruik(time.Now())}
	s.invoer = textinput.New()
	s.invoer.CharLimit = 400
	s.invoer.Prompt = "› "
	return s
}

type maatjeMsg struct {
	id                       int
	vraag, antwoord          string
	gesprek                  []anthropic.BetaMessageParam
	oordeelVraag, oordeelAnt string
	err                      error
}

/* ---------------- grenzen ---------------- */

// maatjeDicht geeft een uitleg als het maatje nu niet kan, anders "".
func (m *model) maatjeDicht() string {
	mj, g := m.mj, m.ouders.Maatje
	switch {
	case !g.Aan:
		return m.t("buddy.off")
	case mj.ai == nil:
		return m.t("buddy.notset")
	case mj.gebruik.PauzeTot.After(time.Now()):
		return m.t("buddy.break", mj.gebruik.PauzeTot.Format("15:04"))
	case g.MaxMinutenPerDag > 0 && mj.gebruik.Seconden >= g.MaxMinutenPerDag*60:
		return m.t("buddy.daydone")
	}
	return ""
}

// overVandaag: hoeveel er vandaag nog van een modus over is.
func (m *model) overVandaag(modus int) int {
	g, u := m.ouders.Maatje, m.mj.gebruik
	// 0 betekent: geen grens
	over := func(grens, gebruikt int) int {
		if grens <= 0 {
			return 999
		}
		return grens - gebruikt
	}
	switch modus {
	case modusVraag:
		return over(g.VrijeVragenPerDag, u.Vragen)
	case modusVerhaal:
		return over(g.VerhaalSessiesPerDag, u.Verhalen)
	}
	return over(g.HuiswerkSessiesPerDag, u.Huiswerk)
}

func (m *model) maxBeurten(modus int) int {
	switch modus {
	case modusVraag:
		return 0 // telt per vraag
	case modusVerhaal:
		return m.ouders.Maatje.BeurtenPerVerhaal
	}
	return m.ouders.Maatje.BeurtenPerHuiswerk
}

// maatjeTik houdt de minuten bij en zet een pauze na een lang blok.
func (m *model) maatjeTik(nu time.Time) {
	mj := m.mj
	if d := nu.Format("2006-01-02"); mj.gebruik.Datum != d {
		mj.gebruik = laadGebruik(nu)
	}
	if mj.sessie == nil || m.tab != tabMaatje || nu.Sub(m.laatsteToets) > 2*time.Minute {
		return
	}
	if nu.Sub(mj.gebruik.LaatstActief) > 10*time.Minute {
		mj.gebruik.Blok = 0 // lang genoeg weggeweest: dat telt als pauze
	}
	mj.gebruik.Seconden += 20
	mj.gebruik.Blok += 20
	mj.gebruik.LaatstActief = nu
	if p := m.ouders.Maatje.PauzeNaMinuten; p > 0 && mj.gebruik.Blok >= p*60 {
		mj.gebruik.Blok = 0
		mj.gebruik.PauzeTot = nu.Add(10 * time.Minute)
		mj.sessie.regels = append(mj.sessie.regels, regel{2, m.t("buddy.pausenow", mj.gebruik.PauzeTot.Format("15:04"))})
	}
	mj.gebruik.bewaar()
}

/* ---------------- toetsen ---------------- */

func (m *model) toetsMaatje(msg tea.KeyMsg) (tea.Cmd, bool) {
	mj := m.mj
	k := msg.String()
	s := mj.sessie
	if s == nil && mj.kiesVerhaal {
		eerste := 1 // plek 0 is "nieuw", behalve bij teruglezen door ouders
		if mj.kiesModus < 0 {
			eerste = 0
		}
		switch k {
		case "up", "k":
			mj.verhaalSel = max(mj.verhaalSel-1, 0)
		case "down", "j":
			mj.verhaalSel = min(mj.verhaalSel+1, len(mj.verhalen)-1+eerste)
		case "enter":
			mj.kiesVerhaal = false
			if eerste == 1 && mj.verhaalSel == 0 {
				return m.startSessie(mj.kiesModus), true
			}
			return m.verderMet(mj.verhalen[mj.verhaalSel-eerste], mj.kiesModus < 0), true
		case "esc":
			mj.kiesVerhaal = false
		default:
			return nil, false
		}
		return nil, true
	}
	if s == nil {
		switch k {
		case "up", "k":
			mj.keuze = (mj.keuze - 1 + aantalModi) % aantalModi
		case "down", "j":
			mj.keuze = (mj.keuze + 1) % aantalModi
		case "enter":
			if mj.keuze != modusVraag {
				if v := eerdereSessies(mj.keuze, 6); len(v) > 0 {
					mj.verhalen, mj.verhaalSel, mj.kiesVerhaal, mj.kiesModus = v, 0, true, mj.keuze
					return nil, true
				}
			}
			return m.startSessie(mj.keuze), true
		case "h":
			if m.ouderModus {
				if v := eerdereSessies(-1, 20); len(v) > 0 {
					mj.verhalen, mj.verhaalSel, mj.kiesVerhaal, mj.kiesModus = v, 0, true, -1
				} else {
					m.zeg(m.t("buddy.hwnone"))
				}
				return nil, true
			}
			return nil, false
		case "w":
			if m.ouderModus {
				mj.week = !mj.week
				return nil, true
			}
			return nil, false
		default:
			return nil, false
		}
		return nil, true
	}

	switch k {
	case "esc":
		m.stopSessie()
		return nil, true
	case "up", "pgup":
		stap := 1
		if k == "pgup" {
			stap = 10
		}
		s.scroll += stap
		return nil, true
	case "down", "pgdown":
		stap := 1
		if k == "pgdown" {
			stap = 10
		}
		s.scroll = max(s.scroll-stap, 0)
		return nil, true
	}
	if s.alleenLezen {
		return nil, true
	}
	switch k {
	case "enter":
		tekst := strings.TrimSpace(mj.invoer.Value())
		if tekst == "" || s.wacht {
			return nil, true
		}
		mj.invoer.SetValue("")
		return m.stuur(tekst, true), true
	case "tab":
		if mj.invoer.Value() == "" && !s.wacht && s.beurten > 0 {
			return m.stuur(m.t("buddy.check"), false), true
		}
		return nil, true
	}
	var cmd tea.Cmd
	mj.invoer, cmd = mj.invoer.Update(msg)
	return cmd, true
}

func (m *model) startSessie(modus int) tea.Cmd {
	mj := m.mj
	if reden := m.maatjeDicht(); reden != "" {
		m.zeg(reden)
		return nil
	}
	if m.overVandaag(modus) <= 0 {
		m.zeg(m.t("buddy.none"))
		return nil
	}
	mj.volgID++
	mj.week = false
	mj.sessie = &sessie{id: mj.volgID, modus: modus, regels: []regel{{1, m.t(fmt.Sprintf("buddy.open.%d", modus))}}}
	mj.invoer.SetValue("")
	return mj.invoer.Focus()
}

func (m *model) stopSessie() {
	mj := m.mj
	if s := mj.sessie; s != nil && s.verhaal != "" && !s.alleenLezen {
		if s.modus == modusVerhaal {
			m.zeg(m.t("buddy.saved"))
		} else {
			m.zeg(m.t("buddy.hwsaved"))
		}
	}
	mj.sessie = nil
	mj.invoer.Blur()
}

// stuur stuurt een bericht van het kind, na het filter voor persoonsgegevens.
func (m *model) stuur(tekst string, controleer bool) tea.Cmd {
	mj, s := m.mj, m.mj.sessie
	if reden := m.maatjeDicht(); reden != "" {
		s.regels = append(s.regels, regel{2, reden})
		return nil
	}
	if mb := m.maxBeurten(s.modus); mb > 0 && s.beurten >= mb {
		s.regels = append(s.regels, regel{2, m.t("buddy.full")})
		return nil
	}
	if s.modus == modusVraag && m.overVandaag(modusVraag) <= 0 {
		s.regels = append(s.regels, regel{2, m.t("buddy.none")})
		return nil
	}
	schoon, gefilterd := filterPrivé(tekst, m.inst.Naam)
	s.regels = append(s.regels, regel{0, tekst})
	if gefilterd {
		s.regels = append(s.regels, regel{2, m.t("buddy.filtered")})
	}
	// tellen bij het versturen, ook als de vraag straks wordt tegengehouden
	switch {
	case s.modus == modusVraag:
		mj.gebruik.Vragen++
	case !s.geteld && s.modus == modusVerhaal:
		mj.gebruik.Verhalen++
	case !s.geteld:
		mj.gebruik.Huiswerk++
	}
	s.geteld = true
	mj.gebruik.LaatstActief = time.Now()
	mj.gebruik.bewaar()
	s.wacht = true

	ai, id, sess := mj.ai, s.id, *s
	systeem := ai.systeem(m.inst.Maatje, s.modus, m.inst.Taal)
	gesprek := append([]anthropic.BetaMessageParam(nil), s.gesprek...)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		uit := maatjeMsg{id: id, vraag: schoon}
		if controleer {
			o, err := ai.Keur(ctx, schoon)
			if err != nil {
				uit.err = err
				return uit
			}
			if o != oordeelOK {
				uit.oordeelVraag = o
				return uit
			}
		}
		gesprek = append(gesprek, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(metEerder(&sess, schoon))))
		resp, antwoord, err := ai.Antwoord(ctx, systeem, gesprek)
		if errors.Is(err, errGeweigerd) {
			uit.oordeelAnt = oordeelOngepast
			return uit
		}
		if err != nil {
			uit.err = err
			return uit
		}
		o, err := ai.Keur(ctx, antwoord)
		if err != nil {
			uit.err = err
			return uit
		}
		if o != oordeelOK {
			uit.oordeelAnt = o
			return uit
		}
		uit.gesprek = append(gesprek, resp.ToParam())
		uit.antwoord = antwoord
		return uit
	}
}

func (m *model) maatjeBericht(msg maatjeMsg) tea.Cmd {
	mj := m.mj
	s := mj.sessie
	if s == nil || s.id != msg.id {
		return nil
	}
	s.wacht = false
	s.scroll = 0
	oordeel := oordeelOK
	var cmd tea.Cmd
	switch {
	case msg.err != nil:
		s.regels = append(s.regels, regel{2, m.t("buddy.err")})
		return nil
	case msg.oordeelVraag == oordeelZwaar || msg.oordeelAnt == oordeelZwaar:
		oordeel = oordeelZwaar
		s.regels = append(s.regels, regel{2, m.t("buddy.heavy")})
		cmd = stuurNtfy(m.ouders, m.t("buddy.ntfy.title"), m.t("buddy.ntfy.body", m.naamOfKind()))
	case msg.oordeelVraag != "":
		oordeel = oordeelOngepast
		s.regels = append(s.regels, regel{2, m.t("buddy.unsuitable")})
	case msg.oordeelAnt != "":
		oordeel = oordeelOngepast
		s.regels = append(s.regels, regel{2, m.t("buddy.badanswer")})
	default:
		s.gesprek = msg.gesprek
		s.beurten++
		s.regels = append(s.regels, regel{1, msg.antwoord})
		if s.modus != modusVraag {
			m.bewaarSessie(s, msg.vraag, msg.antwoord)
		}
	}
	vraag := msg.vraag
	if s.modus == modusVerhaal && oordeel == oordeelOK {
		vraag = "" // verhalen zijn van het kind; ouders zien alleen dat er geschreven is
	}
	schrijfLog(logRegel{Tijd: time.Now(), Modus: s.modus, Vraag: vraag, Oordeel: oordeel})
	return cmd
}

func (m *model) naamOfKind() string {
	if m.inst.Naam != "" {
		return m.inst.Naam
	}
	return m.t("help.someone")
}

// stuurNtfy stuurt een berichtje naar de telefoons van de ouders (als ntfy is ingesteld).
func stuurNtfy(o Ouders, titel, tekst string) tea.Cmd {
	if o.Ntfy.Topic == "" {
		return nil
	}
	return func() tea.Msg {
		req, err := http.NewRequest("POST", strings.TrimRight(o.Ntfy.URL, "/")+"/"+o.Ntfy.Topic, strings.NewReader(tekst))
		if err != nil {
			return nil
		}
		req.Header.Set("Title", titel)
		req.Header.Set("Tags", "speech_balloon")
		c := http.Client{Timeout: 6 * time.Second}
		if resp, err := c.Do(req); err == nil {
			resp.Body.Close()
		}
		return nil
	}
}

/* ---------------- weergave ---------------- */

func (m model) viewMaatje(p palet) string {
	mj := m.mj
	dim := lipgloss.NewStyle().Foreground(p.dim)
	gezicht := lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render(gezichten[m.inst.Gezicht])
	kop := gezicht + "  " + lipgloss.NewStyle().Bold(true).Render(m.menuNaam(tabMaatje)) + "   " + dim.Render(m.t("buddy.honest"))
	breed := min(m.breed-8, 100)

	if mj.week && m.ouderModus {
		return kop + "\n\n" + paneel(p, m.t("buddy.week"), m.viewWeek(p, breed-4), breed, true)
	}

	g := m.ouders.Maatje
	tijd := dim.Render(m.t("buddy.time", mj.gebruik.Seconden/60, g.MaxMinutenPerDag))
	if mj.sessie == nil && mj.kiesVerhaal {
		vraag, nieuw, titel, eerste := "buddy.pickhw", "buddy.newhw", m.t(modusSleutels[max(mj.kiesModus, 0)]), 1
		switch {
		case mj.kiesModus == modusVerhaal:
			vraag, nieuw = "buddy.pickstory", "buddy.newstory"
		case mj.kiesModus < 0:
			vraag, titel, eerste = "buddy.hwread", m.t("buddy.hwlist"), 0
		}
		regels := []string{dim.Render(m.t(vraag)), ""}
		for i := 1 - eerste; i <= len(mj.verhalen); i++ {
			naam := m.t(nieuw)
			if i > 0 {
				v := mj.verhalen[i-1]
				naam = v.tijd.Format("2 Jan 15:04") + "  " + v.titel
				if mj.kiesModus < 0 {
					naam = v.tijd.Format("Mon 2 Jan 15:04") + "  " + m.t(modusSleutels[v.modus]) + ": " + v.titel
				}
			}
			if i-1+eerste == mj.verhaalSel {
				regels = append(regels, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸ "+naam+" "))
			} else {
				regels = append(regels, "  "+naam)
			}
		}
		regels = append(regels, "", dim.Render(m.t("buddy.pickkeys")))
		return kop + "\n\n" + paneel(p, titel, strings.Join(regels, "\n"), breed, true)
	}
	if mj.sessie == nil {
		var regels []string
		if reden := m.maatjeDicht(); reden != "" {
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Width(breed-4).Render(reden), "")
		}
		for i := 0; i < aantalModi; i++ {
			over := m.overVandaag(i)
			info := m.t("buddy.left", max(over, 0))
			if over <= 0 {
				info = m.t("buddy.none")
			}
			naam := fmt.Sprintf(" %-22s ", m.t(modusSleutels[i]))
			uitleg := m.t(modusSleutels[i]+".d") + " · " + info
			if i == mj.keuze {
				regels = append(regels, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸"+naam)+lipgloss.NewStyle().Background(p.sel).Foreground(p.dim).Render(uitleg+" "))
			} else {
				regels = append(regels, " "+naam+dim.Render(uitleg))
			}
		}
		regels = append(regels, "", tijd, dim.Render(m.t("buddy.start")))
		if m.ouderModus {
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("buddy.weekhint")+" · "+m.t("buddy.hwhint")))
		}
		return kop + "\n\n" + paneel(p, m.t("buddy.choose"), strings.Join(regels, "\n"), breed, true)
	}

	s := mj.sessie
	tekstBreed := breed - 6
	var regels []string
	for _, r := range s.regels {
		switch r.wie {
		case 0:
			regels = append(regels, lipgloss.NewStyle().Foreground(p.acc).Width(tekstBreed).Render(m.t("buddy.you")+": "+r.tekst))
		case 1:
			regels = append(regels, lipgloss.NewStyle().Foreground(p.fg).Width(tekstBreed).Render(m.inst.Maatje+": "+r.tekst))
		default:
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Width(tekstBreed).Render(r.tekst))
		}
		regels = append(regels, "")
	}
	if s.wacht {
		regels = append(regels, dim.Render(m.inst.Maatje+" "+m.t("buddy.thinking")))
	}
	alles := strings.Split(strings.TrimRight(strings.Join(regels, "\n"), "\n"), "\n")
	hoogte := max(m.hoog-12, 4)
	if len(alles) > hoogte {
		// scrollen: s.scroll regels omhoog vanaf het einde, niet verder dan het begin
		s.scroll = min(s.scroll, len(alles)-hoogte)
		eind := len(alles) - s.scroll
		alles = alles[eind-hoogte : eind]
	} else {
		s.scroll = 0
	}
	info := tijd
	if mb := m.maxBeurten(s.modus); mb > 0 {
		info = dim.Render(m.t("buddy.turns", s.beurten, mb)) + dim.Render(" · ") + tijd
	} else {
		info = dim.Render(m.t("buddy.left", max(m.overVandaag(modusVraag), 0))) + dim.Render(" · ") + tijd
	}
	if s.alleenLezen {
		return kop + "\n\n" + paneel(p, m.t(modusSleutels[s.modus]), strings.Join(alles, "\n"), breed, true) + "\n" +
			dim.Render(m.t("buddy.readkeys"))
	}
	return kop + "\n\n" + paneel(p, m.t(modusSleutels[s.modus]), strings.Join(alles, "\n"), breed, true) + "\n" +
		mj.invoer.View() + "\n" + info + "\n" + dim.Render(m.t("buddy.keys"))
}

func (m model) viewWeek(p palet, breed int) string {
	dim := lipgloss.NewStyle().Foreground(p.dim)
	log := leesLog(time.Now().AddDate(0, 0, -7))
	if len(log) == 0 {
		return dim.Render(m.t("buddy.weekempty"))
	}
	var regels []string
	dag := ""
	for _, r := range log {
		if d := r.Tijd.Format("Mon 2 Jan"); d != dag {
			dag = d
			regels = append(regels, lipgloss.NewStyle().Bold(true).Render(d))
		}
		v := r.Vraag
		if v == "" {
			v = m.t("buddy.storyturn")
		}
		if len([]rune(v)) > breed-20 {
			v = string([]rune(v)[:breed-21]) + "…"
		}
		regel := r.Tijd.Format("15:04") + " " + m.t(modusSleutels[r.Modus]) + ": " + v
		switch r.Oordeel {
		case oordeelZwaar:
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render("! "+regel))
		case oordeelOngepast:
			regels = append(regels, lipgloss.NewStyle().Foreground(p.rood).Render("× "+regel))
		default:
			regels = append(regels, "  "+regel)
		}
	}
	if len(regels) > m.hoog-10 {
		regels = regels[len(regels)-(m.hoog-10):]
	}
	return strings.Join(regels, "\n") + "\n\n" + dim.Render(m.t("buddy.weeklegend"))
}
