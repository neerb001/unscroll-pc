package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLabAlleMissies(t *testing.T) {
	w := nieuweLabWereld()
	stappen := []string{"pwd", "ls", "cd tuin", "cat briefje.txt", "cd schuur", "ls", "cat sleutel.txt",
		"mv sleutel.txt ~", "cd ..", "cd", "mkdir uitvindingen", "echo een robot die de hond uitlaat > uitvindingen/robot.txt",
		"cd zolder/doos", "cat kaart.txt", "rm ~/tuin/schuur/hark.txt"}
	for _, s := range stappen {
		uit, _ := w.voerUit(s, "nl")
		if strings.Contains(uit, "bestaat niet") || strings.Contains(uit, "ken ik niet") {
			t.Fatalf("%q gaf: %s", s, uit)
		}
	}
	if w.Missie != len(labMissies) {
		t.Errorf("na alle stappen missie %d van %d", w.Missie, len(labMissies))
	}
	if uit, _ := w.voerUit("cat /etc/passwd", "nl"); !strings.Contains(uit, "bestaat niet") {
		t.Errorf("het lab mag niets van het echte systeem laten zien: %s", uit)
	}
	if uit, _ := w.voerUit("sudo rm -rf /", "en"); !strings.Contains(uit, "unknown command") {
		t.Errorf("onbekende opdracht: %s", uit)
	}
}

func TestDierVanDeDag(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	eerste := dieren[dierVanDag(time.Now())]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/api.php"):
			_, _ = w.Write([]byte(`{"query":{"pages":[{"langlinks":[{"lang":"nl","title":"Nederlands Dier"}]}]}}`))
		case strings.Contains(r.URL.Path, "/en/"+strings.ReplaceAll(eerste, " ", "_")):
			w.WriteHeader(404) // dit dier bestaat niet meer: door naar het volgende
		case strings.HasPrefix(r.URL.Path, "/nl/"):
			w.WriteHeader(404) // Nederlandse pagina weg: terugvallen op Engels
		default:
			_, _ = w.Write([]byte(`{"type":"standard","title":"Hedgehog","description":"spiny mammal","extract":"A hedgehog is a spiny mammal."}`))
		}
	}))
	defer srv.Close()
	oud, oudAPI := wikiBasis, wikiAPI
	wikiBasis, wikiAPI = srv.URL+"/%s/", srv.URL+"/%s/w/api.php"
	defer func() { wikiBasis, wikiAPI = oud, oudAPI }()
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m = toets(m, "4")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "A hedgehog is a spiny mammal") {
		t.Fatalf("dier niet te zien:\n%s", m.View())
	}
	// afwisseling: in twee weken geen herhaling en geen lange reeks uit één groep van de lijst
	gezien := map[int]bool{}
	for d := 0; d < len(dieren); d++ {
		i := dierVanDag(time.Now().AddDate(0, 0, d))
		if gezien[i] {
			t.Fatalf("dier %d komt binnen %d dagen twee keer", i, len(dieren))
		}
		gezien[i] = true
		if d > 0 {
			vorige := dierVanDag(time.Now().AddDate(0, 0, d-1))
			if afstand := (i - vorige + len(dieren)) % len(dieren); afstand < 20 || afstand > len(dieren)-20 {
				t.Fatalf("twee dagen na elkaar liggen te dicht bij elkaar in de lijst: %d en %d", vorige, i)
			}
		}
	}
}

func TestNatuurdagboek(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m = toets(m, "4")
	m = toets(m, "down")
	m = toets(m, "down")
	m = toets(m, "enter")
	m = toets(m, "n")
	m = toets(m, "een buizerd boven het weiland")
	m = toets(m, "enter")
	if len(laadNotities()) != 1 || !strings.Contains(m.View(), "buizerd") {
		t.Fatalf("notitie niet bewaard:\n%s", m.View())
	}
}
