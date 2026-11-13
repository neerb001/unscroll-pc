package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func maakEpub(t *testing.T, hoofdstukken int) []byte {
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	schrijf := func(naam, inhoud string) {
		w, _ := z.Create(naam)
		_, _ = io.WriteString(w, inhoud)
	}
	schrijf("mimetype", "application/epub+zip")
	schrijf("META-INF/container.xml", `<?xml version="1.0"?><container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`)
	var manifest, spine string
	for i := 1; i <= hoofdstukken; i++ {
		manifest += fmt.Sprintf(`<item id="h%d" href="tekst/h%d.xhtml" media-type="application/xhtml+xml"/>`, i, i)
		spine += fmt.Sprintf(`<itemref idref="h%d"/>`, i)
		var p strings.Builder
		for j := 0; j < 30; j++ {
			fmt.Fprintf(&p, "<p>Hoofdstuk %d, alinea %d: Pluk woonde in een torenkamer &amp; had een rode takel.</p>", i, j)
		}
		schrijf(fmt.Sprintf("OEBPS/tekst/h%d.xhtml", i), fmt.Sprintf(`<html><head><style>p{}</style></head><body><h1>Hoofdstuk %d</h1>%s</body></html>`, i, p.String()))
	}
	schrijf("OEBPS/content.opf", `<package><manifest>`+manifest+`</manifest><spine>`+spine+`</spine></package>`)
	_ = z.Close()
	return buf.Bytes()
}

