package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Lezen: e-boeken (EPUB) uit Audiobookshelf, in een rustige leesweergave.
// Een boek wordt één keer gedownload en daarna ook zonder wifi gelezen. De plek in het boek
// staat op het apparaat; het percentage gaat naar Audiobookshelf, zodat het ook elders klopt.

const soortEboek = 100

/* ---------------- Audiobookshelf ---------------- */

func (a *ABS) Ebooks() ([]item, error) {
	alle, err := a.Items("book")
	if err != nil {
		return nil, err
	}
	vg := a.voortgang()
	var uit []item
	for _, it := range alle {
		if it.url != "epub" {
			continue
		}
		p := vg[it.id+"/"]
		uit = append(uit, item{id: it.id, titel: it.titel, sub: it.sub, voortgang: p.EbookProgress, soort: soortEboek})
	}
	return uit, nil
}

func (a *ABS) DownloadEboek(id string) ([]byte, error) {
	req, err := http.NewRequest("GET", a.URL+"/api/items/"+id+"/ebook", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.Token)
	c := http.Client{Timeout: 2 * time.Minute}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audiobookshelf: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 200<<20))
}

func (a *ABS) EboekVoortgang(id string, deel float64) error {
	return a.doe("PATCH", "/api/me/progress/"+id, map[string]any{"ebookProgress": deel, "isFinished": deel >= 0.995}, nil)
}

/* ---------------- EPUB ---------------- */

// alinea: tekst, en of het een kop is.
type alinea struct {
	tekst string
	kop   bool
}

// leesEpub haalt de tekst uit een EPUB in de leesvolgorde van het boek.
func leesEpub(data []byte) ([]alinea, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	bestanden := map[string]*zip.File{}
	for _, f := range zr.File {
		bestanden[f.Name] = f
	}
	lees := func(naam string) ([]byte, error) {
		f, ok := bestanden[naam]
		if !ok {
			return nil, fmt.Errorf("epub: %s missing", naam)
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(io.LimitReader(r, 20<<20))
	}
	var container struct {
		Rootfiles []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	c, err := lees("META-INF/container.xml")
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(c, &container); err != nil || len(container.Rootfiles) == 0 {
		return nil, errors.New("epub: no content found")
	}
	opfPad := container.Rootfiles[0].Path
	var opf struct {
		Items []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	o, err := lees(opfPad)
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(o, &opf); err != nil {
		return nil, err
	}
	hrefs := map[string]string{}
	for _, it := range opf.Items {
		hrefs[it.ID] = it.Href
	}
	basis := path.Dir(opfPad)
	var uit []alinea
	for _, s := range opf.Spine {
		href := strings.SplitN(hrefs[s.IDRef], "#", 2)[0]
		if href == "" {
			continue
		}
		pad := path.Clean(path.Join(basis, href))
		if basis == "." {
			pad = path.Clean(href)
		}
		x, err := lees(pad)
		if err != nil {
			continue
		}
		uit = append(uit, htmlTekst(x)...)
	}
	if len(uit) == 0 {
		return nil, errors.New("epub: no text found")
	}
	return uit, nil
}

var blokTags = map[string]bool{"p": true, "div": true, "br": true, "li": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "blockquote": true, "tr": true, "section": true, "hr": true}

// htmlTekst maakt van (X)HTML alinea's: tekst per blok, koppen gemarkeerd.
func htmlTekst(data []byte) []alinea {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	var uit []alinea
	var b strings.Builder
	kop, overslaan := false, 0
	klaar := func() {
		t := strings.Join(strings.Fields(b.String()), " ")
		if t != "" {
			uit = append(uit, alinea{t, kop})
		}
		b.Reset()
		kop = false
	}
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := strings.ToLower(t.Name.Local)
			switch {
			case n == "script" || n == "style" || n == "head":
				overslaan++
			case blokTags[n]:
				klaar()
				kop = len(n) == 2 && n[0] == 'h' && n[1] >= '1' && n[1] <= '6'
			}
		case xml.EndElement:
			n := strings.ToLower(t.Name.Local)
			switch {
			case n == "script" || n == "style" || n == "head":
				overslaan = max(overslaan-1, 0)
			case blokTags[n]:
				klaar()
			}
		case xml.CharData:
			if overslaan == 0 {
				b.Write(t)
				b.WriteByte(' ')
			}
		}
	}
	klaar()
	return uit
}

/* ---------------- staat ---------------- */

type leesStaat struct {
	lijst   lijst
	geladen bool
	boek    *openBoek
	zoeken  bool            // typt in het zoekveld
	zoek    textinput.Model // zoeken met /
}

type openBoek struct {
	id, titel string
	alineas   []alinea
	regels    []string // opgemaakt voor de huidige breedte
	breedte   int
	bovenste  int // eerste regel op het scherm
	gesynct   time.Time
}

type eboekenMsg struct {
	items []item
	err   error
}

type boekMsg struct {
	id, titel string
	alineas   []alinea
	err       error
}

func boekenMap() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "unscroll", "boeken")
}

