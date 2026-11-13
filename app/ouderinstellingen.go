package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// Ouderinstellingen in de app. ouders.yaml is van root; de app draait als het kind. Wijzigen gaat
// daarom via "unscroll --ouder-zet", dat als root mag draaien (een vaste sudo-regel) en zelf de
// pincode controleert. Alleen de instellingen hieronder mogen, met geldige waarden.

type instSoort int

const (
	instBool instSoort = iota
	instGetal
	instTijd
	instKeuze
	instTekst
	instKomma
	instLijst
)

type instelling struct {
	sleutel   string // pad in ouders.yaml, met punten
	label     string // vertaalsleutel
	soort     instSoort
	min, max  int
	stap      int
	keuzes    []string
	verborgen bool // wel toegestaan, niet in de lijst (zoals de coördinaten)
}

var instellingen = []instelling{
	{sleutel: "avondslot.schooldagen", label: "pset.school", soort: instTijd},
	{sleutel: "avondslot.weekend", label: "pset.weekend", soort: instTijd},
	{sleutel: "avondslot.open", label: "pset.open", soort: instTijd},
	{sleutel: "avondslot.lezen_mag", label: "pset.readok", soort: instBool},
	{sleutel: "avondslot.luisterboek_mag", label: "pset.bookok", soort: instBool},
	{sleutel: "max_volume", label: "pset.maxvol", soort: instGetal, min: 10, max: 100, stap: 5},
	{sleutel: "maatje.aan", label: "pset.buddyon", soort: instBool},
	{sleutel: "maatje.model", label: "pset.model", soort: instKeuze, keuzes: []string{"claude-haiku-5-5", "claude-sonnet-5-5", "claude-opus-5-5"}},
	{sleutel: "maatje.leeftijd", label: "pset.age", soort: instGetal, min: 6, max: 18, stap: 1},
	{sleutel: "maatje.max_minuten_per_dag", label: "pset.minutes", soort: instGetal, min: 0, max: 600, stap: 5},
	{sleutel: "maatje.pauze_na_minuten", label: "pset.pause", soort: instGetal, min: 0, max: 120, stap: 5},
	{sleutel: "maatje.vrije_vragen_per_dag", label: "pset.questions", soort: instGetal, min: 0, max: 200, stap: 5},
	{sleutel: "maatje.huiswerk_sessies_per_dag", label: "pset.homework", soort: instGetal, min: 0, max: 20, stap: 1},
	{sleutel: "maatje.beurten_per_huiswerksessie", label: "pset.hwturns", soort: instGetal, min: 0, max: 200, stap: 5},
	{sleutel: "maatje.verhaal_sessies_per_dag", label: "pset.stories", soort: instGetal, min: 0, max: 20, stap: 1},
	{sleutel: "maatje.beurten_per_verhaalsessie", label: "pset.stturns", soort: instGetal, min: 0, max: 200, stap: 5},
	{sleutel: "vandaag.plaats", label: "pset.place", soort: instTekst},
	{sleutel: "vandaag.breedtegraad", soort: instKomma, verborgen: true},
	{sleutel: "vandaag.lengtegraad", soort: instKomma, verborgen: true},
	{sleutel: "spotify.zoeken", label: "pset.spsearch", soort: instBool},
	{sleutel: "spotify.expliciet_mag", label: "pset.spexplicit", soort: instBool},
	{sleutel: "spotify.goedgekeurd", label: "pset.spapproved", soort: instLijst},
}

func zoekInstelling(sleutel string) (instelling, bool) {
	for _, i := range instellingen {
		if i.sleutel == sleutel {
			return i, true
		}
	}
	return instelling{}, false
}

