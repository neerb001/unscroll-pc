package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Het tabblad Luisteren: muziek (Navidrome), luisterboeken en podcasts (Audiobookshelf) en radio.
// Er is geen autoplay: een album speelt tot het einde en dan is het stil.

const (
	soortArtiest = iota + 1
	soortAlbum
	soortNummer
	soortBoek
	soortPodcast
	soortAflevering
	soortZender
	soortSpLijst  // goedgekeurd Spotify-album of -playlist
	soortSpNummer // nummer op Spotify
	soortSpArtiest
	soortSpZoek // "zoeken op Spotify"
)

const (
	bronMuziek = iota
	bronBoeken
	bronPodcasts
	bronRadio
	bronSpotify
	aantalBronnen
)

var bronSleutels = []string{"listen.music", "listen.books", "listen.podcasts", "listen.radio", "listen.spotify"}

var slaapStappen = []int{0, 15, 30, 45, 60}

type item struct {
	id, ouder, titel, sub, info, url string
	soort, aantal                    int
	duur, voortgang                  float64
	klaar                            bool
}

type lijst struct {
	titel, sleutel string
	items          []item
	sel            int
	filter         string
	laden          bool
	fout           string
}

// zichtbaar geeft de plaatsen van de items die bij het zoekwoord passen.
func (l *lijst) zichtbaar() []int {
	var uit []int
	f := strings.ToLower(l.filter)
	for i, it := range l.items {
		if f == "" || strings.Contains(strings.ToLower(it.titel+" "+it.sub), f) {
			uit = append(uit, i)
		}
	}
	return uit
}

func (l *lijst) gekozen() (item, bool) {
	z := l.zichtbaar()
	if l.sel < 0 || l.sel >= len(z) {
		return item{}, false
	}
	return l.items[z[l.sel]], true
}

type afspelen struct {
	soort       int
	titel, sub  string
	items       []item     // muziek: de nummers die geladen zijn
	sessie      string     // Audiobookshelf
	sporen      []absSpoor // Audiobookshelf: geladen sporen met hun plek in het boek
	duur        float64
	stand       SpelerStand
	gestart     bool
	startGezet  bool
	sinds       time.Time
	vorigePoll  time.Time
	geluisterd  float64
	laatsteSync time.Time
}

// positie is de plek in het hele boek (of nummer), in seconden.
func (a *afspelen) positie() float64 {
	if len(a.sporen) > 0 && a.stand.Index >= 0 && a.stand.Index < len(a.sporen) {
		return a.sporen[a.stand.Index].StartOffset + a.stand.Pos
	}
	return a.stand.Pos
}

// huidig is het item dat nu speelt (bij een album: het nummer waar mpv is).
func (a *afspelen) huidig() item {
	if a.stand.Index >= 0 && a.stand.Index < len(a.items) {
		return a.items[a.stand.Index]
	}
	if len(a.items) > 0 {
		return a.items[0]
	}
	return item{}
}

func (a *afspelen) totaal() float64 {
	if a.duur > 0 {
		return a.duur
	}
	return a.stand.Duur
}

type luister struct {
	bron     int
	stapels  [aantalBronnen][]lijst
	zoek     textinput.Model
	zoeken   bool
	spZoeken bool // typt een zoekvraag voor Spotify
	nav      *Navidrome
	abs      *ABS
	sp       *Speler
	spotify  *Spotify
	lspot    *Librespot
	nu       *afspelen
	pollID   int
	slaapTot time.Time
	slaapMin int
}

func nieuwLuister(g Geheimen) *luister {
	apparaat, _ := os.Hostname()
	l := &luister{nav: nieuweNavidrome(g), abs: nieuweABS(g, "unscroll-"+apparaat), sp: nieuweSpeler(),
		spotify: nieuweSpotify(g), lspot: &Librespot{naam: spotifyApparaatNaam()}}
	l.zoek = textinput.New()
	l.zoek.Prompt = "/ "
	l.zoek.CharLimit = 30
	return l
}

/* ---------------- berichten ---------------- */

type lijstMsg struct {
	bron    int
	sleutel string
	items   []item
	err     error
}

type startMsg struct {
	a   *afspelen
	err error
}

type spelerMsg struct {
	id    int
	stand SpelerStand
	err   error
}

func laadCmd(bron int, sleutel string, f func() ([]item, error)) tea.Cmd {
	return func() tea.Msg {
		items, err := f()
		return lijstMsg{bron, sleutel, items, err}
	}
}