// De plek per boek (welke alinea bovenaan stond) en de lijst voor als er geen wifi is.
type leesGeheugen struct {
	Plek   map[string]int `json:"plek"`
	Boeken []leesBoekInfo `json:"boeken"`
}

type leesBoekInfo struct{ ID, Titel, Sub string }

func leesGeheugenPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "lezen.json") }

func laadLeesGeheugen() leesGeheugen {
	g := leesGeheugen{Plek: map[string]int{}}
	if data, err := os.ReadFile(leesGeheugenPad()); err == nil {
		_ = json.Unmarshal(data, &g)
	}
	if g.Plek == nil {
		g.Plek = map[string]int{}
	}
	return g
}

func (g leesGeheugen) bewaar() {
	_ = os.MkdirAll(filepath.Dir(leesGeheugenPad()), 0o755)
	data, _ := json.Marshal(g)
	_ = veiligSchrijven(leesGeheugenPad(), data, 0o644)
}

/* ---------------- opdrachten ---------------- */

func (m *model) lezenOpen() tea.Cmd {
	ls := m.ls
	if ls.geladen || ls.lijst.laden {
		return nil
	}
	if m.l.abs == nil {
		ls.lijst = lijst{fout: m.t("read.notset")}
		ls.geladen = true
		return nil
	}
	ls.lijst = lijst{laden: true}
	abs := m.l.abs
	return func() tea.Msg {
		items, err := abs.Ebooks()
		if err != nil {
			// geen wifi: de boeken die al gedownload zijn
			for _, b := range laadLeesGeheugen().Boeken {
				if _, e := os.Stat(filepath.Join(boekenMap(), b.ID+".epub")); e == nil {
					items = append(items, item{id: b.ID, titel: b.Titel, sub: b.Sub, soort: soortEboek})
				}
			}
			if len(items) > 0 {
				err = nil
			}
		} else {
			g := laadLeesGeheugen()
			g.Boeken = nil
			for _, it := range items {
				g.Boeken = append(g.Boeken, leesBoekInfo{it.id, it.titel, it.sub})
			}
			g.bewaar()
		}
		return eboekenMsg{items, err}
	}
}

func (m *model) openEboek(it item) tea.Cmd {
	abs := m.l.abs
	return func() tea.Msg {
		pad := filepath.Join(boekenMap(), it.id+".epub")
		data, err := os.ReadFile(pad)
		if err != nil {
			if abs == nil {
				return boekMsg{err: err}
			}
			if data, err = abs.DownloadEboek(it.id); err != nil {
				return boekMsg{err: err}
			}
			_ = os.MkdirAll(boekenMap(), 0o700)
			_ = veiligSchrijven(pad, data, 0o600)
		}
		alineas, err := leesEpub(data)
		return boekMsg{id: it.id, titel: it.titel, alineas: alineas, err: err}
	}
}

func (m *model) lezenBericht(msg tea.Msg) tea.Cmd {
	ls := m.ls
	switch msg := msg.(type) {
	case eboekenMsg:
		ls.geladen = true
		ls.lijst = lijst{items: msg.items}
		switch {
		case msg.err != nil:
			ls.lijst.fout = m.t("listen.err", msg.err.Error())
		case len(msg.items) == 0:
			ls.lijst.fout = m.t("read.empty")
		}
	case boekMsg:
		ls.lijst.laden = false
		if msg.err != nil {
			m.zeg(m.t("listen.err", msg.err.Error()))
			return nil
		}
		b := &openBoek{id: msg.id, titel: msg.titel, alineas: msg.alineas}
		b.opmaak(m.leesBreedte())
		plek := laadLeesGeheugen().Plek[msg.id]
		b.bovenste = b.regelVanAlinea(plek)
		ls.boek = b
	}
	return nil
}

