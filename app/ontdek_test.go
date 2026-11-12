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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/nl/") {
			w.WriteHeader(404) // Nederlandse pagina bestaat niet: terugvallen op Engels
			return
		}
		_, _ = w.Write([]byte(`{"type":"standard","title":"Hedgehog","description":"spiny mammal","extract":"A hedgehog is a spiny mammal."}`))
	}))
	defer srv.Close()
	oud := wikiBasis
	wikiBasis = srv.URL + "/%s/"
	defer func() { wikiBasis = oud }()
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m = toets(m, "4")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "A hedgehog is a spiny mammal") {
		t.Fatalf("dier niet te zien:\n%s", m.View())
	}
	if dierVanDag(time.Now()) == dierVanDag(time.Now().AddDate(0, 0, 1)) {
		t.Error("elke dag hoort een ander dier te zijn")
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