// poll vraagt over een tel (Spotify: twee tellen) hoe het afspelen ervoor staat.
func (m *model) poll(a *afspelen) tea.Cmd {
	l := m.l
	if a.soort == soortSpNummer {
		sp, id, lijst := l.spotify, l.pollID, a.items
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			st, err := sp.Stand(lijst)
			return spelerMsg{id, st, err}
		})
	}
	return pollCmd(l.sp, l.pollID)
}

func pollCmd(sp *Speler, id int) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		st, err := sp.Stand()
		return spelerMsg{id, st, err}
	})
}

/* ---------------- lijsten ---------------- */

func (m *model) stapel() *[]lijst { return &m.l.stapels[m.l.bron] }

func (m *model) huidigeLijst() *lijst {
	s := m.stapel()
	if len(*s) == 0 {
		return nil
	}
	return &(*s)[len(*s)-1]
}

// luisterOpen zorgt dat de gekozen bron een lijst heeft (bij de eerste keer openen).
func (m *model) luisterOpen() tea.Cmd {
	if m.dicht() {
		m.l.bron = bronBoeken
	}
	if len(*m.stapel()) > 0 {
		return nil
	}
	return m.laadWortel()
}

func (m *model) laadWortel() tea.Cmd {
	l := m.l
	titel := m.t(bronSleutels[l.bron])
	nieuw := func(sleutel string, f func() ([]item, error)) tea.Cmd {
		l.stapels[l.bron] = []lijst{{titel: titel, sleutel: sleutel, laden: true}}
		return laadCmd(l.bron, sleutel, f)
	}
	nietIngesteld := func() tea.Cmd {
		l.stapels[l.bron] = []lijst{{titel: titel, fout: m.t("listen.notset")}}
		return nil
	}
	switch l.bron {
	case bronMuziek:
		if l.nav == nil {
			return nietIngesteld()
		}
		return nieuw("artiesten", l.nav.Artiesten)
	case bronBoeken:
		if l.abs == nil {
			return nietIngesteld()
		}
		return nieuw("boeken", func() ([]item, error) {
			alle, err := l.abs.Items("book")
			// alleen boeken met geluid: e-boeken (ook uit een aparte e-boekenbibliotheek) staan bij Lezen
			var boeken []item
			for _, it := range alle {
				if it.duur > 0 {
					boeken = append(boeken, it)
				}
			}
			return boeken, err
		})
	case bronPodcasts:
		if l.abs == nil {
			return nietIngesteld()
		}
		return nieuw("podcasts", func() ([]item, error) { return l.abs.Items("podcast") })
	case bronSpotify:
		if l.spotify == nil {
			return nietIngesteld()
		}
		zoeken := m.ouders.Spotify.Zoeken
		if len(m.ouders.Spotify.Goedgekeurd) == 0 && !zoeken {
			l.stapels[l.bron] = []lijst{{titel: titel, fout: m.t("listen.spempty")}}
			return nil
		}
		var vast []item
		if zoeken {
			vast = append(vast, item{titel: m.t("listen.spsearch"), soort: soortSpZoek})
			for _, f := range laadSpFavorieten() {
				vast = append(vast, favorietItem(f))
			}
		}
		links := m.ouders.Spotify.Goedgekeurd
		return nieuw("spotify", func() ([]item, error) {
			if len(links) == 0 {
				return vast, nil
			}
			goed, err := l.spotify.Goedgekeurd(links)
			if err != nil && len(vast) > 0 {
				return vast, nil
			}
			return append(vast, goed...), err
		})
	default:
		var items []item
		for _, z := range m.ouders.zenders() {
			items = append(items, item{titel: z.Naam, url: z.URL, soort: soortZender})
		}
		l.stapels[l.bron] = []lijst{{titel: titel, sleutel: "radio", items: items}}
		return nil
	}
}

