package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Diagnose voor ouders: per onderdeel of het werkt, met de precieze foutmelding.
// Alles wordt tegelijk getest, elk onderdeel met een eigen tijdslimiet, buiten de hoofdlus.
// Ook het verslag van de waakhond (als de app ooit vastzat) is hier te lezen.

const (
	diagGoed = iota
	diagFout
	diagUit // niet ingesteld: geen fout, alleen ter info
	diagLet // werkt, maar let op
)

type diagRegel struct {
	sleutel string
	status  int
	tekst   string
}

type diagStaat struct {
	open    bool
	bezig   bool
	regels  []diagRegel
	gedaan  time.Time
	verslag bool // het verslag van de waakhond tonen
	scroll  int
}

type diagMsg []diagRegel

// diagVolgorde: zo staan ze op het scherm.
var diagVolgorde = []string{"internet", "parents", "storage", "watchdog", "version", "weather", "iss", "spotify", "abs", "navidrome", "ai", "stockfish", "hedy"}

// korteFout: een foutmelding zonder eindeloze JSON, op één regel.
func korteFout(err error) string {
	s := strings.Join(strings.Fields(err.Error()), " ")
	if r := []rune(s); len(r) > 240 {
		s = string(r[:240]) + "…"
	}
	return s
}

func (m *model) diagnoseStart() tea.Cmd {
	d := m.diag
	if d.bezig {
		return nil
	}
	d.bezig = true
	taal, ouders, oudersFout := m.inst.Taal, m.ouders, m.oudersFout
	wifi, l, nieuw := m.wifi, m.l, m.nieuw
	t := func(k string, a ...any) string { return tr(taal, k, a...) }

	return func() tea.Msg {
		var (
			mu  sync.Mutex
			uit []diagRegel
			wg  sync.WaitGroup
		)
		zet := func(sleutel string, status int, tekst string) {
			mu.Lock()
			uit = append(uit, diagRegel{sleutel, status, tekst})
			mu.Unlock()
		}
		test := func(f func()) {
			wg.Add(1)
			go func() { defer wg.Done(); f() }()
		}

		test(func() { // internet: naam opzoeken en verbinden
			if _, err := net.LookupHost("github.com"); err != nil {
				zet("internet", diagFout, t("diag.wifi", wifi)+" · "+t("diag.dns")+": "+korteFout(err))
				return
			}
			c, err := net.DialTimeout("tcp", "github.com:443", 8*time.Second)
			if err != nil {
				zet("internet", diagFout, t("diag.wifi", wifi)+" · "+korteFout(err))
				return
			}
			c.Close()
			zet("internet", diagGoed, t("diag.wifi", wifi)+" · "+t("diag.online"))
		})
		test(func() { // ouders.yaml
			switch {
			case oudersFout != nil:
				zet("parents", diagFout, korteFout(oudersFout))
			case ouders.PinSHA256 == "":
				zet("parents", diagLet, t("diag.nopin"))
			default:
				zet("parents", diagGoed, t("diag.parentsok", bedtijd(ouders, time.Now()), ouders.Avondslot.Open))
			}
		})
		test(func() { // opslag: vrije ruimte en of veilig bewaren lukt
			dir := unscrollData("")
			_ = os.MkdirAll(dir, 0o700)
			proef := filepath.Join(dir, ".diagnose-proef")
			if err := veiligSchrijven(proef, []byte("ok"), 0o600); err != nil {
				zet("storage", diagFout, korteFout(err))
				return
			}
			os.Remove(proef)
			var st syscall.Statfs_t
			if err := syscall.Statfs(dir, &st); err == nil {
				vrij := int64(st.Bavail) * int64(st.Bsize) / (1 << 20)
				status := diagGoed
				if vrij < 200 {
					status = diagLet
				}
				zet("storage", status, t("diag.free", vrij))
				return
			}
			zet("storage", diagGoed, t("diag.writeok"))
		})
		test(func() { // waakhond
			if info, err := os.Stat(vastgelopenPad()); err == nil {
				zet("watchdog", diagLet, t("diag.frozen", info.ModTime().Format("2-1 15:04")))
			} else {
				zet("watchdog", diagGoed, t("diag.neverfrozen"))
			}
		})
		test(func() {
			if nieuw != "" {
				zet("version", diagLet, t("diag.versionready", versie, nieuw))
			} else {
				zet("version", diagGoed, versie)
			}
		})
		test(func() { // weer
			msg := haalWeer(ouders)()
			if w, ok := msg.(weerMsg); ok && w.err != nil {
				zet("weather", diagFout, korteFout(w.err))
				return
			}
			zet("weather", diagGoed, ouders.plaatsNaam())
		})
		test(func() { // ISS: rechtstreeks bij de bron, niet uit de cache
			c := http.Client{Timeout: 10 * time.Second}
			req, _ := http.NewRequest("GET", celestrakISS, nil)
			req.Header.Set("User-Agent", "Unscroll/"+versie)
			resp, err := c.Do(req)
			if err != nil {
				zet("iss", diagFout, korteFout(err))
				return
			}
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK || !strings.Contains(string(data), "\n1 25544") {
				zet("iss", diagFout, "CelesTrak: "+resp.Status)
				return
			}
			zet("iss", diagGoed, "CelesTrak")
		})
		test(func() { // Spotify: inloggen en de speaker van deze computer
			s := l.spotify
			if s == nil {
				zet("spotify", diagUit, t("diag.notset"))
				return
			}
			var ik struct {
				Naam    string `json:"display_name"`
				Product string `json:"product"`
			}
			if _, err := s.doeEens("GET", "/me", nil, &ik); err != nil {
				zet("spotify", diagFout, korteFout(err))
				return
			}
			tekst := t("diag.loggedin", ik.Naam, ik.Product)
			if _, err := exec.LookPath("librespot"); err != nil {
				zet("spotify", diagFout, tekst+" · "+t("diag.nolibrespot"))
				return
			}
			if _, err := os.Stat(filepath.Join(librespotCache(), "credentials.json")); err != nil {
				zet("spotify", diagFout, tekst+" · "+t("diag.notlinked"))
				return
			}
			zet("spotify", diagGoed, tekst+" · "+t("diag.speaker", s.apparaat))
		})
		test(func() { // Audiobookshelf
			a := l.abs
			if a == nil {
				zet("abs", diagUit, t("diag.notset"))
				return
			}
			b, err := a.Bibliotheken()
			if err != nil {
				zet("abs", diagFout, a.URL+" · "+korteFout(err))
				return
			}
			zet("abs", diagGoed, a.URL+" · "+t("diag.libraries", len(b)))
		})
		test(func() { // Navidrome
			n := l.nav
			if n == nil {
				zet("navidrome", diagUit, t("diag.notset"))
				return
			}
			var x struct{}
			if err := n.vraag("ping", nil, &x); err != nil {
				zet("navidrome", diagFout, n.URL+" · "+korteFout(err))
				return
			}
			zet("navidrome", diagGoed, n.URL)
		})
		test(func() { // AI-maatje: sleutel en beide modellen (kost geen tokens)
			ai := nieuwMaatjeAI(laadGeheimen(), ouders)
			switch {
			case !ouders.Maatje.Aan:
				zet("ai", diagUit, t("diag.aioff"))
				return
			case ai == nil:
				zet("ai", diagUit, t("diag.nokey"))
				return
			}
			ctx, klaar := context.WithTimeout(context.Background(), 15*time.Second)
			defer klaar()
			for _, model := range []string{ai.model, ai.keur} {
				if _, err := ai.client.Models.Get(ctx, model, anthropic.ModelGetParams{}); err != nil {
					zet("ai", diagFout, model+": "+korteFout(err))
					return
				}
			}
			zet("ai", diagGoed, ai.model+" · "+ai.keur)
		})
		test(func() { // Stockfish
			e, err := startMotor()
			if err != nil {
				zet("stockfish", diagFout, korteFout(err))
				return
			}
			e.stop()
			zet("stockfish", diagGoed, t("diag.works"))
		})
		test(func() { // Hedy
			if _, err := os.Stat(filepath.Join(hedyMap(), "inhoud.json")); err != nil {
				zet("hedy", diagFout, t("diag.nohedy"))
				return
			}
			if _, err := exec.LookPath("python3"); err != nil {
				zet("hedy", diagFout, t("diag.nopython"))
				return
			}
			zet("hedy", diagGoed, hedyMap())
		})

		klaar := make(chan struct{})
		go func() { wg.Wait(); close(klaar) }()
		select {
		case <-klaar:
		case <-time.After(30 * time.Second): // wat dan nog niet klaar is, hangt
		}
		mu.Lock()
		defer mu.Unlock()
		gezien := map[string]bool{}
		for _, r := range uit {
			gezien[r.sleutel] = true
		}
		for _, k := range diagVolgorde {
			if !gezien[k] {
				uit = append(uit, diagRegel{k, diagFout, t("diag.timeout")})
			}
		}
		return diagMsg(append([]diagRegel(nil), uit...))
	}
}

