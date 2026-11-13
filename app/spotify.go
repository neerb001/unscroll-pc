package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Spotify zonder algoritme: alleen albums en playlists die de ouders hebben goedgekeurd,
// geen aanbevelingen, geen zoeken, geen autoplay. Het geluid komt uit librespot (een
// Spotify Connect-speler op dit apparaat); de app bestuurt hem via de Spotify Web API.

const (
	spotifyAPI      = "https://api.spotify.com/v1"
	spotifyAccounts = "https://accounts.spotify.com"
	spotifyRedirect = "http://127.0.0.1:8898/callback"
	spotifyScopes   = "user-read-playback-state user-modify-playback-state user-read-currently-playing playlist-read-private playlist-read-collaborative"
)

var (
	errGeenLibrespot   = errors.New("librespot not installed")
	errNietGekoppeld   = errors.New("spotify speaker not linked")
	errGeenApparaat    = errors.New("spotify device not found")
	errSpotifyIngelogd = errors.New("spotify login expired")
	errSpotifyDruk     = errors.New("spotify: too many requests")
)

type Spotify struct {
	klantID, apparaat string
	api, accounts     string
	http              *http.Client

	mu         sync.Mutex
	vernieuw   string // refresh token
	bron       string // hash van de token uit geheimen.env, om te zien of een ouder opnieuw heeft ingelogd
	toegang    string
	verloopt   time.Time
	apparaatID string
	wachtTijd  time.Duration
}

func nieuweSpotify(g Geheimen) *Spotify {
	id, tok := strings.TrimSpace(g["SPOTIFY_CLIENT_ID"]), strings.TrimSpace(g["SPOTIFY_REFRESH_TOKEN"])
	if id == "" || tok == "" {
		return nil
	}
	s := &Spotify{klantID: id, api: spotifyAPI, accounts: spotifyAccounts, http: &http.Client{Timeout: 10 * time.Second},
		apparaat: spotifyApparaatNaam(), vernieuw: tok, bron: hash(tok)}
	// Spotify geeft bij het vernieuwen soms een nieuwe refresh token; die bewaren we bij het kind
	// (het geheimenbestand is van root). Logt een ouder opnieuw in, dan wint de nieuwe uit geheimen.env.
	var opgeslagen struct{ Bron, Token string }
	if data, err := os.ReadFile(spotifyTokenPad()); err == nil && json.Unmarshal(data, &opgeslagen) == nil && opgeslagen.Bron == s.bron && opgeslagen.Token != "" {
		s.vernieuw = opgeslagen.Token
	}
	return s
}

func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:8]) }

func spotifyTokenPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "spotify.json") }

func spotifyApparaatNaam() string {
	h, _ := os.Hostname()
	if h == "" {
		h = "pc"
	}
	return "Unscroll-" + h
}

// token geeft een geldige toegangssleutel en vernieuwt hem als dat nodig is.
func (s *Spotify) token() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.toegang != "" && time.Now().Before(s.verloopt) {
		return s.toegang, nil
	}
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {s.vernieuw}, "client_id": {s.klantID}}
	resp, err := s.http.PostForm(s.accounts+"/api/token", form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var t struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		return "", errSpotifyIngelogd
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("spotify: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", err
	}
	if t.AccessToken == "" {
		return "", errSpotifyIngelogd
	}
	s.toegang, s.verloopt = t.AccessToken, time.Now().Add(time.Duration(t.ExpiresIn-60)*time.Second)
	if t.RefreshToken != "" && t.RefreshToken != s.vernieuw {
		s.vernieuw = t.RefreshToken
		data, _ := json.Marshal(map[string]string{"Bron": s.bron, "Token": t.RefreshToken})
		_ = os.MkdirAll(filepath.Dir(spotifyTokenPad()), 0o700)
		_ = veiligSchrijven(spotifyTokenPad(), data, 0o600)
	}
	return s.toegang, nil
}

// doe stuurt een verzoek naar de Web API. uit mag nil zijn.
func (s *Spotify) doe(methode, pad string, in, uit any) (int, error) {
	code, err := s.doeEens(methode, pad, in, uit)
	if code == http.StatusUnauthorized {
		// toegangssleutel ingetrokken of verlopen: één keer een nieuwe halen en opnieuw proberen
		s.mu.Lock()
		s.toegang = ""
		s.mu.Unlock()
		code, err = s.doeEens(methode, pad, in, uit)
		if code == http.StatusUnauthorized {
			return code, errSpotifyIngelogd
		}
	}
	if code == http.StatusTooManyRequests {
		// Spotify vraagt even te wachten; kort wachten we zelf, anders melden we het
		if w := s.wacht(); w > 0 && w <= 5*time.Second {
			time.Sleep(w)
			code, err = s.doeEens(methode, pad, in, uit)
		}
		if code == http.StatusTooManyRequests {
			return code, fmt.Errorf("%w, retry after %s", errSpotifyDruk, klok(s.wacht().Seconds()))
		}
	}
	return code, err
}