// open opent het gekozen item: een map gaat een laag dieper, iets om te luisteren begint te spelen.
func (m *model) open() tea.Cmd {
	l := m.l
	lj := m.huidigeLijst()
	if lj == nil {
		return nil
	}
	it, ok := lj.gekozen()
	if !ok {
		return nil
	}
	dieper := func(sleutel string, f func() ([]item, error)) tea.Cmd {
		*m.stapel() = append(*m.stapel(), lijst{titel: it.titel, sleutel: sleutel, laden: true})
		return laadCmd(l.bron, sleutel, f)
	}
	switch it.soort {
	case soortArtiest:
		return dieper("artiest:"+it.id, func() ([]item, error) { return l.nav.Albums(it.id) })
	case soortAlbum:
		return dieper("album:"+it.id, func() ([]item, error) { return l.nav.Nummers(it.id) })
	case soortPodcast:
		return dieper("podcast:"+it.id, func() ([]item, error) { return l.abs.Afleveringen(it.id) })
	case soortNummer:
		z := lj.zichtbaar()
		var nummers []item
		for _, i := range z[lj.sel:] {
			nummers = append(nummers, lj.items[i])
		}
		return m.speel(&afspelen{soort: soortNummer, titel: it.titel, sub: it.sub, items: nummers})
	case soortZender:
		return m.speel(&afspelen{soort: soortZender, titel: it.titel, items: []item{it}})
	case soortSpLijst:
		expliciet := m.ouders.Spotify.ExplicietMag
		return dieper("sp:"+it.id, func() ([]item, error) {
			items, _, err := l.spotify.Nummers(it.id, expliciet)
			return items, err
		})
	case soortSpZoek:
		l.spZoeken = true
		l.zoek.SetValue("")
		l.zoek.Prompt = m.t("listen.spprompt")
		return l.zoek.Focus()
	case soortSpArtiest:
		b := nieuweBlokkeer(m.ouders.Spotify.Geblokkeerd)
		return dieper("spa:"+it.id, func() ([]item, error) {
			return l.spotify.AlbumsVan(strings.TrimPrefix(it.id, "artist:"), b)
		})
	case soortSpNummer:
		spLog("luister", lj.titel+" · "+it.titel)
		z := lj.zichtbaar()
		var nummers []item
		for _, i := range z[lj.sel:] {
			nummers = append(nummers, lj.items[i])
		}
		return m.speel(&afspelen{soort: soortSpNummer, titel: it.titel, sub: it.sub, items: nummers})
	case soortBoek:
		return m.speel(&afspelen{soort: soortBoek, titel: it.titel, sub: it.sub, items: []item{it}})
	case soortAflevering:
		return m.speel(&afspelen{soort: soortAflevering, titel: it.titel, sub: it.sub, items: []item{it}})
	}
	return nil
}

func (m *model) terug() bool {
	s := m.stapel()
	if len(*s) > 1 {
		*s = (*s)[:len(*s)-1]
		return true
	}
	return false
}

/* ---------------- afspelen ---------------- */

// speel stopt wat er speelt en begint aan iets nieuws.
func (m *model) speel(a *afspelen) tea.Cmd {
	l := m.l
	stop := m.stopHuidig(false)
	a.sinds = time.Now()
	if m.dicht() && a.soort == soortBoek && l.slaapTot.IsZero() {
		m.zetSlaap(30)
	}
	sp, abs, spot, lspot := l.sp, l.abs, l.spotify, l.lspot
	begin := func() tea.Msg {
		switch a.soort {
		case soortSpNummer:
			sp.LeegAlsActief()
			if err := lspot.Start(); err != nil {
				return startMsg{a, err}
			}
			var uris []string
			for _, it := range a.items {
				uris = append(uris, it.url)
			}
			return startMsg{a, spot.Speel(uris)}
		case soortNummer, soortZender:
			var urls []string
			for _, it := range a.items {
				urls = append(urls, it.url)
			}
			return startMsg{a, sp.Speel(urls, 0, "")}
		default: // luisterboek of podcastaflevering via Audiobookshelf
			it := a.items[0]
			id, afl := it.id, ""
			if a.soort == soortAflevering {
				id, afl = it.ouder, it.id
			}
			s, err := abs.Start(id, afl)
			if err != nil {
				return startMsg{a, err}
			}
			a.sessie, a.duur = s.ID, s.Duration
			if s.DisplayTitle != "" {
				a.titel = s.DisplayTitle
			}
			if s.DisplayAuthor != "" {
				a.sub = s.DisplayAuthor
			}
			if len(s.AudioTracks) == 0 {
				return startMsg{a, fmt.Errorf("no audio")}
			}
			eerste := 0
			for i, sp := range s.AudioTracks {
				if s.CurrentTime >= sp.StartOffset {
					eerste = i
				}
			}
			a.sporen = s.AudioTracks[eerste:]
			var urls []string
			for _, sp := range a.sporen {
				urls = append(urls, abs.SpoorURL(sp.ContentURL))
			}
			start := s.CurrentTime - a.sporen[0].StartOffset
			if start > 0 && a.duur-s.CurrentTime < 5 {
				start = 0 // het boek was al uit: opnieuw beginnen
				a.sporen = s.AudioTracks
				urls = urls[:0]
				for _, sp := range a.sporen {
					urls = append(urls, abs.SpoorURL(sp.ContentURL))
				}
			}
			a.startGezet = start > 0
			return startMsg{a, sp.Speel(urls, start, "Authorization: Bearer "+abs.Token)}
		}
	}
	l.nu = a
	l.pollID++
	return tea.Sequence(stop, begin)
}

