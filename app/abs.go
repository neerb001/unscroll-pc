package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Audiobookshelf: luisterboeken en podcasts met voortgang die meegaat naar andere apparaten.
// De sleutel (ABS_TOKEN) is die van het account van het kind, zodat zijn voortgang klopt.
type ABS struct {
	URL, Token string
	http       *http.Client
	apparaat   string
}

func nieuweABS(g Geheimen, apparaat string) *ABS {
	a := &ABS{URL: adres(g["ABS_URL"]), Token: g["ABS_TOKEN"], apparaat: apparaat}
	if a.URL == "" || a.Token == "" {
		return nil
	}
	a.http = &http.Client{Timeout: 10 * time.Second}
	return a
}

func (a *ABS) doe(methode, pad string, in, uit any) error {
	var body *bytes.Reader
	if in != nil {
		data, _ := json.Marshal(in)
		body = bytes.NewReader(data)
	} else {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(methode, a.URL+pad, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.Token)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("audiobookshelf: %s", resp.Status)
	}
	if uit == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(uit)
}

type absBibliotheek struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"mediaType"` // "book" of "podcast"
}

func (a *ABS) Bibliotheken() ([]absBibliotheek, error) {
	var r struct {
		Libraries []absBibliotheek `json:"libraries"`
	}
	err := a.doe("GET", "/api/libraries", nil, &r)
	return r.Libraries, err
}

type absVoortgang struct {
	LibraryItemID string  `json:"libraryItemId"`
	EpisodeID     string  `json:"episodeId"`
	Progress      float64 `json:"progress"`
	IsFinished    bool    `json:"isFinished"`
}

func (a *ABS) voortgang() map[string]absVoortgang {
	var r struct {
		MediaProgress []absVoortgang `json:"mediaProgress"`
	}
	v := map[string]absVoortgang{}
	if a.doe("GET", "/api/me", nil, &r) == nil {
		for _, p := range r.MediaProgress {
			v[p.LibraryItemID+"/"+p.EpisodeID] = p
		}
	}
	return v
}

// Items geeft alle boeken (of podcasts) uit de bibliotheken van één soort, op titel.
func (a *ABS) Items(mediaType string) ([]item, error) {
	libs, err := a.Bibliotheken()
	if err != nil {
		return nil, err
	}
	vg := a.voortgang()
	var uit []item
	for _, l := range libs {
		if l.MediaType != mediaType {
			continue
		}
		var r struct {
			Results []struct {
				ID    string `json:"id"`
				Media struct {
					Metadata struct {
						Title      string `json:"title"`
						AuthorName string `json:"authorName"`
						Author     string `json:"author"`
					} `json:"metadata"`
					Duration    float64 `json:"duration"`
					NumEpisodes int     `json:"numEpisodes"`
				} `json:"media"`
			} `json:"results"`
		}
		q := url.Values{"minified": {"1"}, "sort": {"media.metadata.title"}, "limit": {"1000"}}
		if err := a.doe("GET", "/api/libraries/"+l.ID+"/items?"+q.Encode(), nil, &r); err != nil {
			return nil, err
		}
		for _, x := range r.Results {
			md := x.Media.Metadata
			it := item{id: x.ID, titel: md.Title, sub: md.AuthorName, duur: x.Media.Duration, aantal: x.Media.NumEpisodes}
			if it.sub == "" {
				it.sub = md.Author
			}
			if mediaType == "podcast" {
				it.soort = soortPodcast
			} else {
				it.soort = soortBoek
				it.info = klok(x.Media.Duration)
				p := vg[x.ID+"/"]
				it.voortgang, it.klaar = p.Progress, p.IsFinished
			}
			uit = append(uit, it)
		}
	}
	sort.SliceStable(uit, func(i, j int) bool { return strings.ToLower(uit[i].titel) < strings.ToLower(uit[j].titel) })
	return uit, nil
}

// Afleveringen van één podcast, nieuwste eerst.
func (a *ABS) Afleveringen(podcast string) ([]item, error) {
	var r struct {
		Media struct {
			Metadata struct {
				Title string `json:"title"`
			} `json:"metadata"`
			Episodes []struct {
				ID          string  `json:"id"`
				Title       string  `json:"title"`
				PublishedAt int64   `json:"publishedAt"`
				Duration    float64 `json:"duration"`
				AudioFile   struct {
					Duration float64 `json:"duration"`
				} `json:"audioFile"`
			} `json:"episodes"`
		} `json:"media"`
	}
	if err := a.doe("GET", "/api/items/"+podcast+"?expanded=1", nil, &r); err != nil {
		return nil, err
	}
	vg := a.voortgang()
	eps := r.Media.Episodes
	sort.SliceStable(eps, func(i, j int) bool { return eps[i].PublishedAt > eps[j].PublishedAt })
	var uit []item
	for _, e := range eps {
		d := e.Duration
		if d == 0 {
			d = e.AudioFile.Duration
		}
		it := item{id: e.ID, ouder: podcast, titel: e.Title, sub: r.Media.Metadata.Title, duur: d, soort: soortAflevering}
		if e.PublishedAt > 0 {
			it.info = time.UnixMilli(e.PublishedAt).Format("2 Jan 2006")
		}
		p := vg[podcast+"/"+e.ID]
		it.voortgang, it.klaar = p.Progress, p.IsFinished
		uit = append(uit, it)
	}
	return uit, nil
}

type absSpoor struct {
	StartOffset float64 `json:"startOffset"`
	Duration    float64 `json:"duration"`
	ContentURL  string  `json:"contentUrl"`
}

type absSessie struct {
	ID            string     `json:"id"`
	CurrentTime   float64    `json:"currentTime"`
	Duration      float64    `json:"duration"`
	DisplayTitle  string     `json:"displayTitle"`
	DisplayAuthor string     `json:"displayAuthor"`
	AudioTracks   []absSpoor `json:"audioTracks"`
}

// Start opent een luistersessie. Audiobookshelf onthoudt waar je was.
func (a *ABS) Start(itemID, aflevering string) (absSessie, error) {
	pad := "/api/items/" + itemID + "/play"
	if aflevering != "" {
		pad += "/" + aflevering
	}
	in := map[string]any{
		"deviceInfo":         map[string]string{"clientName": "Unscroll", "deviceId": a.apparaat},
		"forceDirectPlay":    true,
		"forceTranscode":     false,
		"supportedMimeTypes": []string{"audio/mpeg", "audio/mp4", "audio/aac", "audio/flac", "audio/ogg", "audio/x-m4b", "audio/webm", "audio/wav"},
		"mediaPlayer":        "mpv",
	}
	var s absSessie
	err := a.doe("POST", pad, in, &s)
	return s, err
}

// SpoorURL maakt een volledig adres van een contentUrl uit de sessie.
func (a *ABS) SpoorURL(contentURL string) string {
	u := a.URL + contentURL
	if strings.Contains(u, "?") {
		return u + "&token=" + url.QueryEscape(a.Token)
	}
	return u + "?token=" + url.QueryEscape(a.Token)
}

func (a *ABS) Bij(sessie string, tijd, geluisterd, duur float64) error {
	return a.doe("POST", "/api/session/"+sessie+"/sync", map[string]float64{"currentTime": tijd, "timeListened": geluisterd, "duration": duur}, nil)
}

func (a *ABS) Sluit(sessie string, tijd, geluisterd, duur float64) error {
	return a.doe("POST", "/api/session/"+sessie+"/close", map[string]float64{"currentTime": tijd, "timeListened": geluisterd, "duration": duur}, nil)
}