func (s *Spotify) wacht() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wachtTijd
}

func (s *Spotify) doeEens(methode, pad string, in, uit any) (int, error) {
	tok, err := s.token()
	if err != nil {
		return 0, err
	}
	var body io.Reader
	if in != nil {
		data, _ := json.Marshal(in)
		body = strings.NewReader(string(data))
	}
	req, err := http.NewRequest(methode, s.api+pad, body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		sec, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		s.mu.Lock()
		s.wachtTijd = time.Duration(max(sec, 1)) * time.Second
		s.mu.Unlock()
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error.Message == "" {
			e.Error.Message = resp.Status
		}
		return resp.StatusCode, fmt.Errorf("spotify: %s", e.Error.Message)
	}
	if uit != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(uit); err != nil && err != io.EOF {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

/* ---------------- goedgekeurde albums en playlists ---------------- */

// spotifyLink haalt soort en id uit een link of URI van Spotify.
// https://open.spotify.com/album/ID?si=x, https://open.spotify.com/intl-nl/playlist/ID en spotify:album:ID werken alle drie.
func spotifyLink(s string) (soort, id string, ok bool) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "spotify:") {
		d := strings.Split(s, ":")
		if len(d) == 3 {
			soort, id = d[1], d[2]
		}
	} else if u, err := url.Parse(s); err == nil && strings.HasSuffix(u.Host, "spotify.com") {
		d := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(d) >= 2 {
			soort, id = d[len(d)-2], d[len(d)-1]
		}
	}
	if (soort == "album" || soort == "playlist") && id != "" {
		return soort, id, true
	}
	return "", "", false
}

type spTrack struct {
	URI        string `json:"uri"`
	Name       string `json:"name"`
	DurationMS int    `json:"duration_ms"`
	Explicit   bool   `json:"explicit"`
	IsLocal    bool   `json:"is_local"`
	Type       string `json:"type"`
	Artists    []struct {
		Name string `json:"name"`
	} `json:"artists"`
}

func (t spTrack) artiesten() string {
	var n []string
	for _, a := range t.Artists {
		n = append(n, a.Name)
	}
	return strings.Join(n, ", ")
}