// stopHuidig stopt wat er speelt en geeft de plek door aan Audiobookshelf.
func (m *model) stopHuidig(stilte bool) tea.Cmd {
	l := m.l
	a := l.nu
	l.nu = nil
	l.pollID++
	if a == nil {
		return nil
	}
	sp, abs, spot := l.sp, l.abs, l.spotify
	return func() tea.Msg {
		if a.soort == soortSpNummer {
			_ = spot.Pauze(true)
			return nil
		}
		if stilte {
			sp.Leeg()
		}
		if a.sessie != "" && abs != nil {
			_ = abs.Sluit(a.sessie, a.positie(), a.geluisterd, a.totaal())
		}
		return nil
	}
}

func (m *model) zetSlaap(min int) {
	m.l.slaapMin = min
	if min == 0 {
		m.l.slaapTot = time.Time{}
	} else {
		m.l.slaapTot = time.Now().Add(time.Duration(min) * time.Minute)
	}
}

func (m *model) luisterBericht(msg tea.Msg) tea.Cmd {
	l := m.l
	switch msg := msg.(type) {
	case lijstMsg:
		for i := range l.stapels[msg.bron] {
			lj := &l.stapels[msg.bron][i]
			if lj.sleutel != msg.sleutel {
				continue
			}
			lj.laden = false
			lj.items = msg.items
			lj.fout = ""
			if msg.err != nil {
				lj.fout = m.t("listen.err", msg.err.Error())
				if errors.Is(msg.err, errSpotifyDruk) {
					lj.fout = m.t("listen.spbusy", wachtUit(msg.err))
				}
			} else if len(msg.items) == 0 {
				lj.fout = m.t("listen.empty")
			}
			if lj.sel >= len(lj.zichtbaar()) {
				lj.sel = 0
			}
		}
	case startMsg:
		if msg.a != l.nu {
			return nil // intussen al iets anders gekozen
		}
		if msg.err != nil {
			l.nu = nil
			switch {
			case msg.err == errGeenMpv:
				m.zeg(m.t("listen.nompv"))
			case errors.Is(msg.err, errGeenLibrespot):
				m.zeg(m.t("listen.nolibrespot"))
			case errors.Is(msg.err, errNietGekoppeld):
				m.zeg(m.t("listen.splink"))
			case errors.Is(msg.err, errGeenApparaat):
				m.zeg(m.t("listen.spnodevice"))
			case errors.Is(msg.err, errSpotifyDruk):
				m.zeg(m.t("listen.spbusy", wachtUit(msg.err)))
			case errors.Is(msg.err, errSpotifyIngelogd):
				m.zeg(m.t("listen.splogin"))
			default:
				m.zeg(m.t("listen.err", msg.err.Error()))
			}
			return nil
		}
		msg.a.laatsteSync = time.Now()
		return m.poll(msg.a)
	case spelerMsg:
		a := l.nu
		if msg.id != l.pollID || a == nil {
			return nil
		}
		if msg.err != nil {
			m.zeg(m.t("listen.err", msg.err.Error()))
			return m.stopHuidig(false)
		}
		nu := time.Now()
		a.stand = msg.stand
		var cmds []tea.Cmd
		if !msg.stand.Leeg {
			if !a.gestart {
				a.gestart = true
			} else if !msg.stand.Pauze && !a.vorigePoll.IsZero() {
				a.geluisterd += min(nu.Sub(a.vorigePoll).Seconds(), 5)
			}
			if a.startGezet && msg.stand.Pos > 0 {
				a.startGezet = false
				sp := l.sp
				cmds = append(cmds, func() tea.Msg { sp.StartVrij(); return nil })
			}
		}
		a.vorigePoll = nu
		if !l.slaapTot.IsZero() && nu.After(l.slaapTot) {
			m.zetSlaap(0)
			m.zeg(m.t("listen.slept"))
			if !msg.stand.Pauze {
				a.stand.Pauze = true
				cmds = append(cmds, m.pauzeCmd(true))
			}
		}
		if a.gestart && msg.stand.Leeg {
			m.zeg(m.t("listen.done"))
			if a.duur > 0 {
				a.stand.Index, a.stand.Pos = len(a.sporen)-1, a.sporen[len(a.sporen)-1].Duration
			}
			return tea.Batch(append(cmds, m.stopHuidig(false))...)
		}
		if !a.gestart && nu.Sub(a.sinds) > 20*time.Second {
			m.zeg(m.t("listen.failed"))
			return tea.Batch(append(cmds, m.stopHuidig(true))...)
		}
		if a.sessie != "" && nu.Sub(a.laatsteSync) > 30*time.Second && a.gestart {
			a.laatsteSync = nu
			cmds = append(cmds, m.bijwerkCmd(a))
		}
		return tea.Batch(append(cmds, m.poll(a))...)
	}
	return nil
}

