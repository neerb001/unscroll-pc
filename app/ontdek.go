package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Ontdekken: dier van de dag, computerlab en natuurdagboek.

const (
	ontdekMenu = iota
	ontdekDier
	ontdekLab
	ontdekDagboek
)

var ontdekSleutels = []string{"", "disc.animal", "disc.lab", "disc.journal"}

type ontdekStaat struct {
	scherm, keuze int
	// dier van de dag
	dagTerug int // 0 = vandaag, 1 = gisteren, ...
	dier     *dierInfo
	dierFout string
	laden    bool
	// lab
	lab       *labWereld
	labRegels []string
	invoer    textinput.Model
	// natuurdagboek
	notities  []notitie
	schrijven bool
}

func nieuweOntdek() *ontdekStaat {
	o := &ontdekStaat{}
	o.invoer = textinput.New()
	o.invoer.CharLimit = 300
	return o
}

/* ---------------- dier van de dag ---------------- */

// De lijst staat in de app: Wikipedia kiest niets. [Nederlandse titel, Engelse titel]
var dieren = [][2]string{
	{"Egel", "Hedgehog"}, {"Octopussen", "Octopus"}, {"Gewone pad", "Common toad"}, {"Bultrug", "Humpback whale"},
	{"Steenuil", "Little owl"}, {"Bever", "Beaver"}, {"Axolotl", "Axolotl"}, {"Honingbij", "Western honey bee"},
	{"Zeepaardjes", "Seahorse"}, {"Das (dier)", "European badger"}, {"Kameleons", "Chameleon"}, {"Pinguïns", "Penguin"},
	{"Eekhoorn", "Red squirrel"}, {"Gewone tuinslak", "Cornu aspersum"}, {"Olifanten", "Elephant"}, {"Vos", "Red fox"},
	{"Kolibries", "Hummingbird"}, {"Reuzenpanda", "Giant panda"}, {"Mier", "Ant"}, {"Gierzwaluw", "Common swift"},
	{"Spechtvinken", "Woodpecker finch"}, {"Baardagame", "Pogona"}, {"Zeehond", "Harbor seal"}, {"Dolfijnen", "Dolphin"},
	{"Libellen", "Dragonfly"}, {"Wombats", "Wombat"}, {"Ijsvogel", "Common kingfisher"}, {"Lieveheersbeestjes", "Coccinellidae"},
	{"Schildpadden", "Turtle"}, {"Oehoe", "Eurasian eagle-owl"}, {"Luiaards", "Sloth"}, {"Kwallen", "Jellyfish"},
	{"Raaf", "Common raven"}, {"Ree", "Roe deer"}, {"Vleermuizen", "Bat"}, {"Kangoeroes", "Kangaroo"},
	{"Ooievaar", "White stork"}, {"Haaien", "Shark"}, {"Regenworm", "Earthworm"}, {"Flamingo's", "Flamingo"},
	{"Otter", "Eurasian otter"}, {"Giraffe", "Giraffe"}, {"Slakken", "Snail"}, {"Pauw", "Indian peafowl"},
	{"Zeeotter", "Sea otter"}, {"Wolf", "Wolf"}, {"Mol (dier)", "European mole"}, {"Koolmees", "Great tit"},
	{"Kraanvogel", "Common crane"}, {"Tapirs", "Tapir"}, {"Okapi", "Okapi"}, {"Kikkers", "Frog"},
	{"Rendier", "Reindeer"}, {"Gekko's", "Gecko"}, {"Walrus", "Walrus"}, {"Koala", "Koala"},
	{"Merel", "Common blackbird"}, {"Mantelbavianen", "Hamadryas baboon"}, {"Narwal", "Narwhal"}, {"Vuurvliegjes", "Firefly"},
}

type dierInfo struct {
	Titel, Omschrijving, Tekst string
}

func dierVanDag(d time.Time) [2]string {
	dag := d.YearDay() + d.Year()*366
	return dieren[dag%len(dieren)]
}

func dierenCache() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "unscroll", "dieren")
}

// wikiBasis is los te zetten voor tests.
var wikiBasis = "https://%s.wikipedia.org/api/rest_v1/page/summary/"

func haalDier(taal string, d time.Time) tea.Cmd {
	return func() tea.Msg {
		titels := dierVanDag(d)
		pogingen := [][2]string{{"en", titels[1]}}
		if taal == "nl" {
			// bestaat de Nederlandse pagina niet, dan de Engelse
			pogingen = [][2]string{{"nl", titels[0]}, {"en", titels[1]}}
		}
		var laatste error
		for _, p := range pogingen {
			info, err := wikiSamenvatting(p[0], p[1])
			if err == nil {
				return dierMsg{info: info}
			}
			laatste = err
		}
		return dierMsg{err: laatste}
	}
}