// Goedgekeurd haalt de naam op van elk goedgekeurd album en elke playlist.
func (s *Spotify) Goedgekeurd(links []string) ([]item, error) {
	var uit []item
	var laatsteFout error
	for _, l := range links {
		soort, id, ok := spotifyLink(l)
		if !ok {
			continue
		}
		var r struct {
			Name    string `json:"name"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Owner struct {
				DisplayName string `json:"display_name"`
			} `json:"owner"`
		}
		pad := "/albums/" + id
		if soort == "playlist" {
			pad = "/playlists/" + id + "?fields=name,owner(display_name)"
		}
		if _, err := s.doe("GET", pad, nil, &r); err != nil {
			laatsteFout = err
			continue
		}
		it := item{id: soort + ":" + id, titel: r.Name, soort: soortSpLijst}
		if len(r.Artists) > 0 {
			it.sub = r.Artists[0].Name
		} else {
			it.sub = r.Owner.DisplayName
		}
		uit = append(uit, it)
	}
	if len(uit) == 0 && laatsteFout != nil {
		return nil, laatsteFout
	}
	return uit, nil
}

// Nummers van een goedgekeurd album of playlist. Expliciete nummers vallen weg tenzij de ouders ze toestaan.
func (s *Spotify) Nummers(lijst string, expliciet bool) ([]item, int, error) {
	soort, id, _ := strings.Cut(lijst, ":")
	var sporen []spTrack
	if soort == "album" {
		pad := "/albums/" + id + "/tracks"
		for pad != "" {
			var r struct {
				Items []spTrack `json:"items"`
				Next  string    `json:"next"`
			}
			if _, err := s.doe("GET", pad, nil, &r); err != nil {
				return nil, 0, err
			}
			sporen = append(sporen, r.Items...)
			pad = strings.TrimPrefix(r.Next, s.api)
		}
	} else {
		// nieuwe naam van het eindpunt eerst, de oude als terugval
		pad := "/playlists/" + id + "/items"
		eerste := true
		for pad != "" {
			var r struct {
				Items []struct {
					Item  *spTrack `json:"item"`
					Track *spTrack `json:"track"`
				} `json:"items"`
				Next string `json:"next"`
			}
			code, err := s.doe("GET", pad, nil, &r)
			if err != nil && eerste && code == http.StatusNotFound {
				pad = "/playlists/" + id + "/tracks"
				eerste = false
				continue
			}
			if err != nil {
				return nil, 0, err
			}
			eerste = false
			for _, x := range r.Items {
				if t := x.Item; t != nil {
					sporen = append(sporen, *t)
				} else if x.Track != nil {
					sporen = append(sporen, *x.Track)
				}
			}
			pad = strings.TrimPrefix(r.Next, s.api)
		}
	}
	var uit []item
	verborgen := 0
	for _, t := range sporen {
		if t.IsLocal || t.URI == "" || (t.Type != "" && t.Type != "track") {
			continue
		}
		if t.Explicit && !expliciet {
			verborgen++
			continue
		}
		uit = append(uit, item{id: t.URI, url: t.URI, titel: t.Name, sub: t.artiesten(), info: klok(float64(t.DurationMS) / 1000),
			duur: float64(t.DurationMS) / 1000, soort: soortSpNummer})
	}
	return uit, verborgen, nil
}

/* ---------------- afspelen op het eigen apparaat ---------------- */

func (s *Spotify) zoekApparaat() (string, error) {
	var r struct {
		Devices []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"devices"`
	}
	if _, err := s.doe("GET", "/me/player/devices", nil, &r); err != nil {
		return "", err
	}
	for _, d := range r.Devices {
		if d.Name == s.apparaat {
			s.mu.Lock()
			s.apparaatID = d.ID
			s.mu.Unlock()
			return d.ID, nil
		}
	}
	return "", errGeenApparaat
}

// Speel speelt de lijst nummers vanaf het eerste, zonder shuffle, herhalen of autoplay.
func (s *Spotify) Speel(uris []string) error {
	var id string
	var err error
	for i := 0; i < 20; i++ { // librespot heeft na het starten even nodig om zich aan te melden
		if id, err = s.zoekApparaat(); err == nil {
			break
		}
		if !errors.Is(err, errGeenApparaat) {
			return err
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return err
	}
	if len(uris) > 100 {
		uris = uris[:100]
	}
	q := "?device_id=" + url.QueryEscape(id)
	if _, err := s.doe("PUT", "/me/player/play"+q, map[string]any{"uris": uris}, nil); err != nil {
		return err
	}
	_, _ = s.doe("PUT", "/me/player/shuffle"+q+"&state=false", nil, nil)
	_, _ = s.doe("PUT", "/me/player/repeat"+q+"&state=off", nil, nil)
	return nil
}

func (s *Spotify) apparaatQ() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.apparaatID == "" {
		return ""
	}
	return "?device_id=" + url.QueryEscape(s.apparaatID)
}

func (s *Spotify) Pauze(aan bool) error {
	if aan {
		_, err := s.doe("PUT", "/me/player/pause"+s.apparaatQ(), nil, nil)
		return err
	}
	_, err := s.doe("PUT", "/me/player/play"+s.apparaatQ(), nil, nil)
	return err
}

func (s *Spotify) Volgende() { _, _ = s.doe("POST", "/me/player/next"+s.apparaatQ(), nil, nil) }
func (s *Spotify) Vorige()   { _, _ = s.doe("POST", "/me/player/previous"+s.apparaatQ(), nil, nil) }
func (s *Spotify) Spoel(naar float64) {
	q := s.apparaatQ()
	sep := "?"
	if q != "" {
		sep = "&"
	}
	_, _ = s.doe("PUT", fmt.Sprintf("/me/player/seek%s%sposition_ms=%d", q, sep, int(max(naar, 0)*1000)), nil, nil)
}

// Stand leest wat er nu speelt. Speelt er iets dat niet in de lijst staat (autoplay van
// Spotify, of iemand anders die het apparaat overneemt), dan pauzeert de app en telt het als klaar.
func (s *Spotify) Stand(lijst []item) (SpelerStand, error) {
	var r struct {
		Device struct {
			Name string `json:"name"`
		} `json:"device"`
		IsPlaying  bool `json:"is_playing"`
		ProgressMS int  `json:"progress_ms"`
		Item       *struct {
			URI        string `json:"uri"`
			Name       string `json:"name"`
			DurationMS int    `json:"duration_ms"`
			LinkedFrom struct {
				URI string `json:"uri"`
			} `json:"linked_from"`
		} `json:"item"`
	}
	code, err := s.doe("GET", "/me/player", nil, &r)
	if err != nil {
		return SpelerStand{}, err
	}
	if code == http.StatusNoContent || r.Item == nil || r.Device.Name != s.apparaat {
		return SpelerStand{Leeg: true}, nil
	}
	st := SpelerStand{Pauze: !r.IsPlaying, Pos: float64(r.ProgressMS) / 1000, Duur: float64(r.Item.DurationMS) / 1000, Index: -1}
	// Spotify speelt soms een andere uitgave van hetzelfde nummer (relinking): dan staat het
	// gevraagde nummer in linked_from, of heeft het in elk geval dezelfde naam
	for i, it := range lijst {
		if it.url == r.Item.URI || (r.Item.LinkedFrom.URI != "" && it.url == r.Item.LinkedFrom.URI) {
			st.Index = i
			break
		}
	}
	if st.Index < 0 {
		for i, it := range lijst {
			if strings.EqualFold(it.titel, r.Item.Name) {
				st.Index = i
				break
			}
		}
	}
	if st.Index < 0 {
		if r.IsPlaying {
			_ = s.Pauze(true)
		}
		return SpelerStand{Leeg: true}, nil
	}
	// laatste nummer afgelopen: Spotify blijft op het einde staan
	if !r.IsPlaying && st.Index == len(lijst)-1 && (r.ProgressMS == 0 || st.Duur-st.Pos < 2) {
		st.Leeg = true
	}
	return st, nil
}

/* ---------------- librespot ---------------- */

// Librespot start de Spotify Connect-speler als hij nodig is. Hij logt in met de gegevens
// die een ouder eenmalig heeft gekoppeld (sudo unscroll-ouder), opgeslagen in de cache.
type Librespot struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	naam string
}