// bijwerkCmd geeft de plek in het boek door aan Audiobookshelf.
func (m *model) bijwerkCmd(a *afspelen) tea.Cmd {
	abs := m.l.abs
	if a.sessie == "" || abs == nil {
		return nil
	}
	pos, geluisterd, duur, sessie := a.positie(), a.geluisterd, a.totaal(), a.sessie
	a.geluisterd = 0
	return func() tea.Msg { _ = abs.Bij(sessie, pos, geluisterd, duur); return nil }
}

func (m *model) pauzeCmd(aan bool) tea.Cmd {
	if a := m.l.nu; a != nil && a.soort == soortSpNummer {
		spot := m.l.spotify
		return func() tea.Msg { _ = spot.Pauze(aan); return nil }
	}
	sp := m.l.sp
	var bij tea.Cmd
	if aan && m.l.nu != nil {
		bij = m.bijwerkCmd(m.l.nu)
	}
	return tea.Batch(func() tea.Msg { sp.Pauze(aan); return nil }, bij)
}

// luisterSlot wordt elke tik aangeroepen: tijdens het avondslot alleen een luisterboek, met slaaptimer.
func (m *model) luisterSlot() tea.Cmd {
	a := m.l.nu
	if !m.dicht() || a == nil {
		return nil
	}
	if a.soort != soortBoek || !m.ouders.Avondslot.LuisterboekMag {
		m.zeg(m.t("listen.lockstop"))
		return m.stopHuidig(true)
	}
	if m.l.slaapTot.IsZero() {
		m.zetSlaap(30)
	}
	return nil
}

/* ---------------- toetsen ---------------- */

