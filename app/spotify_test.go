package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func TestSpotifyLink(t *testing.T) {
	gevallen := map[string]string{
		"https://open.spotify.com/album/4aawyAB9vmqN3uQ7FjRGTy?si=abc":     "album 4aawyAB9vmqN3uQ7FjRGTy",
		"https://open.spotify.com/intl-nl/playlist/37i9dQZF1DXcBWIGoYBM5M": "playlist 37i9dQZF1DXcBWIGoYBM5M",
		"spotify:album:123":                  "album 123",
		"https://open.spotify.com/track/123": "",
		"https://example.com/album/123":      "",
		"spotify:artist:123":                 "",
	}
	for in, uit := range gevallen {
		soort, id, ok := spotifyLink(in)
		got := ""
		if ok {
			got = soort + " " + id
		}
		if got != uit {
			t.Errorf("spotifyLink(%q) = %q, verwacht %q", in, got, uit)
		}
	}
}

// nepSpotify doet alsof het Spotify is (accounts en Web API op één server).
type nepSpotify struct {
	mu         sync.Mutex
	paden      []string
	speelBody  string
	nu         string // uri die nu speelt
	speelt     bool
	nieuwToken string
}

func (n *nepSpotify) server(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.mu.Lock()
		defer n.mu.Unlock()
		n.paden = append(n.paden, r.Method+" "+r.URL.Path)
		schrijf := func(v any) { w.Header().Set("Content-Type", "application/json"); _ = json.NewEncoder(w).Encode(v) }
		if r.URL.Path == "/api/token" {
			_ = r.ParseForm()
			if r.Form.Get("grant_type") == "authorization_code" {
				if r.Form.Get("code") != "de-code" || r.Form.Get("code_verifier") == "" {
					w.WriteHeader(400)
					return
				}
				schrijf(map[string]any{"access_token": "a", "refresh_token": "verse-token", "expires_in": 3600})
				return
			}
			if r.Form.Get("refresh_token") == "verlopen" {
				w.WriteHeader(400)
				return
			}
			schrijf(map[string]any{"access_token": "toegang", "refresh_token": n.nieuwToken, "expires_in": 3600})
			return
		}
		if r.Header.Get("Authorization") != "Bearer toegang" {
			w.WriteHeader(401)
			return
		}
		nr := func(uri, naam string, expliciet bool) map[string]any {
			return map[string]any{"uri": uri, "name": naam, "duration_ms": 180000, "explicit": expliciet, "type": "track", "artists": []map[string]string{{"name": "Band"}}}
		}
		switch {
		case r.URL.Path == "/albums/alb1":
			schrijf(map[string]any{"name": "Goed Album", "artists": []map[string]string{{"name": "Band"}}})
		case r.URL.Path == "/playlists/pl1":
			schrijf(map[string]any{"name": "Onze Lijst", "owner": map[string]string{"display_name": "Papa"}})
		case r.URL.Path == "/albums/alb1/tracks" && r.URL.Query().Get("offset") == "":
			schrijf(map[string]any{"items": []any{nr("spotify:track:1", "Een", false), nr("spotify:track:2", "Twee (expliciet)", true)}, "next": "http://" + r.Host + "/albums/alb1/tracks?offset=2"})
		case r.URL.Path == "/albums/alb1/tracks":
			schrijf(map[string]any{"items": []any{nr("spotify:track:3", "Drie", false)}})
		case r.URL.Path == "/playlists/pl1/items":
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":{"status":404,"message":"Not found"}}`))
		case r.URL.Path == "/playlists/pl1/tracks":
			schrijf(map[string]any{"items": []any{map[string]any{"track": nr("spotify:track:9", "Lijstnummer", false)}}})
		case r.URL.Path == "/search":
			schrijf(map[string]any{
				"artists": map[string]any{"items": []map[string]string{{"id": "art1", "name": "Band"}, {"id": "art2", "name": "Foute Band"}}},
				"albums": map[string]any{"items": []any{
					map[string]any{"id": "alb1", "name": "Goed Album", "release_date": "2020-01-01", "artists": []map[string]string{{"id": "art1", "name": "Band"}}},
					map[string]any{"id": "alb9", "name": "Fout Album", "artists": []map[string]string{{"id": "art2", "name": "Foute Band"}}}}},
			})
		case r.URL.Path == "/artists/art1/albums":
			schrijf(map[string]any{"items": []any{map[string]any{"id": "alb1", "name": "Goed Album", "release_date": "2020", "artists": []map[string]string{{"id": "art1", "name": "Band"}}}}})
		case r.URL.Path == "/me/player/devices":
			schrijf(map[string]any{"devices": []map[string]string{{"id": "telefoon", "name": "iPhone"}, {"id": "dev1", "name": spotifyApparaatNaam()}}})
		case r.URL.Path == "/me/player/play" && r.Method == "PUT":
			b, _ := io.ReadAll(r.Body)
			if len(b) > 0 {
				n.speelBody = string(b)
				n.nu = "spotify:track:1"
			}
			n.speelt = true
			w.WriteHeader(204)
		case r.URL.Path == "/me/player/pause":
			n.speelt = false
			w.WriteHeader(204)
		case r.URL.Path == "/me/player" && r.Method == "GET":
			if n.nu == "" {
				w.WriteHeader(204)
				return
			}
			schrijf(map[string]any{"device": map[string]string{"name": spotifyApparaatNaam()}, "is_playing": n.speelt, "progress_ms": 42000,
				"item": map[string]any{"uri": n.nu, "duration_ms": 180000}})
		default:
			w.WriteHeader(204)
		}
	}))
}