func (m *model) leesBreedte() int { return max(min(m.breed-10, 72), 30) }

func (m *model) leesHoogte() int { return max(m.hoog-8, 5) }

// opmaak breekt de alinea's af op de breedte van het scherm; alineaBegin houdt bij waar elke alinea begint.
func (b *openBoek) opmaak(breedte int) {
	b.breedte = breedte
	b.regels = b.regels[:0]
	for _, a := range b.alineas {
		tekst := lipgloss.NewStyle().Width(breedte).Render(a.tekst)
		if a.kop {
			tekst = "\x00" + tekst // markering voor vet
		}
		b.regels = append(b.regels, strings.Split(tekst, "\n")...)
		b.regels = append(b.regels, "")
	}
}

func (b *openBoek) regelVanAlinea(n int) int {
	r := 0
	for i := 0; i < n && i < len(b.alineas); i++ {
		r += strings.Count(lipgloss.NewStyle().Width(b.breedte).Render(b.alineas[i].tekst), "\n") + 2
	}
	return min(r, max(len(b.regels)-1, 0))
}

func (b *openBoek) alineaVanRegel(regel int) int {
	r := 0
	for i := range b.alineas {
		r += strings.Count(lipgloss.NewStyle().Width(b.breedte).Render(b.alineas[i].tekst), "\n") + 2
		if r > regel {
			return i
		}
	}
	return len(b.alineas) - 1
}

func (b *openBoek) deel() float64 {
	if len(b.regels) == 0 {
		return 0
	}
	return min(float64(b.bovenste)/float64(max(len(b.regels)-1, 1)), 1)
}

// bewaarPlek onthoudt de plek en stuurt af en toe het percentage naar Audiobookshelf.
func (m *model) bewaarPlek(forceer bool) tea.Cmd {
	b := m.ls.boek
	if b == nil {
		return nil
	}
	g := laadLeesGeheugen()
	g.Plek[b.id] = b.alineaVanRegel(b.bovenste)
	g.bewaar()
	abs := m.l.abs
	if abs == nil || (!forceer && time.Since(b.gesynct) < time.Minute) {
		return nil
	}
	b.gesynct = time.Now()
	id, deel := b.id, b.deel()
	return func() tea.Msg { _ = abs.EboekVoortgang(id, deel); return nil }
}

func (m *model) toetsLezen(msg tea.KeyMsg) (tea.Cmd, bool) {
	ls := m.ls
	k := msg.String()
	if b := ls.boek; b != nil {
		pagina := m.leesHoogte() - 1
		switch k {
		case "right", "pgdown", " ", "l":
			b.bovenste = min(b.bovenste+pagina, max(len(b.regels)-pagina, 0))
		case "left", "pgup", "b", "h":
			b.bovenste = max(b.bovenste-pagina, 0)
		case "down", "j":
			b.bovenste = min(b.bovenste+1, max(len(b.regels)-1, 0))
		case "up", "k":
			b.bovenste = max(b.bovenste-1, 0)
		case "home":
			b.bovenste = 0
		case "esc", "q":
			cmd := m.bewaarPlek(true)
			ls.boek = nil
			ls.geladen = false // percentages verversen
			return tea.Batch(cmd, m.lezenOpen()), true
		default:
			return nil, false
		}
		return m.bewaarPlek(false), true
	}
	lj := &ls.lijst
	if ls.zoeken {
		switch k {
		case "esc":
			m.stopLeesZoeken()
		case "enter":
			it, ok := lj.gekozen()
			m.stopLeesZoeken()
			if ok {
				lj.laden = true
				return m.openEboek(it), true
			}
		case "up":
			lj.sel = max(lj.sel-1, 0)
		case "down":
			lj.sel = min(lj.sel+1, max(len(lj.zichtbaar())-1, 0))
		default:
			var cmd tea.Cmd
			ls.zoek, cmd = ls.zoek.Update(msg)
			if lj.filter != ls.zoek.Value() {
				lj.filter, lj.sel = ls.zoek.Value(), 0
			}
			return cmd, true
		}
		return nil, true
	}
	switch k {
	case "/":
		if lj.laden || lj.fout != "" || len(lj.items) == 0 {
			return nil, true
		}
		ls.zoek = textinput.New()
		ls.zoek.Prompt = "/ "
		ls.zoek.Placeholder = m.t("read.searchhint")
		ls.zoek.CharLimit = 40
		ls.zoek.Cursor.SetMode(cursor.CursorStatic) // rustig: geen knipperende cursor
		ls.zoeken = true
		return ls.zoek.Focus(), true
	case "up", "k":
		lj.sel = max(lj.sel-1, 0)
	case "down", "j":
		lj.sel = min(lj.sel+1, max(len(lj.zichtbaar())-1, 0))
	case "enter":
		if it, ok := lj.gekozen(); ok {
			lj.laden = true
			return m.openEboek(it), true
		}
	case "r":
		ls.geladen = false
		return m.lezenOpen(), true
	default:
		return nil, false
	}
	return nil, true
}

