package main

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShellWaarde(t *testing.T) {
	gevallen := map[string]string{
		`abc`:                 "abc",
		`"a b"`:               "a b",
		`'it'\''s'`:           "it's",
		`'p#ss word'`:         "p#ss word",
		`a\ b`:                "a b",
		`"x\"y"`:              `x"y`,
		`waarde # commentaar`: "waarde",
		`ab#c`:                "ab#c",
		`""`:                  "",
		`'pa'\''ss $x"y#z'`:   `pa'ss $x"y#z`,
	}
	for in, uit := range gevallen {
		if got := shellWaarde(in); got != uit {
			t.Errorf("shellWaarde(%s) = %q, verwacht %q", in, got, uit)
		}
	}
	pad := filepath.Join(t.TempDir(), "geheimen.env")
	_ = os.WriteFile(pad, []byte("# commentaar\nNAVIDROME_URL='http://nas:4533'\nexport ABS_TOKEN=\"abc\"\n\nLEEG=\n"), 0o600)
	t.Setenv("UNSCROLL_GEHEIMEN", pad)
	g := laadGeheimen()
	if g["NAVIDROME_URL"] != "http://nas:4533" || g["ABS_TOKEN"] != "abc" || g["LEEG"] != "" {
		t.Errorf("geheimen verkeerd gelezen: %v", g)
	}
}

func nepNavidrome(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("u") != "kind" || q.Get("t") == "" || q.Get("s") == "" || q.Get("p") != "" {
			t.Errorf("verkeerde inlog: %v", q)
		}
		switch r.URL.Path {
		case "/rest/getArtists":
			_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","artists":{"index":[{"artist":[{"id":"a2","name":"zappa","albumCount":1},{"id":"a1","name":"ABBA","albumCount":2}]}]}}}`))
		case "/rest/getArtist":
			_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","artist":{"album":[{"id":"al1","name":"Arrival","year":1976,"songCount":2}]}}}`))
		case "/rest/getAlbum":
			_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","album":{"name":"Arrival","song":[{"id":"s1","title":"When I Kissed the Teacher","artist":"ABBA","track":1,"duration":181},{"id":"s2","title":"Dancing Queen","artist":"ABBA","track":2,"duration":231}]}}}`))
		default:
			_, _ = w.Write([]byte(`{"subsonic-response":{"status":"failed","error":{"code":70,"message":"not found"}}}`))
		}
	}))
}

func TestNavidrome(t *testing.T) {
	srv := nepNavidrome(t)
	defer srv.Close()
	n := nieuweNavidrome(Geheimen{"NAVIDROME_URL": srv.URL + "/", "NAVIDROME_GEBRUIKER": "kind", "NAVIDROME_WACHTWOORD": "geheim"})
	art, err := n.Artiesten()
	if err != nil || len(art) != 2 || art[0].titel != "ABBA" {
		t.Fatalf("artiesten: %v %v", art, err)
	}
	al, err := n.Albums("a1")
	if err != nil || len(al) != 1 || al[0].info != "1976" {
		t.Fatalf("albums: %v %v", al, err)
	}
	nr, err := n.Nummers("al1")
	if err != nil || len(nr) != 2 || nr[1].info != "3:51" || !strings.Contains(nr[1].url, "/rest/stream?") || strings.Contains(nr[1].url, "geheim") {
		t.Fatalf("nummers: %v %v", nr, err)
	}
	if err := n.vraag("bestaatNiet", nil, &struct{}{}); err == nil {
		t.Error("een fout van Navidrome moet een fout geven")
	}
	if nieuweNavidrome(Geheimen{}) != nil {
		t.Error("zonder adres is Navidrome niet ingesteld")
	}
}