func (m *model) toetsLuisteren(msg tea.KeyMsg) (tea.Cmd, bool) {
	l := m.l
	k := msg.String()
	lj := m.huidigeLijst()

	if l.spZoeken {
		switch k {
		case "esc":
			l.spZoeken = false
			l.zoek.Blur()
			l.zoek.Prompt = "/ "
		case "enter":
			vraag := strings.TrimSpace(l.zoek.Value())
			l.spZoeken = false
			l.zoek.Blur()
			l.zoek.SetValue("")
			l.zoek.Prompt = "/ "
			if vraag == "" {
				return nil, true
			}
			spLog("zoek", vraag)
			b := nieuweBlokkeer(m.ouders.Spotify.Geblokkeerd)
			sleutel := "spz:" + vraag
			*m.stapel() = append(*m.stapel(), lijst{titel: "“" + vraag + "”", sleutel: sleutel, laden: true})
			return laadCmd(l.bron, sleutel, func() ([]item, error) { return l.spotify.Zoek(vraag, b) }), true
		default:
			var cmd tea.Cmd
			l.zoek, cmd = l.zoek.Update(msg)
			return cmd, true
		}
		return nil, true
	}

	if l.zoeken && lj != nil {
		switch k {
		case "esc":
			l.zoeken = false
			l.zoek.Blur()
			l.zoek.SetValue("")
			lj.filter, lj.sel = "", 0
		case "enter":
			l.zoeken = false
			l.zoek.Blur()
			return m.open(), true
		case "up":
			lj.sel = max(lj.sel-1, 0)
		case "down":
			lj.sel = min(lj.sel+1, max(len(lj.zichtbaar())-1, 0))
		default:
			var cmd tea.Cmd
			l.zoek, cmd = l.zoek.Update(msg)
			if lj.filter != l.zoek.Value() {
				lj.filter, lj.sel = l.zoek.Value(), 0
			}
			return cmd, true
		}
		return nil, true
	}

	switch k {
	case "tab", "shift+tab":
		if m.dicht() {
			m.zeg(m.t("listen.lockonly"))
			return nil, true
		}
		d := 1
		if k == "shift+tab" {
			d = aantalBronnen - 1
		}
		l.bron = (l.bron + d) % aantalBronnen
		return m.luisterOpen(), true
	case "up", "k":
		if lj != nil && lj.sel > 0 {
			lj.sel--
		}
	case "down", "j":
		if lj != nil && lj.sel < len(lj.zichtbaar())-1 {
			lj.sel++
		}
	case "pgup":
		if lj != nil {
			lj.sel = max(lj.sel-10, 0)
		}
	case "pgdown":
		if lj != nil {
			lj.sel = max(min(lj.sel+10, len(lj.zichtbaar())-1), 0)
		}
	case "left", "right":
		if l.bron == bronRadio && lj != nil {
			if k == "left" && lj.sel > 0 {
				lj.sel--
			} else if k == "right" && lj.sel < len(lj.items)-1 {
				lj.sel++
			}
			return nil, true
		}
		if l.nu != nil && l.nu.soort != soortZender {
			sec := 30.0
			if k == "left" {
				sec = -30
			}
			if l.nu.soort == soortSpNummer {
				spot, naar := l.spotify, l.nu.stand.Pos+sec
				return func() tea.Msg { spot.Spoel(naar); return nil }, true
			}
			sp := l.sp
			return func() tea.Msg { sp.Spoel(sec); return nil }, true
		}
	case "enter":
		return m.open(), true
	case "backspace":
		m.terug()
	case "esc":
		if lj != nil && lj.filter != "" {
			lj.filter, lj.sel = "", 0
			l.zoek.SetValue("")
			return nil, true
		}
		return nil, m.terug()
	case "/":
		if lj != nil && len(lj.items) > 0 {
			l.zoeken = true
			l.zoek.SetValue(lj.filter)
			return l.zoek.Focus(), true
		}
	case "f":
		if l.bron == bronSpotify && m.ouders.Spotify.Zoeken && lj != nil {
			if it, ok := lj.gekozen(); ok && (it.soort == soortSpArtiest || it.soort == soortSpLijst) {
				if wisselFavoriet(it) {
					m.zeg(m.t("listen.spfavon", strings.TrimPrefix(it.titel, "★ ")))
				} else {
					m.zeg(m.t("listen.spfavoff", strings.TrimPrefix(it.titel, "★ ")))
				}
				if len(*m.stapel()) == 1 {
					*m.stapel() = nil
					return m.laadWortel(), true
				}
			}
			return nil, true
		}
	case "w":
		if l.bron == bronSpotify && m.ouderModus {
			*m.stapel() = append(*m.stapel(), lijst{titel: m.t("buddy.week"), sleutel: "spweek",
				items: spWeek(m.t("listen.splogsearch"), m.t("listen.sploglisten"))})
			if len(m.huidigeLijst().items) == 0 {
				m.huidigeLijst().fout = m.t("buddy.weekempty")
			}
			return nil, true
		}
	case "r":
		if len(*m.stapel()) <= 1 {
			*m.stapel() = nil
			return m.laadWortel(), true
		}
	case " ":
		if l.nu != nil {
			l.nu.stand.Pauze = !l.nu.stand.Pauze
			return m.pauzeCmd(l.nu.stand.Pauze), true
		}
	case "x":
		if l.nu != nil {
			return m.stopHuidig(true), true
		}
	case "n", "p":
		if l.nu != nil && l.nu.soort == soortSpNummer {
			spot := l.spotify
			if k == "n" {
				return func() tea.Msg { spot.Volgende(); return nil }, true
			}
			return func() tea.Msg { spot.Vorige(); return nil }, true
		}
		if l.nu != nil && l.nu.soort == soortNummer {
			sp := l.sp
			if k == "n" {
				return func() tea.Msg { sp.Volgende(); return nil }, true
			}
			return func() tea.Msg { sp.Vorige(); return nil }, true
		}
	case "s":
		i := 0
		for j, s := range slaapStappen {
			if s == l.slaapMin {
				i = j
			}
		}
		m.zetSlaap(slaapStappen[(i+1)%len(slaapStappen)])
		if l.slaapMin == 0 {
			m.zeg(m.t("listen.sleepoff"))
		} else {
			m.zeg(m.t("listen.sleepon", l.slaapMin))
		}
	case "+", "=":
		m.vol = zetVolume(5, m.ouders.MaxVolume)
		m.zeg(m.t("listen.vol", m.vol))
	case "-":
		m.vol = zetVolume(-5, m.ouders.MaxVolume)
		m.zeg(m.t("listen.vol", m.vol))
	default:
		return nil, false
	}
	return nil, true
}

