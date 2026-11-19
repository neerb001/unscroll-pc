package main

import (
	"net/http"
	"net/http/httptest"
	"os"
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

// Missie en opnieuw zijn woorden die je typt; opnieuw vraagt eerst of je het zeker weet.
func TestLabMissieEnOpnieuw(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m = toets(m, "4")
	m = toets(m, "down") // computerlab
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "typ missie") || !strings.Contains(m.View(), "typ opnieuw") {
		t.Fatalf("de uitleg zegt niet dat je het woord moet typen:\n%s", m.View())
	}
	for _, s := range []string{"pwd", "ls"} {
		m = typ(m, s)
	}
	gedaan := m.od.lab.Missie
	if gedaan == 0 {
		t.Fatal("de eerste missies zouden gedaan moeten zijn")
	}
	m = typ(m, "missie")
	if !strings.Contains(strings.Join(m.od.labRegels, "\n"), m.od.lab.missieTekst("nl")) {
		t.Error("missie laat de opdracht niet nog eens zien")
	}
	m = typ(m, "opnieuw")
	if m.od.lab.Missie != gedaan || !strings.Contains(m.View(), "Weet je het zeker") {
		t.Fatalf("opnieuw wist meteen alles, zonder te vragen:\n%s", m.View())
	}
	m = typ(m, "ls") // iets anders: dan gaat het niet door
	m = typ(m, "opnieuw")
	if m.od.lab.Missie != gedaan {
		t.Fatal("na een andere opdracht hoort opnieuw weer eerst te vragen")
	}
	m = typ(m, "opnieuw")
	if m.od.lab.Missie != 0 {
		t.Errorf("twee keer opnieuw begint niet opnieuw: missie %d", m.od.lab.Missie)
	}
	// in het Engels heten ze mission en restart
	m.inst.Taal = "en"
	if !strings.Contains(m.View(), "type mission") || !strings.Contains(m.View(), "type restart") {
		t.Errorf("Engelse uitleg:\n%s", m.View())
	}
}

func TestDierVanDeDag(t *testing.T) {
	// het ophalen van de week loopt op de achtergrond door na de test: die map ruimen we zelf
	// op, zonder te klagen als er nog net een bestand bij komt
	cache, _ := os.MkdirTemp("", "dieren")
	t.Cleanup(func() { time.Sleep(200 * time.Millisecond); os.RemoveAll(cache) })
	t.Setenv("XDG_CACHE_HOME", cache)
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
