package main

import (
	"bufio"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Zelf ontdekken op Spotify (als de ouders "zoeken: true" zetten): artiesten en albums zoeken,
// alle albums van een artiest bekijken en favorieten bewaren. Nog steeds geen aanbevelingen,
// geen radio en geen autoplay; expliciete nummers en geblokkeerde artiesten blijven weg.
// Ouders zien in de ouderstand (w) wat er is gezocht en geluisterd.

type spFavoriet struct {
	ID    string `json:"id"` // "artist:ID" of "album:ID"
	Titel string `json:"titel"`
	Sub   string `json:"sub"`
}

func spFavorietenPad() string {
	return filepath.Join(filepath.Dir(instellingenPad()), "spotify-favorieten.json")
}

func laadSpFavorieten() []spFavoriet {
	var f []spFavoriet
	if data, err := os.ReadFile(spFavorietenPad()); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	return f
}

func bewaarSpFavorieten(f []spFavoriet) {
	_ = os.MkdirAll(filepath.Dir(spFavorietenPad()), 0o755)
	data, _ := json.MarshalIndent(f, "", "  ")
	_ = veiligSchrijven(spFavorietenPad(), data, 0o644)
}

func favorietItem(f spFavoriet) item {
	s := soortSpLijst
	if strings.HasPrefix(f.ID, "artist:") {
		s = soortSpArtiest
	}
	return item{id: f.ID, titel: "★ " + f.Titel, sub: f.Sub, soort: s}
}

// wisselFavoriet zet een artiest of album in of uit de favorieten; true als hij er nu in staat.
func wisselFavoriet(it item) bool {
	fav := laadSpFavorieten()
	for i, f := range fav {
		if f.ID == it.id {
			bewaarSpFavorieten(append(fav[:i], fav[i+1:]...))
			return false
		}
	}
	bewaarSpFavorieten(append(fav, spFavoriet{ID: it.id, Titel: strings.TrimPrefix(it.titel, "★ "), Sub: it.sub}))
	return true
}

/* ---------------- blokkeerlijst ---------------- */

// geblokkeerd: Spotify-links van artiesten of albums, of gewoon de naam van een artiest.
type blokkeer struct{ ids, namen map[string]bool }

func nieuweBlokkeer(lijst []string) blokkeer {
	b := blokkeer{map[string]bool{}, map[string]bool{}}
	for _, x := range lijst {
		x = strings.TrimSpace(x)
		if soort, id, ok := spotifyID(x); ok {
			b.ids[soort+":"+id] = true
		} else if x != "" {
			b.namen[strings.ToLower(x)] = true
		}
	}
	return b
}

func (b blokkeer) artiest(id, naam string) bool {
	return b.ids["artist:"+id] || b.namen[strings.ToLower(naam)]
}

// spotifyID is als spotifyLink, maar kent ook artiesten.
func spotifyID(s string) (soort, id string, ok bool) {
	if soort, id, ok = spotifyLink(s); ok {
		return
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "spotify:artist:") {
		return "artist", strings.TrimPrefix(s, "spotify:artist:"), true
	}
	if u, err := url.Parse(s); err == nil && strings.HasSuffix(u.Host, "spotify.com") {
		d := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(d) >= 2 && d[len(d)-2] == "artist" {
			return "artist", d[len(d)-1], true
		}
	}
	return "", "", false
}

/* ---------------- Web API ---------------- */

type spAlbum struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ReleaseDate string `json:"release_date"`
	Artists     []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artists"`
}

func (a spAlbum) item(b blokkeer) (item, bool) {
	sub := ""
	for i, x := range a.Artists {
		if b.artiest(x.ID, x.Name) {
			return item{}, false
		}
		if i == 0 {
			sub = x.Name
		}
	}
	if b.ids["album:"+a.ID] {
		return item{}, false
	}
	jaar := a.ReleaseDate
	if len(jaar) > 4 {
		jaar = jaar[:4]
	}
	return item{id: "album:" + a.ID, titel: a.Name, sub: sub, info: jaar, soort: soortSpLijst}, true
}

// Zoek zoekt artiesten en albums. Geen nummers of playlists: die leiden naar mixen van Spotify.
func (s *Spotify) Zoek(vraag string, b blokkeer) ([]item, error) {
	var r struct {
		Artists struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"artists"`
		Albums struct {
			Items []spAlbum `json:"items"`
		} `json:"albums"`
	}
	q := url.Values{"q": {vraag}, "type": {"artist,album"}}
	if _, err := s.doe("GET", "/search?"+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	var uit []item
	for _, a := range r.Artists.Items {
		if !b.artiest(a.ID, a.Name) {
			uit = append(uit, item{id: "artist:" + a.ID, titel: a.Name, soort: soortSpArtiest})
		}
	}
	for _, a := range r.Albums.Items {
		if it, ok := a.item(b); ok {
			uit = append(uit, it)
		}
	}
	return uit, nil
}

// AlbumsVan geeft de albums en singles van een artiest, nieuwste eerst (zoals Spotify ze geeft).
func (s *Spotify) AlbumsVan(artiest string, b blokkeer) ([]item, error) {
	// alleen albums en hooguit twee pagina's: grote artiesten hebben honderden releases,
	// en al die pagina's ophalen loopt tegen de grens van Spotify aan
	pad := "/artists/" + artiest + "/albums?include_groups=album"
	var uit []item
	for pagina := 0; pad != "" && pagina < 2; pagina++ {
		var r struct {
			Items []spAlbum `json:"items"`
			Next  string    `json:"next"`
		}
		if _, err := s.doe("GET", pad, nil, &r); err != nil {
			return nil, err
		}
		for _, a := range r.Items {
			if it, ok := a.item(b); ok {
				uit = append(uit, it)
			}
		}
		pad = strings.TrimPrefix(r.Next, s.api)
	}
	return uit, nil
}

/* ---------------- logboek voor de ouders ---------------- */

type spLogRegel struct {
	Tijd  time.Time `json:"tijd"`
	Soort string    `json:"soort"` // zoek, luister
	Tekst string    `json:"tekst"`
}

func spLogPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "spotify-log.jsonl") }

func spLog(soort, tekst string) {
	_ = os.MkdirAll(filepath.Dir(spLogPad()), 0o755)
	f, err := os.OpenFile(spLogPad(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	data, _ := json.Marshal(spLogRegel{time.Now(), soort, tekst})
	_, _ = f.Write(append(data, '\n'))
}

// spWeek geeft het logboek van de afgelopen week als lijst (nieuwste bovenaan).
func spWeek(zoekLabel, luisterLabel string) []item {
	f, err := os.Open(spLogPad())
	if err != nil {
		return nil
	}
	defer f.Close()
	vanaf := time.Now().AddDate(0, 0, -7)
	var uit []item
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r spLogRegel
		if json.Unmarshal(sc.Bytes(), &r) != nil || r.Tijd.Before(vanaf) {
			continue
		}
		label := luisterLabel
		if r.Soort == "zoek" {
			label = zoekLabel
		}
		uit = append([]item{{titel: label + ": " + r.Tekst, info: r.Tijd.Format("Mon 2 Jan 15:04")}}, uit...)
	}
	return uit
}
