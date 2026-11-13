package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Woordenboek: een woord opzoeken in Wiktionary (Nederlands of Engels), met woordsoort,
// betekenissen en synoniemen. Bestaat het woord niet, dan "Bedoel je ...?" als hulp bij spelling.
// Betekenissen met een label als vulgair of scheldwoord laat de app weg.

type woordUitleg struct {
	Woord        string   `json:"woord"`
	Soorten      []string `json:"soorten"`
	Betekenissen []string `json:"betekenissen"`
	Synoniemen   []string `json:"synoniemen"`
	Suggesties   []string `json:"suggesties"`
}

// wiktBasis is los te zetten voor tests: %s is de taal (nl of en).
var wiktBasis = "https://%s.wiktionary.org"

func woordCache(taal, woord string) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "unscroll", "woorden", taal+"-"+url.PathEscape(strings.ToLower(woord))+".json")
}

func wiktVraag(adres string, uit any) (int, error) {
	req, _ := http.NewRequest("GET", adres, nil)
	req.Header.Set("User-Agent", wikiAgent())
	c := http.Client{Timeout: 10 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("wiktionary: %s", resp.Status)
	}
	return resp.StatusCode, json.NewDecoder(resp.Body).Decode(uit)
}

func zoekWoord(taal, woord string) (woordUitleg, error) {
	woord = strings.TrimSpace(woord)
	if taal != "nl" {
		taal = "en"
	}
	var u woordUitleg
	if data, err := os.ReadFile(woordCache(taal, woord)); err == nil && json.Unmarshal(data, &u) == nil {
		return u, nil
	}
	basis := fmt.Sprintf(wiktBasis, taal)
	u = woordUitleg{Woord: woord}
	var err error
	if taal == "nl" {
		err = zoekNL(basis, &u)
	} else {
		err = zoekEN(basis, &u)
	}
	if err != nil {
		return u, err
	}
	if len(u.Betekenissen) == 0 {
		u.Suggesties = suggesties(basis, woord)
	}
	_ = os.MkdirAll(filepath.Dir(woordCache(taal, woord)), 0o755)
	if data, err := json.Marshal(u); err == nil {
		_ = veiligSchrijven(woordCache(taal, woord), data, 0o644)
	}
	return u, nil
}

// suggesties: woorden die er op lijken (voor spelling), via de zoekfunctie van Wiktionary.
func suggesties(basis, woord string) []string {
	q := url.Values{"action": {"opensearch"}, "search": {woord}, "limit": {"6"}, "namespace": {"0"}, "format": {"json"}}
	var r []any
	if _, err := wiktVraag(basis+"/w/api.php?"+q.Encode(), &r); err != nil || len(r) < 2 {
		return nil
	}
	var uit []string
	if lijst, ok := r[1].([]any); ok {
		for _, x := range lijst {
			if s, ok := x.(string); ok && !strings.EqualFold(s, woord) {
				uit = append(uit, s)
			}
		}
	}
	return uit
}

/* ---------------- Nederlands: wikitekst van nl.wiktionary ---------------- */

var (
	nlSoorten = map[string]string{"noun": "zelfstandig naamwoord", "verb": "werkwoord", "adjc": "bijvoeglijk naamwoord",
		"adverb": "bijwoord", "prep": "voorzetsel", "pronoun": "voornaamwoord", "num": "telwoord", "conj": "voegwoord",
		"interj": "tussenwerpsel", "name": "naam", "art": "lidwoord", "phrase": "uitdrukking", "verb-form": "werkwoordsvorm",
		"noun-form": "vorm van een zelfstandig naamwoord", "adjc-form": "vorm van een bijvoeglijk naamwoord"}
	// labels waarbij de betekenis niet voor kinderen bedoeld is
	nlWeg      = regexp.MustCompile(`(?i)\{\{(vulg|vulgair|plat|scheldwoord|seksualiteit|seks|schuttingtaal|pejoratief|grof|erotiek)\b`)
	sjabloon   = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	koppelingT = regexp.MustCompile(`\[\[(?:[^\]|]*\|)?([^\]]*)\]\]`)
	htmlTag    = regexp.MustCompile(`<ref[^>]*/>|<ref[^>]*>.*?</ref>|<[^>]+>`)
	soortKop   = regexp.MustCompile(`^\{\{-([a-z-]+)-\|nld\}\}`)
)