func nepABS(t *testing.T, gesloten *[]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sleutel" {
			w.WriteHeader(401)
			return
		}
		switch {
		case r.URL.Path == "/api/libraries":
			_, _ = w.Write([]byte(`{"libraries":[{"id":"lb","name":"Boeken","mediaType":"book"},{"id":"lp","name":"Podcasts","mediaType":"podcast"}]}`))
		case r.URL.Path == "/api/me":
			_, _ = w.Write([]byte(`{"mediaProgress":[{"libraryItemId":"b1","progress":0.42},{"libraryItemId":"p1","episodeId":"e2","isFinished":true}]}`))
		case r.URL.Path == "/api/libraries/lb/items":
			_, _ = w.Write([]byte(`{"results":[{"id":"b1","media":{"metadata":{"title":"Pluk van de Petteflet","authorName":"Annie M.G. Schmidt"},"duration":7200}}]}`))
		case r.URL.Path == "/api/libraries/lp/items":
			_, _ = w.Write([]byte(`{"results":[{"id":"p1","media":{"metadata":{"title":"Klokhuis","author":"NTR"},"numEpisodes":2}}]}`))
		case r.URL.Path == "/api/items/p1":
			_, _ = w.Write([]byte(`{"media":{"metadata":{"title":"Klokhuis"},"episodes":[{"id":"e1","title":"Oud","publishedAt":1000},{"id":"e2","title":"Nieuw","publishedAt":2000,"audioFile":{"duration":600}}]}}`))
		case r.URL.Path == "/api/items/b1/play" && r.Method == "POST":
			_, _ = w.Write([]byte(`{"id":"sessie1","currentTime":4000,"duration":7200,"displayTitle":"Pluk","audioTracks":[{"startOffset":0,"duration":3600,"contentUrl":"/api/items/b1/file/1"},{"startOffset":3600,"duration":3600,"contentUrl":"/api/items/b1/file/2"}]}`))
		case strings.HasPrefix(r.URL.Path, "/api/session/"):
			*gesloten = append(*gesloten, r.URL.Path)
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestAudiobookshelf(t *testing.T) {
	var gesloten []string
	srv := nepABS(t, &gesloten)
	defer srv.Close()
	a := nieuweABS(Geheimen{"ABS_URL": srv.URL, "ABS_TOKEN": "sleutel"}, "test")
	boeken, err := a.Items("book")
	if err != nil || len(boeken) != 1 || boeken[0].voortgang != 0.42 || boeken[0].info != "2:00:00" {
		t.Fatalf("boeken: %+v %v", boeken, err)
	}
	pods, err := a.Items("podcast")
	if err != nil || len(pods) != 1 || pods[0].sub != "NTR" || pods[0].soort != soortPodcast {
		t.Fatalf("podcasts: %+v %v", pods, err)
	}
	eps, err := a.Afleveringen("p1")
	if err != nil || len(eps) != 2 || eps[0].titel != "Nieuw" || !eps[0].klaar || eps[0].duur != 600 {
		t.Fatalf("afleveringen: %+v %v", eps, err)
	}
	if u := a.SpoorURL("/api/items/b1/file/1"); u != srv.URL+"/api/items/b1/file/1?token=sleutel" {
		t.Errorf("spoor-url: %s", u)
	}
	fout := nieuweABS(Geheimen{"ABS_URL": srv.URL, "ABS_TOKEN": "fout"}, "test")
	if _, err := fout.Items("book"); err == nil {
		t.Error("verkeerde sleutel moet een fout geven")
	}
}

// nepMpv doet alsof het mpv is: het beantwoordt opdrachten op een unix-socket.
type nepMpv struct {
	mu         sync.Mutex
	opdrachten [][]any
	leeg       bool
	pos        float64
	index      int
}

func startNepMpv(t *testing.T) (*nepMpv, string) {
	sock := filepath.Join(t.TempDir(), "mpv.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	n := &nepMpv{pos: 12.5}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				rd := bufio.NewReader(c)
				for {
					regel, err := rd.ReadBytes('\n')
					if err != nil {
						return
					}
					var v struct {
						Command   []any `json:"command"`
						RequestID int   `json:"request_id"`
					}
					_ = json.Unmarshal(regel, &v)
					n.mu.Lock()
					n.opdrachten = append(n.opdrachten, v.Command)
					var data any
					if v.Command[0] == "get_property" {
						switch v.Command[1] {
						case "idle-active":
							data = n.leeg
						case "time-pos":
							data = n.pos
						case "playlist-pos":
							data = n.index
						case "duration":
							data = 3600
						case "pause":
							data = false
						}
					}
					n.mu.Unlock()
					_, _ = c.Write([]byte(`{"event":"property-change"}` + "\n"))
					uit, _ := json.Marshal(map[string]any{"request_id": v.RequestID, "error": "success", "data": data})
					_, _ = c.Write(append(uit, '\n'))
				}
			}()
		}
	}()
	return n, sock
}