var tijdPatroon = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// controleer: is de waarde geldig voor deze instelling? Geeft de waarde terug zoals die in YAML komt.
func (i instelling) controleer(w string) (string, string, error) {
	w = strings.TrimSpace(w)
	switch i.soort {
	case instBool:
		if w != "true" && w != "false" {
			return "", "", errors.New("true or false")
		}
		return w, "!!bool", nil
	case instGetal:
		n, err := strconv.Atoi(w)
		if err != nil || n < i.min || n > i.max {
			return "", "", fmt.Errorf("a number from %d to %d", i.min, i.max)
		}
		return strconv.Itoa(n), "!!int", nil
	case instKomma:
		f, err := strconv.ParseFloat(w, 64)
		if err != nil || f < -180 || f > 180 {
			return "", "", errors.New("a coordinate")
		}
		return strconv.FormatFloat(f, 'f', 4, 64), "!!float", nil
	case instTijd:
		if !tijdPatroon.MatchString(w) {
			return "", "", errors.New("a time like 20:00")
		}
		return w, "!!str", nil
	case instKeuze:
		for _, k := range i.keuzes {
			if k == w {
				return w, "!!str", nil
			}
		}
		return "", "", errors.New("not one of the choices")
	case instTekst:
		if len([]rune(w)) > 60 || strings.ContainsAny(w, "\n\r") {
			return "", "", errors.New("too long")
		}
		return w, "!!str", nil
	}
	return "", "", errors.New("unknown")
}

/* ---------------- de root-kant: unscroll --ouder-zet ---------------- */

type wijziging struct {
	Sleutel string `json:"sleutel"`
	Waarde  string `json:"waarde"`
}

type zetVerzoek struct {
	Pin         string      `json:"pin"`
	Wijzigingen []wijziging `json:"wijzigingen"`
}

// ouderZet draait als root. Het verzoek komt via stdin, zodat de pincode niet in de proceslijst staat.
func ouderZet(in io.Reader) error {
	var v zetVerzoek
	if err := json.NewDecoder(io.LimitReader(in, 64<<10)).Decode(&v); err != nil {
		return err
	}
	o, err := laadOuders()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !o.pinKlopt(v.Pin) {
		return errors.New("wrong PIN")
	}
	pad := oudersPad()
	data, err := os.ReadFile(pad)
	if err != nil {
		return err
	}
	nieuw, err := pasYAMLAan(data, v.Wijzigingen)
	if err != nil {
		return err
	}
	// controleren dat het resultaat nog te lezen is, en dan in één keer vervangen
	var proef Ouders
	if err := yaml.Unmarshal(nieuw, &proef); err != nil {
		return fmt.Errorf("result not valid: %w", err)
	}
	perm := os.FileMode(0o644)
	if info, err := os.Stat(pad); err == nil {
		perm = info.Mode().Perm()
	}
	return veiligSchrijven(pad, nieuw, perm)
}

// pasYAMLAan wijzigt waarden in ouders.yaml en laat commentaar en volgorde staan.
func pasYAMLAan(data []byte, wijzigingen []wijziging) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	wortel := doc.Content[0]
	for _, w := range wijzigingen {
		inst, ok := zoekInstelling(w.Sleutel)
		if !ok {
			return nil, fmt.Errorf("%s may not be changed here", w.Sleutel)
		}
		delen := strings.Split(w.Sleutel, ".")
		ouder := wortel
		for _, d := range delen[:len(delen)-1] {
			ouder = kindVan(ouder, d, yaml.MappingNode)
		}
		laatste := delen[len(delen)-1]
		if inst.soort == instLijst {
			lijst := kindVan(ouder, laatste, yaml.SequenceNode)
			lijst.Style = 0 // geen [] meer
			switch {
			case strings.HasPrefix(w.Waarde, "+"):
				link := strings.TrimSpace(w.Waarde[1:])
				if _, _, ok := spotifyLink(link); !ok {
					return nil, errors.New("not a Spotify album or playlist link")
				}
				lijst.Content = append(lijst.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: link, Style: yaml.DoubleQuotedStyle})
			case strings.HasPrefix(w.Waarde, "-"):
				n, err := strconv.Atoi(w.Waarde[1:])
				if err != nil || n < 0 || n >= len(lijst.Content) {
					return nil, errors.New("no such item")
				}
				lijst.Content = append(lijst.Content[:n], lijst.Content[n+1:]...)
			default:
				return nil, errors.New("use +link or -number")
			}
			continue
		}
		waarde, tag, err := inst.controleer(w.Waarde)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", w.Sleutel, err)
		}
		n := kindVan(ouder, laatste, yaml.ScalarNode)
		n.Kind, n.Tag, n.Value, n.Content = yaml.ScalarNode, tag, waarde, nil
		n.Style = 0
		if tag == "!!str" {
			n.Style = yaml.DoubleQuotedStyle
		}
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// kindVan zoekt (of maakt) de waarde bij een sleutel in een mapping.
func kindVan(m *yaml.Node, sleutel string, soort yaml.Kind) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == sleutel {
			w := m.Content[i+1]
			if w.Kind != soort && soort != yaml.ScalarNode {
				w.Kind, w.Tag, w.Value = soort, map[yaml.Kind]string{yaml.MappingNode: "!!map", yaml.SequenceNode: "!!seq"}[soort], ""
			}
			return w
		}
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sleutel}
	w := &yaml.Node{Kind: soort}
	switch soort {
	case yaml.MappingNode:
		w.Tag = "!!map"
	case yaml.SequenceNode:
		w.Tag = "!!seq"
	}
	m.Content = append(m.Content, k, w)
	return w
}