func librespotCache() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "unscroll", "librespot")
}

func (l *Librespot) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cmd != nil {
		return nil
	}
	pad, err := exec.LookPath("librespot")
	if err != nil {
		return errGeenLibrespot
	}
	if _, err := os.Stat(filepath.Join(librespotCache(), "credentials.json")); err != nil {
		return errNietGekoppeld
	}
	cmd := exec.Command(pad, "--name", l.naam, "--cache", librespotCache(), "--disable-audio-cache",
		"--disable-discovery", "--backend", "alsa", "--bitrate", "160", "--autoplay", "off",
		"--device-type", "computer", "--volume-ctrl", "fixed", "--initial-volume", "100", "--quiet")
	stopMetOuder(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	l.cmd = cmd
	go func() {
		_ = cmd.Wait()
		l.mu.Lock()
		if l.cmd == cmd {
			l.cmd = nil
		}
		l.mu.Unlock()
	}()
	return nil
}

func (l *Librespot) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cmd != nil && l.cmd.Process != nil {
		_ = l.cmd.Process.Kill()
	}
	l.cmd = nil
}

/* ---------------- eenmalig inloggen (ouderhulp) ---------------- */

// spotifyInloggen is "unscroll --spotify-login CLIENT_ID": een ouder opent de link op een
// telefoon of computer, logt in, en plakt het adres waar Spotify naartoe stuurt hier terug.
// Alleen de refresh token gaat naar stdout, zodat de ouderhulp hem kan bewaren.
func spotifyInloggen(klantID string, in io.Reader, uit io.Writer) error {
	return spotifyInloggenBij(spotifyAccounts, klantID, in, uit, os.Stderr)
}

func spotifyInloggenBij(accounts, klantID string, in io.Reader, uit, meld io.Writer) error {
	b := make([]byte, 48)
	_, _ = rand.Read(b)
	verifier := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	_, _ = rand.Read(b[:12])
	state := base64.RawURLEncoding.EncodeToString(b[:12])
	q := url.Values{"client_id": {klantID}, "response_type": {"code"}, "redirect_uri": {spotifyRedirect},
		"code_challenge_method": {"S256"}, "code_challenge": {challenge}, "scope": {spotifyScopes}, "state": {state}}
	fmt.Fprintf(meld, "\n1. Open this link on a phone or computer and log in with the child's Spotify account:\n\n%s/authorize?%s\n\n", accounts, q.Encode())
	fmt.Fprintf(meld, "2. Spotify then sends you to a page that does not load (127.0.0.1). That is expected.\n   Copy the whole address from the address bar and paste it here:\n> ")
	regel, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && regel == "" {
		return err
	}
	terug, err := url.Parse(strings.TrimSpace(regel))
	if err != nil {
		return err
	}
	if e := terug.Query().Get("error"); e != "" {
		return fmt.Errorf("spotify: %s", e)
	}
	if terug.Query().Get("state") != state {
		return errors.New("this address does not belong to this login, try again")
	}
	code := terug.Query().Get("code")
	if code == "" {
		return errors.New("no code found in the address")
	}
	resp, err := http.PostForm(accounts+"/api/token", url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {spotifyRedirect}, "client_id": {klantID}, "code_verifier": {verifier}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var t struct {
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error_description"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&t)
	if resp.StatusCode != http.StatusOK || t.RefreshToken == "" {
		return fmt.Errorf("spotify: %s %s", resp.Status, t.Error)
	}
	fmt.Fprintln(uit, t.RefreshToken)
	return nil
}