func (m *model) diagnoseBericht(msg diagMsg) {
	d := m.diag
	d.bezig = false
	d.gedaan = time.Now()
	plek := map[string]int{}
	for i, k := range diagVolgorde {
		plek[k] = i
	}
	regels := []diagRegel(msg)
	for i := 1; i < len(regels); i++ { // op volgorde zetten
		for j := i; j > 0 && plek[regels[j].sleutel] < plek[regels[j-1].sleutel]; j-- {
			regels[j], regels[j-1] = regels[j-1], regels[j]
		}
	}
	d.regels = regels
}

func (m *model) toetsDiagnose(k string) tea.Cmd {
	d := m.diag
	switch k {
	case "esc", "q":
		if d.verslag {
			d.verslag, d.scroll = false, 0
		} else {
			d.open = false
		}
	case "r":
		return m.diagnoseStart()
	case "w":
		d.verslag, d.scroll = !d.verslag, 0
	case "x":
		if d.verslag {
			os.Remove(vastgelopenPad())
			d.verslag, d.scroll = false, 0
			return m.diagnoseStart()
		}
	case "down", "j", " ", "pgdown":
		stap := 1
		if k == " " || k == "pgdown" {
			stap = 10
		}
		d.scroll += stap
	case "up", "k", "pgup":
		stap := 1
		if k == "pgup" {
			stap = 10
		}
		d.scroll = max(d.scroll-stap, 0)
	}
	return nil
}