/* ---------------- de app-kant ---------------- */

// ouderZetCmd vraagt de root-helper om wijzigingen door te voeren (via sudo, zonder wachtwoord).
var ouderZetUitvoeren = func(verzoek []byte) ([]byte, error) {
	cmd := exec.Command("sudo", "-n", "/usr/local/bin/unscroll", "--ouder-zet")
	cmd.Stdin = bytes.NewReader(verzoek)
	return cmd.CombinedOutput()
}

type instMsg struct {
	err   error
	tekst string
}

func (m *model) zetInstellingen(w []wijziging) tea.Cmd {
	data, _ := json.Marshal(zetVerzoek{Pin: m.ouderPin, Wijzigingen: w})
	return func() tea.Msg {
		uit, err := ouderZetUitvoeren(data)
		if err != nil {
			return instMsg{err: fmt.Errorf("%s", strings.TrimSpace(string(uit)+" "+err.Error()))}
		}
		return instMsg{}
	}
}

type ouderInstStaat struct {
	open     bool
	sel      int
	bewerken bool
	invoer   textinput.Model
	lijstSel int
	inLijst  bool
	melding  string
	bezig    bool
}

func nieuweOuderInst() *ouderInstStaat {
	s := &ouderInstStaat{}
	s.invoer = textinput.New()
	s.invoer.CharLimit = 200
	return s
}

func zichtbareInstellingen() []instelling {
	var uit []instelling
	for _, i := range instellingen {
		if !i.verborgen {
			uit = append(uit, i)
		}
	}
	return uit
}

// huidigeWaarde: wat er nu staat, als tekst.
func (m *model) huidigeWaarde(sleutel string) string {
	o := m.ouders
	a, mj := o.Avondslot, o.Maatje
	waarden := map[string]any{
		"avondslot.schooldagen": a.Schooldagen, "avondslot.weekend": a.Weekend, "avondslot.open": a.Open,
		"avondslot.lezen_mag": a.LezenMag, "avondslot.luisterboek_mag": a.LuisterboekMag, "max_volume": o.MaxVolume,
		"maatje.aan": mj.Aan, "maatje.model": mj.Model, "maatje.leeftijd": mj.Leeftijd,
		"maatje.max_minuten_per_dag": mj.MaxMinutenPerDag, "maatje.pauze_na_minuten": mj.PauzeNaMinuten,
		"maatje.vrije_vragen_per_dag": mj.VrijeVragenPerDag, "maatje.huiswerk_sessies_per_dag": mj.HuiswerkSessiesPerDag,
		"maatje.beurten_per_huiswerksessie": mj.BeurtenPerHuiswerk, "maatje.verhaal_sessies_per_dag": mj.VerhaalSessiesPerDag,
		"maatje.beurten_per_verhaalsessie": mj.BeurtenPerVerhaal, "vandaag.plaats": o.plaatsNaam(),
		"spotify.zoeken": o.Spotify.Zoeken, "spotify.expliciet_mag": o.Spotify.ExplicietMag,
		"spotify.goedgekeurd": len(o.Spotify.Goedgekeurd),
	}
	return fmt.Sprint(waarden[sleutel])
}