func (n *nepSpotify) gezien(pad string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, p := range n.paden {
		if strings.HasPrefix(p, pad) {
			return true
		}
	}
	return false
}

func testSpotify(t *testing.T, nep *nepSpotify) *Spotify {
	srv := nep.server(t)
	t.Cleanup(srv.Close)
	s := nieuweSpotify(Geheimen{"SPOTIFY_CLIENT_ID": "klant", "SPOTIFY_REFRESH_TOKEN": "eerste"})
	s.api, s.accounts = srv.URL, srv.URL
	return s
}

func TestSpotifyLijstenEnAfspelen(t *testing.T) {
	t.Setenv("UNSCROLL_INSTELLINGEN", t.TempDir()+"/instellingen.json")
	nep := &nepSpotify{}
	s := testSpotify(t, nep)

	lijsten, err := s.Goedgekeurd([]string{"https://open.spotify.com/album/alb1", "spotify:playlist:pl1", "https://open.spotify.com/track/x"})
	if err != nil || len(lijsten) != 2 || lijsten[0].titel != "Goed Album" || lijsten[1].sub != "Papa" {
		t.Fatalf("goedgekeurd: %+v %v", lijsten, err)
	}
	nummers, verborgen, err := s.Nummers("album:alb1", false)
	if err != nil || len(nummers) != 2 || verborgen != 1 || nummers[1].titel != "Drie" {
		t.Fatalf("albumnummers (expliciet weg, tweede pagina erbij): %+v %d %v", nummers, verborgen, err)
	}
	if alle, _, _ := s.Nummers("album:alb1", true); len(alle) != 3 {
		t.Error("met expliciet_mag horen alle nummers erbij")
	}
	lijst, _, err := s.Nummers("playlist:pl1", false)
	if err != nil || len(lijst) != 1 || lijst[0].titel != "Lijstnummer" {
		t.Fatalf("playlist (terugval naar /tracks): %+v %v", lijst, err)
	}

	if err := s.Speel([]string{"spotify:track:1", "spotify:track:3"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(nep.speelBody, "spotify:track:3") || !nep.gezien("PUT /me/player/shuffle") || !nep.gezien("PUT /me/player/repeat") {
		t.Errorf("afspelen zonder shuffle en herhalen: %s %v", nep.speelBody, nep.paden)
	}
	st, err := s.Stand(nummers)
	if err != nil || st.Leeg || st.Index != 0 || st.Pos != 42 {
		t.Errorf("stand: %+v %v", st, err)
	}
	// iets dat niet in de lijst staat (autoplay): pauzeren en klaar
	nep.mu.Lock()
	nep.nu, nep.speelt = "spotify:track:aanbevolen", true
	nep.mu.Unlock()
	st, _ = s.Stand(nummers)
	if !st.Leeg || nep.speelt {
		t.Error("een nummer buiten de lijst moet gepauzeerd worden")
	}
}

func TestSpotifyTokenWissel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("UNSCROLL_INSTELLINGEN", dir+"/instellingen.json")
	nep := &nepSpotify{nieuwToken: "tweede"}
	s := testSpotify(t, nep)
	if _, err := s.token(); err != nil {
		t.Fatal(err)
	}
	// een herstart gebruikt de nieuwe refresh token
	if s2 := nieuweSpotify(Geheimen{"SPOTIFY_CLIENT_ID": "klant", "SPOTIFY_REFRESH_TOKEN": "eerste"}); s2.vernieuw != "tweede" {
		t.Errorf("bewaarde token niet gebruikt: %q", s2.vernieuw)
	}
	// logt een ouder opnieuw in, dan wint de token uit geheimen.env
	if s3 := nieuweSpotify(Geheimen{"SPOTIFY_CLIENT_ID": "klant", "SPOTIFY_REFRESH_TOKEN": "opnieuw"}); s3.vernieuw != "opnieuw" {
		t.Errorf("nieuwe login van de ouders moet winnen: %q", s3.vernieuw)
	}
	info, _ := os.Stat(spotifyTokenPad())
	if info.Mode().Perm() != 0o600 {
		t.Errorf("tokenbestand moet alleen voor het kind leesbaar zijn: %v", info.Mode())
	}
	verlopen := nieuweSpotify(Geheimen{"SPOTIFY_CLIENT_ID": "klant", "SPOTIFY_REFRESH_TOKEN": "verlopen"})
	verlopen.accounts = s.accounts
	if _, err := verlopen.token(); err != errSpotifyIngelogd {
		t.Errorf("verlopen login: %v", err)
	}
}

// plakker speelt de ouder: leest de link, "logt in" en plakt het adres terug.
type plakker struct {
	w   *io.PipeWriter
	buf bytes.Buffer
	ok  bool
}

func (p *plakker) Write(b []byte) (int, error) {
	p.buf.Write(b)
	if m := regexp.MustCompile(`state=([A-Za-z0-9_-]+)`).FindStringSubmatch(p.buf.String()); m != nil && !p.ok {
		p.ok = true
		go func() { _, _ = io.WriteString(p.w, spotifyRedirect+"?code=de-code&state="+url.QueryEscape(m[1])+"\n") }()
	}
	return len(b), nil
}

func TestSpotifyInloggen(t *testing.T) {
	nep := &nepSpotify{}
	srv := nep.server(t)
	defer srv.Close()
	r, w := io.Pipe()
	var uit bytes.Buffer
	meld := &plakker{w: w}
	if err := spotifyInloggenBij(srv.URL, "klant", r, &uit, meld); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(uit.String()) != "verse-token" {
		t.Errorf("alleen de refresh token hoort op stdout, kreeg %q", uit.String())
	}
	if !strings.Contains(meld.buf.String(), "code_challenge_method=S256") || !strings.Contains(meld.buf.String(), "user-modify-playback-state") {
		t.Errorf("inloglink mist PKCE of rechten: %s", meld.buf.String())
	}
	// een adres van een andere poging wordt geweigerd
	r2 := strings.NewReader(spotifyRedirect + "?code=de-code&state=fout\n")
	if err := spotifyInloggenBij(srv.URL, "klant", r2, io.Discard, io.Discard); err == nil {
		t.Error("verkeerde state moet geweigerd worden")
	}
}

func TestSpotifyInLuisteren(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	nep := &nepSpotify{}
	m.l.spotify = testSpotify(t, nep)
	m.ouders.Spotify.Goedgekeurd = []string{"https://open.spotify.com/album/alb1"}
	m = toets(m, "1")
	for i := 0; i < 4; i++ {
		m = toets(m, "tab")
	}
	if m.l.bron != bronSpotify || !strings.Contains(m.View(), "Goed Album") {
		t.Fatalf("Spotify-lijst niet te zien:\n%s", m.View())
	}
	m = toets(m, "enter")
	if strings.Contains(m.View(), "expliciet") || !strings.Contains(m.View(), "Drie") {
		t.Fatalf("nummers niet goed gefilterd:\n%s", m.View())
	}
	t.Setenv("PATH", t.TempDir()) // librespot is hier niet geïnstalleerd
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "librespot") {
		t.Errorf("zonder librespot moet dat uitgelegd worden:\n%s", m.View())
	}
}