func zoekNL(basis string, u *woordUitleg) error {
	q := url.Values{"action": {"parse"}, "page": {u.Woord}, "prop": {"wikitext"}, "format": {"json"}, "formatversion": {"2"}, "redirects": {"1"}}
	var r struct {
		Parse struct {
			Title    string `json:"title"`
			Wikitext string `json:"wikitext"`
		} `json:"parse"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if _, err := wiktVraag(basis+"/w/api.php?"+q.Encode(), &r); err != nil {
		return err
	}
	if r.Error != nil {
		return nil // woord bestaat niet: suggesties volgen
	}
	if r.Parse.Title != "" {
		u.Woord = r.Parse.Title
	}
	leesNLWikitekst(r.Parse.Wikitext, u)
	return nil
}

func schoonWikitekst(s string) string {
	s = htmlTag.ReplaceAllString(s, "")
	for i := 0; i < 4 && strings.Contains(s, "{{"); i++ {
		s = sjabloon.ReplaceAllString(s, "")
	}
	s = koppelingT.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("'''", "", "''", "", "[[", "", "]]", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func leesNLWikitekst(tekst string, u *woordUitleg) {
	// alleen het Nederlandse deel (vanaf {{=nld=}} tot de volgende taal)
	if i := strings.Index(tekst, "{{=nld=}}"); i >= 0 {
		tekst = tekst[i+len("{{=nld=}}"):]
		if j := strings.Index(tekst, "{{="); j >= 0 {
			tekst = tekst[:j]
		}
	}
	inSyn := false
	gezien := map[string]bool{}
	for _, regel := range strings.Split(tekst, "\n") {
		regel = strings.TrimSpace(regel)
		switch {
		case soortKop.MatchString(regel):
			inSyn = false
			if naam, ok := nlSoorten[soortKop.FindStringSubmatch(regel)[1]]; ok && !gezien[naam] {
				gezien[naam] = true
				u.Soorten = append(u.Soorten, naam)
			}
		case strings.HasPrefix(regel, "{{-syn-}}"):
			inSyn = true
		case strings.HasPrefix(regel, "{{-"):
			inSyn = false
		case inSyn && strings.HasPrefix(regel, "*"):
			for _, w := range strings.Split(schoonWikitekst(strings.TrimLeft(regel, "*: ")), ",") {
				if w = strings.TrimSpace(w); w != "" && len(u.Synoniemen) < 8 {
					u.Synoniemen = append(u.Synoniemen, w)
				}
			}
		case strings.HasPrefix(regel, "#") && !strings.HasPrefix(regel, "#:") && !strings.HasPrefix(regel, "#*"):
			if nlWeg.MatchString(regel) {
				continue
			}
			if b := schoonWikitekst(strings.TrimLeft(regel, "# ")); b != "" && len(u.Betekenissen) < 8 {
				u.Betekenissen = append(u.Betekenissen, b)
			}
		}
	}
}

/* ---------------- Engels: definities van en.wiktionary ---------------- */

var enWeg = regexp.MustCompile(`(?i)^\s*\((?:[^)]*\b)?(vulgar|offensive|derogatory|slur|sexual|sex|obscene|ethnic slur)`)

func zoekEN(basis string, u *woordUitleg) error {
	var r map[string][]struct {
		PartOfSpeech string `json:"partOfSpeech"`
		Definitions  []struct {
			Definition string `json:"definition"`
		} `json:"definitions"`
	}
	code, err := wiktVraag(basis+"/api/rest_v1/page/definition/"+url.PathEscape(u.Woord), &r)
	if code == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	gezien := map[string]bool{}
	for _, s := range r["en"] {
		if !gezien[s.PartOfSpeech] {
			gezien[s.PartOfSpeech] = true
			u.Soorten = append(u.Soorten, strings.ToLower(s.PartOfSpeech))
		}
		for _, d := range s.Definitions {
			b := strings.Join(strings.Fields(htmlTag.ReplaceAllString(d.Definition, "")), " ")
			if b != "" && !enWeg.MatchString(b) && len(u.Betekenissen) < 8 {
				u.Betekenissen = append(u.Betekenissen, b)
			}
		}
	}
	return nil
}

/* ---------------- scherm ---------------- */

type woordStaat struct {
	invoer textinput.Model
	uitleg *woordUitleg
	fout   string
	laden  bool
	eerder []string
}

type woordMsg struct {
	u   woordUitleg
	err error
}

func nieuwWoordenboek() *woordStaat {
	w := &woordStaat{}
	w.invoer = textinput.New()
	w.invoer.CharLimit = 40
	w.invoer.Prompt = "🔍 "
	return w
}

func (m *model) toetsWoord(msg tea.KeyMsg) (tea.Cmd, bool) {
	w := m.sh.wb
	switch msg.String() {
	case "esc":
		w.invoer.Blur()
		m.sh.scherm = schrijfMenu
	case "enter":
		woord := strings.TrimSpace(w.invoer.Value())
		if woord == "" || w.laden {
			return nil, true
		}
		w.laden, w.fout, w.uitleg = true, "", nil
		taal := m.inst.Taal
		return func() tea.Msg { u, err := zoekWoord(taal, woord); return woordMsg{u, err} }, true
	default:
		var cmd tea.Cmd
		w.invoer, cmd = w.invoer.Update(msg)
		return cmd, true
	}
	return nil, true
}

func (m *model) woordBericht(msg woordMsg) {
	w := m.sh.wb
	w.laden = false
	if msg.err != nil {
		w.fout = m.t("dict.offline")
		return
	}
	w.uitleg = &msg.u
	if len(msg.u.Betekenissen) > 0 {
		w.eerder = append([]string{msg.u.Woord}, w.eerder...)
		if len(w.eerder) > 8 {
			w.eerder = w.eerder[:8]
		}
	}
}

func (m model) viewWoord(p palet, breed int) string {
	w := m.sh.wb
	dim := lipgloss.NewStyle().Foreground(p.dim)
	regels := []string{w.invoer.View(), ""}
	switch {
	case w.laden:
		regels = append(regels, dim.Render(m.t("listen.loading")))
	case w.fout != "":
		regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render(w.fout))
	case w.uitleg != nil && len(w.uitleg.Betekenissen) > 0:
		u := w.uitleg
		regels = append(regels, lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(u.Woord)+dim.Render("  "+strings.Join(u.Soorten, ", ")), "")
		for i, b := range u.Betekenissen {
			regels = append(regels, lipgloss.NewStyle().Width(breed-6).Render(fmt.Sprintf("%d. %s", i+1, b)))
		}
		if len(u.Synoniemen) > 0 {
			regels = append(regels, "", dim.Render(m.t("dict.synonyms"))+" "+strings.Join(u.Synoniemen, ", "))
		}
		regels = append(regels, "", dim.Render(m.t("dict.source")))
	case w.uitleg != nil:
		regels = append(regels, m.t("dict.notfound", w.uitleg.Woord))
		if len(w.uitleg.Suggesties) > 0 {
			regels = append(regels, lipgloss.NewStyle().Foreground(p.groen).Render(m.t("dict.didyoumean", strings.Join(w.uitleg.Suggesties, ", "))))
		}
	default:
		regels = append(regels, dim.Render(m.t("dict.hint")))
		if len(w.eerder) > 0 {
			regels = append(regels, "", dim.Render(m.t("dict.earlier")+" "+strings.Join(w.eerder, ", ")))
		}
	}
	return strings.Join(regels, "\n")
}