/* ---------------- weergave ---------------- */

func (m model) viewLuisteren(p palet) string {
	l := m.l
	dim := lipgloss.NewStyle().Foreground(p.dim)

	var bronnen []string
	for i, s := range bronSleutels {
		naam := " " + m.t(s) + " "
		switch {
		case i == l.bron:
			bronnen = append(bronnen, lipgloss.NewStyle().Background(p.acc).Foreground(p.bg).Bold(true).Render(naam))
		case m.dicht() && i != bronBoeken:
			bronnen = append(bronnen, lipgloss.NewStyle().Foreground(p.lijn).Render(naam))
		default:
			bronnen = append(bronnen, dim.Render(naam))
		}
	}
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("menu.listen")) + "   " + strings.Join(bronnen, " ") + dim.Render("   "+m.t("listen.tab"))

	rbreed := 36
	if m.breed < 100 {
		rbreed = 30
	}
	lbreed := m.breed - rbreed - 9
	hoogte := m.hoog - 9

	var pad []string
	for _, lj := range *m.stapelVan(l.bron) {
		pad = append(pad, lj.titel)
	}
	links := ""
	lj := m.huidigeLijstVan(l.bron)
	switch {
	case lj == nil || lj.laden:
		links += dim.Render(m.t("listen.loading"))
	case lj.fout != "":
		links += lipgloss.NewStyle().Foreground(p.geel).Width(lbreed - 4).Render(lj.fout)
	default:
		if l.bron == bronRadio {
			links += m.viewDraaiknop(p, lj, lbreed-4) + "\n"
			hoogte -= 3
		}
		if l.zoeken || l.spZoeken || lj.filter != "" {
			links += l.zoek.View() + "\n"
			hoogte--
		}
		links += m.viewRegels(p, lj, lbreed-4, max(hoogte, 3))
	}
	links = strings.TrimSuffix(links, "\n")
	lPaneel := paneel(p, strings.Join(pad, " › "), links, lbreed, true)
	rPaneel := paneel(p, m.t("start.now"), m.viewNu(p, rbreed-4), rbreed, false)
	spKeys := ""
	if l.bron == bronSpotify && m.ouders.Spotify.Zoeken {
		spKeys = "\n" + dim.Render(m.t("listen.spkeys"))
	}
	return kop + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, lPaneel, " ", rPaneel) + "\n" + dim.Render(m.t("listen.keys")) + spKeys
}

func (m model) stapelVan(bron int) *[]lijst { return &m.l.stapels[bron] }

func (m model) huidigeLijstVan(bron int) *lijst {
	s := m.stapelVan(bron)
	if len(*s) == 0 {
		return nil
	}
	return &(*s)[len(*s)-1]
}

func (m model) viewRegels(p palet, lj *lijst, breed, hoogte int) string {
	z := lj.zichtbaar()
	if len(z) == 0 {
		return lipgloss.NewStyle().Foreground(p.dim).Render(m.t("listen.nomatch"))
	}
	begin := 0
	if lj.sel >= hoogte {
		begin = lj.sel - hoogte + 1
	}
	var regels []string
	for r := begin; r < len(z) && r < begin+hoogte; r++ {
		it := lj.items[z[r]]
		info := it.info
		switch {
		case it.klaar:
			info = "✓ " + info
		case it.voortgang > 0.01:
			info = fmt.Sprintf("%d%% · %s", int(it.voortgang*100), info)
		}
		if it.soort == soortArtiest && it.aantal > 0 {
			info = fmt.Sprint(it.aantal)
		}
		if m.l.nu != nil && m.l.nu.huidig().id == it.id && m.l.nu.huidig().url == it.url {
			info = "♪ " + info
		}
		titel := it.titel
		ruimte := breed - lipgloss.Width(info) - 4
		if lipgloss.Width(titel) > ruimte {
			titel = string([]rune(titel)[:max(ruimte-1, 1)]) + "…"
		}
		regel := " " + titel + strings.Repeat(" ", max(breed-lipgloss.Width(titel)-lipgloss.Width(info)-3, 1)) + info + " "
		if r == lj.sel {
			regels = append(regels, lipgloss.NewStyle().Background(p.sel).Foreground(p.acc).Bold(true).Render("▸"+regel))
		} else {
			regels = append(regels, " "+lipgloss.NewStyle().Foreground(p.fg).Render(regel))
		}
	}
	return strings.Join(regels, "\n")
}