func (n *nepMpv) heeft(woord string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, o := range n.opdrachten {
		if strings.Contains(jsonTekst(o), woord) {
			return true
		}
	}
	return false
}

func jsonTekst(v any) string { b, _ := json.Marshal(v); return string(b) }

// voer laat een opdracht en alle berichten die eruit volgen door het model lopen,
// behalve de tikken van de speler (die worden apart getest).
func voer(m model, cmd tea.Cmd) model {
	for i := 0; i < 20 && cmd != nil; i++ {
		uit := make(chan tea.Msg, 1)
		go func(c tea.Cmd) { uit <- c() }(cmd)
		var msg tea.Msg
		select {
		case msg = <-uit:
		case <-time.After(1500 * time.Millisecond):
			return m // klokken van 20 seconden (tick) slaan we over
		}
		cmd = nil
		switch v := msg.(type) {
		case tea.BatchMsg:
			for _, c := range v {
				m = voer(m, c)
			}
		case tea.Cmd:
			cmd = v
		case nil:
		default:
			if rv := reflect.ValueOf(msg); rv.Kind() == reflect.Slice { // tea.Sequence
				for i := 0; i < rv.Len(); i++ {
					if c, ok := rv.Index(i).Interface().(tea.Cmd); ok {
						m = voer(m, c)
					}
				}
				continue
			}
			if _, ok := msg.(spelerMsg); ok { // één tik verwerken, de volgende niet afwachten
				nm, _ := m.Update(msg)
				return nm.(model)
			}
			var nm tea.Model
			nm, cmd = m.Update(msg)
			m = nm.(model)
		}
	}
	return m
}

func toets(m model, k string) model {
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case " ":
		msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	nm, cmd := m.Update(msg)
	return voer(nm.(model), cmd)
}

func testModel(t *testing.T, g Geheimen) (model, *nepMpv) {
	dir := t.TempDir()
	t.Setenv("UNSCROLL_INSTELLINGEN", filepath.Join(dir, "instellingen.json"))
	t.Setenv("UNSCROLL_OUDERS", filepath.Join(dir, "bestaat-niet.yaml"))
	m := nieuwModel()
	m.ouders.Avondslot.Schooldagen, m.ouders.Avondslot.Weekend, m.ouders.Avondslot.Open = "23:59", "23:59", "00:00"
	m.l = nieuwLuister(g)
	n, sock := startNepMpv(t)
	m.l.sp = &Speler{sock: sock, extern: true}
	m.breed, m.hoog = 120, 36
	return m, n
}

func TestLuisterenMuziek(t *testing.T) {
	srv := nepNavidrome(t)
	defer srv.Close()
	m, mpv := testModel(t, Geheimen{"NAVIDROME_URL": srv.URL, "NAVIDROME_GEBRUIKER": "kind", "NAVIDROME_WACHTWOORD": "geheim"})

	m = toets(m, "1")
	if m.tab != tabLuisteren || !strings.Contains(m.View(), "ABBA") {
		t.Fatalf("artiesten niet te zien:\n%s", m.View())
	}
	m = toets(m, "enter") // ABBA
	m = toets(m, "enter") // Arrival
	if !strings.Contains(m.View(), "Dancing Queen") {
		t.Fatalf("nummers niet te zien:\n%s", m.View())
	}
	m = toets(m, "/")
	m = toets(m, "d")
	m = toets(m, "a")
	if strings.Contains(m.View(), "Teacher") || !strings.Contains(m.View(), "Dancing Queen") {
		t.Fatalf("zoeken werkt niet:\n%s", m.View())
	}
	m = toets(m, "enter") // speel Dancing Queen
	if m.l.nu == nil || len(m.l.nu.items) != 1 || !mpv.heeft("loadfile") {
		t.Fatalf("speelt niet: %+v", m.l.nu)
	}
	// een tik van de speler: hij speelt
	m = voer(m, func() tea.Msg { st, err := m.l.sp.Stand(); return spelerMsg{m.l.pollID, st, err} })
	if !m.l.nu.gestart {
		t.Error("speler zou gestart moeten zijn")
	}
	m = toets(m, " ")
	if !m.l.nu.stand.Pauze || !mpv.heeft(`"pause",true`) {
		t.Error("pauze werkt niet")
	}
	m = toets(m, "esc") // zoekwoord weg
	m = toets(m, "esc") // terug naar albums
	if len(m.l.stapels[bronMuziek]) != 2 {
		t.Errorf("esc moet een laag terug gaan, stapel is %d", len(m.l.stapels[bronMuziek]))
	}
	m = toets(m, "x")
	if m.l.nu != nil || !mpv.heeft(`["stop"]`) {
		t.Error("stoppen werkt niet")
	}
}