func (m *model) toetsOuderInst(msg tea.KeyMsg) (tea.Cmd, bool) {
	s := m.oi
	k := msg.String()
	lijst := zichtbareInstellingen()
	inst := lijst[s.sel]
	if s.bezig {
		return nil, true
	}
	if s.bewerken {
		switch k {
		case "esc":
			s.bewerken = false
			s.invoer.Blur()
		case "enter":
			w := strings.TrimSpace(s.invoer.Value())
			s.bewerken = false
			s.invoer.Blur()
			if w == "" {
				return nil, true
			}
			s.bezig = true
			switch {
			case inst.soort == instLijst:
				return m.zetInstellingen([]wijziging{{inst.sleutel, "+" + w}}), true
			case inst.sleutel == "vandaag.plaats":
				return m.zoekPlaats(w), true
			default:
				return m.zetInstellingen([]wijziging{{inst.sleutel, w}}), true
			}
		default:
			var cmd tea.Cmd
			s.invoer, cmd = s.invoer.Update(msg)
			return cmd, true
		}
		return nil, true
	}
	if s.inLijst {
		links := m.ouders.Spotify.Goedgekeurd
		switch k {
		case "up", "k":
			s.lijstSel = max(s.lijstSel-1, 0)
		case "down", "j":
			s.lijstSel = min(s.lijstSel+1, max(len(links)-1, 0))
		case "a", "+", "n":
			s.bewerken = true
			s.invoer.SetValue("")
			s.invoer.Prompt = m.t("pset.pastelink")
			return s.invoer.Focus(), true
		case "delete", "backspace", "-", "x":
			if s.lijstSel < len(links) {
				s.bezig = true
				return m.zetInstellingen([]wijziging{{inst.sleutel, fmt.Sprintf("-%d", s.lijstSel)}}), true
			}
		case "esc":
			s.inLijst = false
		}
		return nil, true
	}
	verander := func(stap int) tea.Cmd {
		huidig := m.huidigeWaarde(inst.sleutel)
		var nieuw string
		switch inst.soort {
		case instBool:
			nieuw = strconv.FormatBool(huidig != "true")
		case instGetal:
			n, _ := strconv.Atoi(huidig)
			nieuw = strconv.Itoa(min(max(n+stap*inst.stap, inst.min), inst.max))
		case instKeuze:
			i := 0
			for j, c := range inst.keuzes {
				if c == huidig {
					i = j
				}
			}
			nieuw = inst.keuzes[(i+stap+len(inst.keuzes))%len(inst.keuzes)]
		case instTijd:
			t, err := time.Parse("15:04", huidig)
			if err != nil {
				t, _ = time.Parse("15:04", "20:00")
			}
			nieuw = t.Add(time.Duration(stap*15) * time.Minute).Format("15:04")
		default:
			return nil
		}
		if nieuw == huidig {
			return nil
		}
		s.bezig = true
		return m.zetInstellingen([]wijziging{{inst.sleutel, nieuw}})
	}
	switch k {
	case "up", "k":
		s.sel = max(s.sel-1, 0)
	case "down", "j":
		s.sel = min(s.sel+1, len(lijst)-1)
	case "left":
		return verander(-1), true
	case "right", " ":
		return verander(1), true
	case "enter":
		switch inst.soort {
		case instLijst:
			s.inLijst, s.lijstSel = true, 0
		case instBool, instKeuze:
			return verander(1), true
		default:
			s.bewerken = true
			s.invoer.SetValue(m.huidigeWaarde(inst.sleutel))
			s.invoer.Prompt = m.t(inst.label) + ": "
			s.invoer.CursorEnd()
			return s.invoer.Focus(), true
		}
	case "esc":
		s.open = false
	}
	return nil, true
}

// zoekPlaats zoekt de coördinaten bij een plaatsnaam (Open-Meteo) en zet ze alle drie.
var geoBasis = "https://geocoding-api.open-meteo.com/v1/search"