func (m model) viewDiagnose(p palet) string {
	d := m.diag
	dim := lipgloss.NewStyle().Foreground(p.dim)
	breed := m.breed - 6
	hoogte := max(m.hoog-8, 5)
	kop := lipgloss.NewStyle().Bold(true).Render(m.t("diag.title")) + "  " + dim.Render(m.t("sys.parentmode"))

	var regels []string
	if d.verslag {
		data, err := os.ReadFile(vastgelopenPad())
		if err != nil {
			regels = []string{dim.Render(m.t("diag.noreport"))}
		} else {
			for _, r := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
				regels = append(regels, lipgloss.NewStyle().MaxWidth(breed-4).Render(r))
			}
		}
	} else {
		naamBreed := 15
		tekstBreed := breed - 4 - naamBreed - 3
		if d.bezig && len(d.regels) == 0 {
			regels = append(regels, dim.Render(m.t("diag.running")))
		}
		for _, r := range d.regels {
			teken, kleur := "✓", p.groen
			switch r.status {
			case diagFout:
				teken, kleur = "✗", p.rood
			case diagUit:
				teken, kleur = "–", p.dim
			case diagLet:
				teken, kleur = "!", p.geel
			}
			naam := lipgloss.NewStyle().Width(naamBreed).Render(m.t("diag." + r.sleutel))
			tekst := lipgloss.NewStyle().Width(tekstBreed).Render(r.tekst)
			if r.status == diagFout {
				tekst = lipgloss.NewStyle().Foreground(p.rood).Width(tekstBreed).Render(r.tekst)
			}
			blok := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Foreground(kleur).Bold(true).Render(teken+" "), naam, " ", tekst)
			regels = append(regels, strings.Split(blok, "\n")...)
		}
		if !d.gedaan.IsZero() {
			stand := m.t("diag.checked", d.gedaan.Format("15:04:05"))
			if d.bezig {
				stand = m.t("diag.running")
			}
			regels = append(regels, "", dim.Render(stand))
		}
	}
	// scrollen
	d.scroll = min(d.scroll, max(len(regels)-hoogte, 0))
	if len(regels) > hoogte {
		regels = regels[d.scroll:min(d.scroll+hoogte, len(regels))]
	}
	titel := m.t("diag.checks")
	toetsen := m.t("diag.keys")
	if d.verslag {
		titel, toetsen = m.t("diag.report"), m.t("diag.reportkeys")
	}
	return kop + "\n\n" + paneel(p, titel, strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(toetsen)
}