// viewDraaiknop tekent een radioschaal met een naald bij de gekozen zender.
func (m model) viewDraaiknop(p palet, lj *lijst, breed int) string {
	n := len(lj.items)
	if n == 0 {
		return ""
	}
	breed = min(breed, 50)
	plek := func(i int) int {
		if n == 1 {
			return breed / 2
		}
		return 1 + i*(breed-3)/(n-1)
	}
	schaal := []rune(strings.Repeat("─", breed))
	for i := 0; i < n; i++ {
		schaal[plek(i)] = '┼'
	}
	naald := strings.Repeat(" ", plek(lj.sel)) + "▲"
	return lipgloss.NewStyle().Foreground(p.dim).Render(string(schaal)) + "\n" +
		lipgloss.NewStyle().Foreground(p.acc).Bold(true).Render(naald)
}

func (m model) viewNu(p palet, breed int) string {
	l := m.l
	dim := lipgloss.NewStyle().Foreground(p.dim)
	var regels []string
	a := l.nu
	if a == nil {
		regels = append(regels, dim.Render(m.t("listen.nothing")))
	} else {
		regels = append(regels, lipgloss.NewStyle().Bold(true).Width(breed).Render(a.titel))
		sub := a.sub
		if a.soort == soortNummer || a.soort == soortSpNummer {
			it := a.huidig()
			regels[0] = lipgloss.NewStyle().Bold(true).Width(breed).Render(it.titel)
			sub = it.sub
		}
		if a.soort == soortZender && a.stand.Titel != "" {
			sub = a.stand.Titel
		}
		if sub != "" {
			regels = append(regels, dim.Width(breed).Render(sub))
		}
		regels = append(regels, "")
		if tot := a.totaal(); tot > 0 && a.soort != soortZender {
			pos := a.positie()
			balk := breed - 2
			vol := int(float64(balk) * min(pos/tot, 1))
			regels = append(regels, lipgloss.NewStyle().Foreground(p.acc).Render(strings.Repeat("█", vol))+dim.Render(strings.Repeat("░", balk-vol)),
				dim.Render(klok(pos)+" / "+klok(tot)))
		}
		switch {
		case !a.gestart:
			regels = append(regels, dim.Render(m.t("listen.starting")))
		case a.stand.Pauze:
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render("⏸ "+m.t("listen.paused")))
		default:
			regels = append(regels, lipgloss.NewStyle().Foreground(p.groen).Render("▶ "+m.t("listen.playing")))
		}
	}
	regels = append(regels, "")
	if l.slaapTot.IsZero() {
		regels = append(regels, dim.Render(m.t("listen.sleep")+" "+m.t("listen.sleepnone")))
	} else {
		over := int(time.Until(l.slaapTot).Minutes()) + 1
		regels = append(regels, m.t("listen.sleep")+" "+lipgloss.NewStyle().Foreground(p.acc).Render(m.t("listen.sleepleft", over)))
	}
	if m.vol >= 0 {
		regels = append(regels, dim.Render(m.t("sys.volume")+fmt.Sprintf(" %d%%", m.vol)+m.t("sys.maxvol", m.ouders.MaxVolume)))
	}
	if m.dicht() {
		regels = append(regels, "", lipgloss.NewStyle().Foreground(p.geel).Width(breed).Render(m.t("listen.lockonly")))
	}
	return strings.Join(regels, "\n")
}

// nuKort is de regel voor het startscherm.
func (m model) nuKort() string {
	a := m.l.nu
	if a == nil {
		return m.t("listen.nothing")
	}
	t := a.titel
	if a.soort == soortNummer || a.soort == soortSpNummer {
		t = a.huidig().titel
	}
	if a.stand.Pauze {
		return "⏸ " + t
	}
	return "♪ " + t
}

// wachtUit haalt de wachttijd uit een foutmelding van Spotify ("..., retry after 1:30").
func wachtUit(err error) string {
	_, w, _ := strings.Cut(err.Error(), "retry after ")
	return w
}