func TestLuisterboekVoortgang(t *testing.T) {
	var gesloten []string
	srv := nepABS(t, &gesloten)
	defer srv.Close()
	m, mpv := testModel(t, Geheimen{"ABS_URL": srv.URL, "ABS_TOKEN": "sleutel"})
	m = toets(m, "1")
	m = toets(m, "tab") // luisterboeken
	if !strings.Contains(m.View(), "Pluk van de Petteflet") || !strings.Contains(m.View(), "42%") {
		t.Fatalf("boeken niet te zien:\n%s", m.View())
	}
	m = toets(m, "enter")
	a := m.l.nu
	if a == nil || a.sessie != "sessie1" || len(a.sporen) != 1 || !mpv.heeft(`"start","400.0"`) || !mpv.heeft("Bearer sleutel") {
		t.Fatalf("luisterboek niet goed gestart: %+v %v", a, mpv.opdrachten)
	}
	m = voer(m, func() tea.Msg { st, err := m.l.sp.Stand(); return spelerMsg{m.l.pollID, st, err} })
	if got := a.positie(); got != 3612.5 {
		t.Errorf("positie in het boek %v, verwacht 3612.5 (spoor 2 begint op 3600)", got)
	}
	_ = m
	m = toets(m, "x")
	if len(gesloten) == 0 || !strings.HasSuffix(gesloten[len(gesloten)-1], "/close") {
		t.Errorf("sessie niet gesloten: %v", gesloten)
	}
}

func TestNietIngesteldEnAvondslot(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m = toets(m, "1")
	if !strings.Contains(m.View(), "sudo unscroll-ouder") {
		t.Errorf("zonder instellingen moet de uitleg voor ouders te zien zijn:\n%s", m.View())
	}
	// radio werkt altijd
	for i := 0; i < 3; i++ {
		m = toets(m, "tab")
	}
	if !strings.Contains(m.View(), "NPO Radio 1") || !strings.Contains(m.View(), "▲") {
		t.Errorf("radio niet te zien:\n%s", m.View())
	}
	m = toets(m, "enter")
	if m.l.nu == nil || m.l.nu.soort != soortZender {
		t.Fatal("radio speelt niet")
	}
	// avondslot begint: radio stopt, alleen luisterboeken
	m.ouders.Avondslot.Schooldagen, m.ouders.Avondslot.Weekend, m.ouders.Avondslot.Open = "00:00", "00:00", "23:59"
	m.ontgrendeld = time.Time{}
	nm, cmd := m.Update(tickMsg(time.Now()))
	m = voer(nm.(model), cmd)
	if m.l.nu != nil || m.l.bron != bronBoeken {
		t.Errorf("tijdens het avondslot moet de radio stoppen en alleen luisterboeken over blijven")
	}
	m = toets(m, "tab")
	if m.l.bron != bronBoeken {
		t.Error("tijdens het avondslot kun je niet naar een andere bron")
	}
}

func TestSlaaptimer(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m = toets(m, "1")
	m = toets(m, "s")
	if m.l.slaapMin != 15 || time.Until(m.l.slaapTot) < 14*time.Minute {
		t.Errorf("slaaptimer: %d", m.l.slaapMin)
	}
	for i := 0; i < 4; i++ {
		m = toets(m, "s")
	}
	if m.l.slaapMin != 0 || !m.l.slaapTot.IsZero() {
		t.Error("na 60 minuten moet de slaaptimer weer uit gaan")
	}
}