func wikiSamenvatting(taal, titel string) (*dierInfo, error) {
	cache := filepath.Join(dierenCache(), taal+"-"+strings.ReplaceAll(titel, "/", "_")+".json")
	var info dierInfo
	if data, err := os.ReadFile(cache); err == nil && json.Unmarshal(data, &info) == nil && info.Tekst != "" {
		return &info, nil
	}
	req, _ := http.NewRequest("GET", fmt.Sprintf(wikiBasis, taal)+url.PathEscape(strings.ReplaceAll(titel, " ", "_")), nil)
	req.Header.Set("User-Agent", "Unscroll/"+versie+" (calm kids computer)")
	c := http.Client{Timeout: 10 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r struct {
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Extract     string `json:"extract"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&r) != nil || r.Extract == "" || r.Type == "disambiguation" {
		return nil, fmt.Errorf("wikipedia: %s", resp.Status)
	}
	info = dierInfo{r.Title, r.Description, r.Extract}
	_ = os.MkdirAll(dierenCache(), 0o755)
	data, _ := json.Marshal(info)
	_ = os.WriteFile(cache, data, 0o644)
	return &info, nil
}

type dierMsg struct {
	info *dierInfo
	err  error
}

/* ---------------- natuurdagboek ---------------- */

type notitie struct {
	Tijd  time.Time `json:"tijd"`
	Tekst string    `json:"tekst"`
}

func dagboekPad() string { return filepath.Join(unscrollData(""), "natuurdagboek.jsonl") }

func laadNotities() []notitie {
	f, err := os.Open(dagboekPad())
	if err != nil {
		return nil
	}
	defer f.Close()
	var uit []notitie
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var n notitie
		if json.Unmarshal(sc.Bytes(), &n) == nil {
			uit = append([]notitie{n}, uit...) // nieuwste eerst
		}
	}
	return uit
}

func bewaarNotitie(n notitie) {
	_ = os.MkdirAll(filepath.Dir(dagboekPad()), 0o700)
	f, err := os.OpenFile(dagboekPad(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	data, _ := json.Marshal(n)
	_, _ = f.Write(append(data, '\n'))
}

/* ---------------- toetsen ---------------- */

func (m *model) ontdekOpen() tea.Cmd { return nil }

func (m *model) toetsOntdek(msg tea.KeyMsg) (tea.Cmd, bool) {
	o := m.od
	k := msg.String()
	switch o.scherm {
	case ontdekMenu:
		switch k {
		case "up", "k":
			o.keuze = (o.keuze + 2) % 3
		case "down", "j":
			o.keuze = (o.keuze + 1) % 3
		case "enter":
			o.scherm = o.keuze + 1
			switch o.scherm {
			case ontdekDier:
				o.dagTerug, o.laden, o.dier, o.dierFout = 0, true, nil, ""
				return haalDier(m.inst.Taal, time.Now()), true
			case ontdekLab:
				if o.lab == nil {
					o.lab = laadLab()
					o.labRegels = []string{m.t("disc.labwelcome"), "", "▸ " + o.lab.missieTekst(m.inst.Taal)}
				}
				o.invoer.SetValue("")
				o.invoer.Prompt = ""
				return o.invoer.Focus(), true
			case ontdekDagboek:
				o.notities = laadNotities()
			}
		default:
			return nil, false
		}
		return nil, true
	case ontdekDier:
		switch k {
		case "left", "right":
			if k == "left" {
				o.dagTerug = min(o.dagTerug+1, 6)
			} else {
				o.dagTerug = max(o.dagTerug-1, 0)
			}
			o.laden, o.dier, o.dierFout = true, nil, ""
			return haalDier(m.inst.Taal, time.Now().AddDate(0, 0, -o.dagTerug)), true
		case "esc":
			o.scherm = ontdekMenu
			return nil, true
		}
		return nil, false
	case ontdekLab:
		switch k {
		case "esc":
			o.scherm = ontdekMenu
			o.invoer.Blur()
			o.lab.bewaar()
			return nil, true
		case "enter":
			regel := strings.TrimSpace(o.invoer.Value())
			o.invoer.SetValue("")
			o.labRegels = append(o.labRegels, m.labPrompt()+regel)
			switch regel {
			case "clear":
				o.labRegels = nil
			case "opnieuw", "restart":
				o.lab = nieuweLabWereld()
				o.labRegels = []string{m.t("disc.labwelcome"), "", "▸ " + o.lab.missieTekst(m.inst.Taal)}
			default:
				uit, gelukt := o.lab.voerUit(regel, m.inst.Taal)
				if uit != "" {
					o.labRegels = append(o.labRegels, strings.Split(uit, "\n")...)
				}
				if gelukt {
					o.labRegels = append(o.labRegels, "", "✓ "+m.t("disc.labdone"), "▸ "+o.lab.missieTekst(m.inst.Taal))
				}
			}
			o.lab.bewaar()
			return nil, true
		}
		var cmd tea.Cmd
		o.invoer, cmd = o.invoer.Update(msg)
		return cmd, true
	case ontdekDagboek:
		if o.schrijven {
			switch k {
			case "esc":
				o.schrijven = false
				o.invoer.Blur()
			case "enter":
				if t := strings.TrimSpace(o.invoer.Value()); t != "" {
					n := notitie{time.Now(), t}
					bewaarNotitie(n)
					o.notities = append([]notitie{n}, o.notities...)
				}
				o.schrijven = false
				o.invoer.Blur()
			default:
				var cmd tea.Cmd
				o.invoer, cmd = o.invoer.Update(msg)
				return cmd, true
			}
			return nil, true
		}
		switch k {
		case "n", "enter":
			o.schrijven = true
			o.invoer.SetValue("")
			o.invoer.Prompt = "✎ "
			return o.invoer.Focus(), true
		case "esc":
			o.scherm = ontdekMenu
			return nil, true
		}
		return nil, false
	}
	return nil, false
}

func (m *model) labPrompt() string {
	naam := strings.ToLower(strings.Fields(m.inst.Naam + " kind")[0])
	return naam + "@" + strings.ToLower(m.inst.Computer) + ":" + m.od.lab.kort(m.od.lab.Cwd) + "$ "
}

/* ---------------- weergave ---------------- */

func (m model) viewOntdek(p palet) string {
	o := m.od
	dim := lipgloss.NewStyle().Foreground(p.dim)
	breed := min(m.breed-8, 100)
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.discover"))
	switch o.scherm {
	case ontdekDier:
		var inhoud string
		switch {
		case o.laden:
			inhoud = dim.Render(m.t("listen.loading"))
		case o.dierFout != "":
			inhoud = lipgloss.NewStyle().Foreground(p.geel).Render(o.dierFout)
		case o.dier != nil:
			inhoud = lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(o.dier.Titel) + "\n" +
				dim.Render(o.dier.Omschrijving) + "\n\n" + lipgloss.NewStyle().Width(breed-6).Render(o.dier.Tekst) +
				"\n\n" + dim.Render(m.t("disc.wiki"))
		}
		dag := m.t("disc.today")
		if o.dagTerug > 0 {
			dag = time.Now().AddDate(0, 0, -o.dagTerug).Format("Mon 2 Jan")
		}
		return kop + "\n\n" + paneel(p, m.t("disc.animal")+" · "+dag, inhoud, breed, true) + "\n" + dim.Render(m.t("disc.animalkeys"))
	case ontdekLab:
		h := max(m.hoog-10, 4)
		regels := o.labRegels
		breedRegels := []string{}
		for _, r := range regels {
			breedRegels = append(breedRegels, strings.Split(lipgloss.NewStyle().Width(breed-6).Render(r), "\n")...)
		}
		if len(breedRegels) > h {
			breedRegels = breedRegels[len(breedRegels)-h:]
		}
		scherm := strings.Join(breedRegels, "\n") + "\n" + lipgloss.NewStyle().Foreground(p.groen).Render(m.labPrompt()) + o.invoer.View()
		return kop + "\n\n" + paneel(p, m.t("disc.lab"), scherm, breed, true) + "\n" + dim.Render(m.t("disc.labkeys"))
	case ontdekDagboek:
		var regels []string
		if o.schrijven {
			regels = append(regels, m.t("disc.journalask"), o.invoer.View(), "")
		}
		if len(o.notities) == 0 && !o.schrijven {
			regels = append(regels, dim.Render(m.t("disc.journalempty")))
		}
		for i, n := range o.notities {
			if i >= max(m.hoog-14, 3) {
				break
			}
			regels = append(regels, lipgloss.NewStyle().Foreground(p.acc).Render(n.Tijd.Format("Mon 2 Jan 15:04")), lipgloss.NewStyle().Width(breed-6).Render(n.Tekst), "")
		}
		return kop + "\n\n" + paneel(p, m.t("disc.journal"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("disc.journalkeys"))
	}
	var regels []string
	for i := 1; i <= 3; i++ {
		naam := fmt.Sprintf(" %-18s ", m.t(ontdekSleutels[i]))
		uitleg := m.t(ontdekSleutels[i] + ".d")
		if i-1 == o.keuze {
			regels = append(regels, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸"+naam)+lipgloss.NewStyle().Background(p.sel).Foreground(p.dim).Render(uitleg+" "))
		} else {
			regels = append(regels, " "+naam+dim.Render(uitleg))
		}
	}
	regels = append(regels, "", dim.Render(m.t("buddy.start")), dim.Render(m.t("disc.later")))
	return kop + "\n\n" + paneel(p, m.t("buddy.choose"), strings.Join(regels, "\n"), breed, true)
}

func (m *model) ontdekBericht(msg dierMsg) {
	o := m.od
	o.laden = false
	if msg.err != nil {
		o.dierFout = m.t("listen.err", msg.err.Error())
		return
	}
	o.dier = msg.info
}