func TestLeesEpub(t *testing.T) {
	al, err := leesEpub(maakEpub(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	if len(al) != 62 || !al[0].kop || al[0].tekst != "Hoofdstuk 1" || !strings.Contains(al[1].tekst, "torenkamer & had") || strings.Contains(al[1].tekst, "p{}") {
		t.Fatalf("epub verkeerd gelezen: %d alinea's, eerste %+v, tweede %+v", len(al), al[0], al[1])
	}
}

func TestLezen(t *testing.T) {
	epub := maakEpub(t, 3)
	var mu sync.Mutex
	var voortgang []float64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/libraries":
			_, _ = w.Write([]byte(`{"libraries":[{"id":"lb","mediaType":"book"}]}`))
		case r.URL.Path == "/api/me":
			_, _ = w.Write([]byte(`{"mediaProgress":[{"libraryItemId":"e1","ebookProgress":0.25}]}`))
		case r.URL.Path == "/api/libraries/lb/items":
			_, _ = w.Write([]byte(`{"results":[{"id":"e1","media":{"metadata":{"title":"Pluk van de Petteflet","authorName":"Annie M.G. Schmidt"},"ebookFormat":"epub"}},{"id":"a1","media":{"metadata":{"title":"Alleen audio"},"duration":100}}]}`))
		case r.URL.Path == "/api/items/e1/ebook":
			_, _ = w.Write(epub)
		case r.URL.Path == "/api/me/progress/e1" && r.Method == "PATCH":
			var v map[string]float64
			_ = json.NewDecoder(r.Body).Decode(&v)
			mu.Lock()
			voortgang = append(voortgang, v["ebookProgress"])
			mu.Unlock()
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{"ABS_URL": srv.URL, "ABS_TOKEN": "x"})
	m.l = nieuwLuister(Geheimen{"ABS_URL": srv.URL, "ABS_TOKEN": "x"})
	m = toets(m, "2")
	v := m.View()
	if !strings.Contains(v, "Pluk van de Petteflet") || strings.Contains(v, "Alleen audio") || !strings.Contains(v, "25%") {
		t.Fatalf("boekenlijst:\n%s", v)
	}
	m = toets(m, "enter")
	if m.ls.boek == nil || !strings.Contains(m.View(), "Hoofdstuk 1") {
		t.Fatalf("boek niet open:\n%s", m.View())
	}
	for i := 0; i < 5; i++ {
		m = toets(m, "right")
	}
	if strings.Contains(m.View(), "Hoofdstuk 1, alinea 0:") {
		t.Error("bladeren werkt niet")
	}
	plek := m.ls.boek.alineaVanRegel(m.ls.boek.bovenste)
	m = toets(m, "esc")
	mu.Lock()
	if len(voortgang) == 0 || voortgang[len(voortgang)-1] <= 0 {
		t.Errorf("voortgang niet naar Audiobookshelf: %v", voortgang)
	}
	mu.Unlock()
	// opnieuw openen: zelfde plek, en zonder server (offline) gaat het ook
	srv.Close()
	m.ls = &leesStaat{}
	m = toets(m, "1")
	m = toets(m, "2")
	m = toets(m, "enter")
	if m.ls.boek == nil || m.ls.boek.alineaVanRegel(m.ls.boek.bovenste) != plek {
		t.Fatalf("plek niet onthouden of offline niet te lezen:\n%s", m.View())
	}
}

func TestLezenZoeken(t *testing.T) {
	epub := maakEpub(t, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/libraries":
			_, _ = w.Write([]byte(`{"libraries":[{"id":"lb","mediaType":"book"}]}`))
		case r.URL.Path == "/api/me":
			_, _ = w.Write([]byte(`{"mediaProgress":[]}`))
		case r.URL.Path == "/api/libraries/lb/items":
			_, _ = w.Write([]byte(`{"results":[
				{"id":"e1","media":{"metadata":{"title":"Pluk van de Petteflet","authorName":"Annie M.G. Schmidt"},"ebookFormat":"epub"}},
				{"id":"e2","media":{"metadata":{"title":"Harry Potter en de Steen der Wijzen","authorName":"J.K. Rowling"},"ebookFormat":"epub"}},
				{"id":"e3","media":{"metadata":{"title":"Éloïse en de draak","authorName":"Thé Tjong-Khing"},"ebookFormat":"epub"}},
				{"id":"e4","media":{"metadata":{"title":"Harry Potter en de Geheime Kamer","authorName":"J.K. Rowling"},"ebookFormat":"epub"}}]}`))
		case strings.HasSuffix(r.URL.Path, "/ebook"):
			_, _ = w.Write(epub)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{"ABS_URL": srv.URL, "ABS_TOKEN": "x"})
	m.l = nieuwLuister(Geheimen{"ABS_URL": srv.URL, "ABS_TOKEN": "x"})
	m.inst.Taal = "nl"
	m = toets(m, "2")
	typ := func(m model, s string) model {
		for _, r := range s {
			m = toets(m, string(r))
		}
		return m
	}
	m = toets(m, "/")
	if !m.ls.zoeken {
		t.Fatal("/ opent het zoekveld niet")
	}
	m = typ(m, "harry steen")
	v := m.View()
	if !strings.Contains(v, "Steen der Wijzen") || strings.Contains(v, "Geheime Kamer") || strings.Contains(v, "Pluk") || !strings.Contains(v, "1 van 4 boeken") {
		t.Fatalf("zoeken op twee woorden:\n%s", v)
	}
	// zoeken zonder accenten en op schrijver
	for i := 0; i < len("harry steen"); i++ {
		m = toets(m, "backspace")
	}
	m = typ(m, "eloise")
	if v := m.View(); !strings.Contains(v, "draak") || strings.Contains(v, "Harry") {
		t.Errorf("zoeken zonder accenten:\n%s", v)
	}
	for i := 0; i < len("eloise"); i++ {
		m = toets(m, "backspace")
	}
	m = typ(m, "rowling")
	if v := m.View(); !strings.Contains(v, "2 van 4 boeken") {
		t.Errorf("zoeken op schrijver:\n%s", v)
	}
	// letters als j en k zijn gewoon letters in het zoekveld; pijltjes kiezen
	m = toets(m, "down")
	m = toets(m, "enter")
	// op alfabet: eerst Geheime Kamer, dan Steen der Wijzen
	if m.ls.boek == nil || m.ls.boek.id != "e2" {
		t.Fatalf("het tweede gevonden boek is niet geopend")
	}
	if m.ls.zoeken || m.ls.lijst.filter != "" {
		t.Error("na het openen hoort het zoeken klaar te zijn")
	}
	// esc wist het zoeken en laat de hele lijst weer zien
	m = toets(m, "esc") // boek dicht
	m = toets(m, "/")
	m = typ(m, "pluk")
	m = toets(m, "esc")
	if v := m.View(); m.ls.zoeken || !strings.Contains(v, "Pluk") || !strings.Contains(v, "Harry") {
		t.Errorf("esc geeft de hele lijst niet terug:\n%s", v)
	}
}