func TestSpotifyZoekenEnFavorieten(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m.l.spotify = testSpotify(t, &nepSpotify{})
	m.ouders.Spotify.Zoeken = true
	m.ouders.Spotify.Geblokkeerd = []string{"foute band"}
	m = toets(m, "1")
	for i := 0; i < 4; i++ {
		m = toets(m, "tab")
	}
	if !strings.Contains(m.View(), "Search for an artist") {
		t.Fatalf("zoeken niet te zien:\n%s", m.View())
	}
	m = toets(m, "enter")
	m = toets(m, "band")
	m = toets(m, "enter")
	v := m.View()
	if !strings.Contains(v, "Goed Album") || strings.Contains(v, "Foute") || strings.Contains(v, "Fout Album") {
		t.Fatalf("zoekresultaten (geblokkeerd hoort weg):\n%s", v)
	}
	m = toets(m, "f") // eerste resultaat: artiest Band
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "Goed Album") || len(m.l.stapels[bronSpotify]) != 3 {
		t.Fatalf("albums van de artiest:\n%s", m.View())
	}
	m = toets(m, "esc")
	m = toets(m, "esc")
	m = toets(m, "r")
	if !strings.Contains(m.View(), "★ Band") {
		t.Errorf("favoriet niet in de lijst:\n%s", m.View())
	}
	m.ouderModus = true
	m = toets(m, "w")
	if !strings.Contains(m.View(), "searched: band") {
		t.Errorf("ouders zien de zoekvraag niet:\n%s", m.View())
	}
}