// stopLeesZoeken: zoekveld dicht en weer de hele lijst.
func (m *model) stopLeesZoeken() {
	ls := m.ls
	ls.zoeken = false
	ls.zoek.Blur()
	ls.lijst.filter, ls.lijst.sel = "", 0
}

/* ---------------- weergave ---------------- */

func (m model) viewLezen(p palet) string {
	ls := m.ls
	dim := lipgloss.NewStyle().Foreground(p.dim)
	if b := ls.boek; b != nil {
		if b.breedte != m.leesBreedte() {
			plek := b.alineaVanRegel(b.bovenste)
			b.opmaak(m.leesBreedte())
			b.bovenste = b.regelVanAlinea(plek)
		}
		h := m.leesHoogte()
		eind := min(b.bovenste+h, len(b.regels))
		var regels []string
		for _, r := range b.regels[b.bovenste:eind] {
			if strings.HasPrefix(r, "\x00") {
				r = lipgloss.NewStyle().Bold(true).Foreground(p.acc).Render(strings.TrimPrefix(r, "\x00"))
			}
			regels = append(regels, r)
		}
		pagina, totaal := b.bovenste/max(h-1, 1)+1, max((len(b.regels)+h-2)/max(h-1, 1), 1)
		kop := lipgloss.NewStyle().Bold(true).Render(b.titel)
		voet := dim.Render(m.t("read.page", pagina, totaal, int(b.deel()*100)) + "   " + m.t("read.keys"))
		tekst := lipgloss.NewStyle().Width(b.breedte).Render(strings.Join(regels, "\n"))
		midden := lipgloss.PlaceHorizontal(m.breed-4, lipgloss.Center, tekst)
		return lipgloss.PlaceHorizontal(m.breed-4, lipgloss.Center, kop) + "\n\n" + midden + "\n\n" + lipgloss.PlaceHorizontal(m.breed-4, lipgloss.Center, voet)
	}
	lj := &ls.lijst
	var inhoud string
	switch {
	case lj.laden:
		inhoud = dim.Render(m.t("listen.loading"))
	case lj.fout != "":
		inhoud = lipgloss.NewStyle().Foreground(p.geel).Width(m.breed - 12).Render(lj.fout)
	case ls.zoeken:
		aantal := dim.Render("  " + m.t("read.found", len(lj.zichtbaar()), len(lj.items)))
		inhoud = ls.zoek.View() + aantal + "\n\n" + m.viewRegels(p, lj, m.breed-14, max(m.hoog-12, 3))
	default:
		inhoud = m.viewRegels(p, lj, m.breed-14, max(m.hoog-10, 3))
	}
	toetsen := m.t("read.listkeys")
	if ls.zoeken {
		toetsen = m.t("read.searchkeys")
	}
	return lipgloss.NewStyle().Bold(true).Render(m.t("menu.read")) + "\n\n" +
		paneel(p, m.t("read.books"), inhoud, m.breed-8, true) + "\n" + dim.Render(toetsen)
}