func (m *model) zoekPlaats(naam string) tea.Cmd {
	pin := m.ouderPin
	taal := m.inst.Taal
	return func() tea.Msg {
		q := url.Values{"name": {naam}, "count": {"1"}, "language": {taal}, "format": {"json"}}
		c := http.Client{Timeout: 10 * time.Second}
		resp, err := c.Get(geoBasis + "?" + q.Encode())
		if err != nil {
			return instMsg{err: err}
		}
		defer resp.Body.Close()
		var r struct {
			Results []struct {
				Name      string  `json:"name"`
				Country   string  `json:"country"`
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"results"`
		}
		if json.NewDecoder(resp.Body).Decode(&r) != nil || len(r.Results) == 0 {
			return instMsg{err: errors.New("place not found")}
		}
		g := r.Results[0]
		data, _ := json.Marshal(zetVerzoek{Pin: pin, Wijzigingen: []wijziging{
			{"vandaag.plaats", naam}, {"vandaag.breedtegraad", fmt.Sprint(g.Latitude)}, {"vandaag.lengtegraad", fmt.Sprint(g.Longitude)}}})
		if uit, err := ouderZetUitvoeren(data); err != nil {
			return instMsg{err: fmt.Errorf("%s", strings.TrimSpace(string(uit)+" "+err.Error()))}
		}
		return instMsg{tekst: g.Name + ", " + g.Country}
	}
}

// instBericht: na een wijziging ouders.yaml opnieuw lezen, zodat het meteen werkt.
func (m *model) instBericht(msg instMsg) tea.Cmd {
	s := m.oi
	s.bezig = false
	if msg.err != nil {
		s.melding = m.t("pset.failed", msg.err.Error())
		return nil
	}
	o, err := laadOuders()
	if err != nil {
		s.melding = m.t("pset.failed", err.Error())
		return nil
	}
	oudModel := m.ouders.Maatje.Model
	m.ouders, m.oudersFout = o, nil
	if o.Maatje.Model != oudModel || (m.mj.ai == nil) != !o.Maatje.Aan {
		m.mj.ai = nieuwMaatjeAI(laadGeheimen(), o)
	}
	m.vd.weer = nil // weer voor de nieuwe plek
	s.melding = m.t("pset.saved")
	if msg.tekst != "" {
		s.melding = m.t("pset.placefound", msg.tekst)
	}
	return nil
}

func (m model) viewOuderInst(p palet) string {
	s := m.oi
	dim := lipgloss.NewStyle().Foreground(p.dim)
	breed := min(m.breed-8, 90)
	var r []string
	if s.inLijst {
		links := m.ouders.Spotify.Goedgekeurd
		if len(links) == 0 {
			r = append(r, dim.Render(m.t("pset.nolinks")))
		}
		for i, l := range links {
			if i == s.lijstSel {
				r = append(r, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Render("▸ "+l+" "))
			} else {
				r = append(r, "  "+l)
			}
		}
		if s.bewerken {
			r = append(r, "", s.invoer.View())
		}
		r = append(r, "", dim.Render(m.t("pset.listkeys")))
	} else {
		lijst := zichtbareInstellingen()
		hoogte := max(m.hoog-10, 6)
		begin := min(max(s.sel-hoogte/2, 0), max(len(lijst)-hoogte, 0))
		for i := begin; i < len(lijst) && i < begin+hoogte; i++ {
			inst := lijst[i]
			w := m.huidigeWaarde(inst.sleutel)
			switch {
			case inst.soort == instBool && w == "true":
				w = m.t("yes")
			case inst.soort == instBool:
				w = m.t("no")
			case inst.soort == instLijst:
				w = m.t("pset.links", w)
			case strings.HasSuffix(inst.sleutel, "_per_dag") || strings.Contains(inst.sleutel, "beurten") || strings.HasSuffix(inst.sleutel, "minuten"):
				if w == "0" {
					w = m.t("pset.nolimit")
				}
			}
			label := fmt.Sprintf("%-34s", m.t(inst.label))
			if i == s.sel {
				if s.bewerken {
					r = append(r, "▸ "+s.invoer.View())
				} else {
					r = append(r, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸ "+label+" ‹ "+w+" › "))
				}
			} else {
				r = append(r, "  "+label+dim.Render("  "+w))
			}
		}
		r = append(r, "", dim.Render(m.t("pset.keys")))
	}
	if s.bezig {
		r = append(r, "", dim.Render(m.t("listen.loading")))
	} else if s.melding != "" {
		r = append(r, "", lipgloss.NewStyle().Foreground(p.geel).Width(breed-6).Render(s.melding))
	}
	return lipgloss.NewStyle().Bold(true).Render(m.t("pset.title")) + "\n\n" + paneel(p, m.t("sys.parentmode"), strings.Join(r, "\n"), breed, true)
}
