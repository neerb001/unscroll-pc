package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Navidrome spreekt de Subsonic-API. Het wachtwoord gaat nooit zelf over het netwerk:
// elk verzoek krijgt een nieuw zoutje en md5(wachtwoord+zout).
type Navidrome struct {
	URL, Gebruiker, Wachtwoord string
	http                       *http.Client
}

func nieuweNavidrome(g Geheimen) *Navidrome {
	n := &Navidrome{URL: adres(g["NAVIDROME_URL"]), Gebruiker: g["NAVIDROME_GEBRUIKER"], Wachtwoord: g["NAVIDROME_WACHTWOORD"]}
	if n.URL == "" || n.Gebruiker == "" {
		return nil
	}
	n.http = &http.Client{Timeout: 10 * time.Second}
	return n
}

func (n *Navidrome) adres(methode string, extra url.Values) string {
	zout := make([]byte, 6)
	_, _ = rand.Read(zout)
	s := hex.EncodeToString(zout)
	h := md5.Sum([]byte(n.Wachtwoord + s))
	q := url.Values{"u": {n.Gebruiker}, "t": {hex.EncodeToString(h[:])}, "s": {s}, "v": {"1.16.1"}, "c": {"unscroll"}, "f": {"json"}}
	for k, v := range extra {
		q[k] = v
	}
	return n.URL + "/rest/" + methode + "?" + q.Encode()
}

func (n *Navidrome) vraag(methode string, extra url.Values, doel any) error {
	resp, err := n.http.Get(n.adres(methode, extra))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("navidrome: %s", resp.Status)
	}
	var antwoord struct {
		R json.RawMessage `json:"subsonic-response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&antwoord); err != nil {
		return err
	}
	var status struct {
		Status string `json:"status"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(antwoord.R, &status)
	if status.Status != "ok" {
		return errors.New("navidrome: " + status.Error.Message)
	}
	return json.Unmarshal(antwoord.R, doel)
}

func (n *Navidrome) Artiesten() ([]item, error) {
	var r struct {
		Artists struct {
			Index []struct {
				Artist []struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					AlbumCount int    `json:"albumCount"`
				} `json:"artist"`
			} `json:"index"`
		} `json:"artists"`
	}
	if err := n.vraag("getArtists", nil, &r); err != nil {
		return nil, err
	}
	var uit []item
	for _, ix := range r.Artists.Index {
		for _, a := range ix.Artist {
			uit = append(uit, item{id: a.ID, titel: a.Name, aantal: a.AlbumCount, soort: soortArtiest})
		}
	}
	sort.SliceStable(uit, func(i, j int) bool { return strings.ToLower(uit[i].titel) < strings.ToLower(uit[j].titel) })
	return uit, nil
}

func (n *Navidrome) Albums(artiest string) ([]item, error) {
	var r struct {
		Artist struct {
			Album []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Year      int    `json:"year"`
				SongCount int    `json:"songCount"`
			} `json:"album"`
		} `json:"artist"`
	}
	if err := n.vraag("getArtist", url.Values{"id": {artiest}}, &r); err != nil {
		return nil, err
	}
	var uit []item
	for _, a := range r.Artist.Album {
		it := item{id: a.ID, titel: a.Name, aantal: a.SongCount, soort: soortAlbum}
		if a.Year > 0 {
			it.info = fmt.Sprint(a.Year)
		}
		uit = append(uit, it)
	}
	return uit, nil
}

func (n *Navidrome) Nummers(album string) ([]item, error) {
	var r struct {
		Album struct {
			Name string `json:"name"`
			Song []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Artist   string `json:"artist"`
				Track    int    `json:"track"`
				Duration int    `json:"duration"`
			} `json:"song"`
		} `json:"album"`
	}
	if err := n.vraag("getAlbum", url.Values{"id": {album}}, &r); err != nil {
		return nil, err
	}
	var uit []item
	for _, s := range r.Album.Song {
		uit = append(uit, item{id: s.ID, titel: s.Title, info: klok(float64(s.Duration)), sub: s.Artist + " · " + r.Album.Name,
			duur: float64(s.Duration), soort: soortNummer, url: n.adres("stream", url.Values{"id": {s.ID}})})
	}
	return uit, nil
}

// klok zet seconden om naar 3:05 of 1:02:03.
func klok(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	s := int(sec)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
