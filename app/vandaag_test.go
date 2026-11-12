package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPasen(t *testing.T) {
	for jaar, verwacht := range map[int]string{2024: "2024-03-31", 2025: "2025-04-20", 2026: "2026-04-05", 2027: "2027-03-28", 2038: "2038-04-25"} {
		if got := pasen(jaar).Format("2006-01-02"); got != verwacht {
			t.Errorf("Pasen %d: %s, verwacht %s", jaar, got, verwacht)
		}
	}
}

func TestMaanEnSeizoen(t *testing.T) {
	if f := maanFase(time.Date(2024, 4, 8, 18, 0, 0, 0, time.UTC)); f != 0 {
		t.Errorf("8 april 2024 (zonsverduistering) hoort nieuwe maan te zijn, fase %d", f)
	}
	if f := maanFase(time.Date(2024, 4, 23, 23, 0, 0, 0, time.UTC)); f != 4 {
		t.Errorf("23 april 2024 hoort volle maan te zijn, fase %d", f)
	}
	if nu, volgend, _ := seizoen(time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)); nu != "autumn" || volgend != "winter" {
		t.Errorf("november: %s → %s", nu, volgend)
	}
	if nu, _, begin := seizoen(time.Date(2026, 12, 25, 12, 0, 0, 0, time.Local)); nu != "winter" || begin.Month() != time.March || begin.Year() != 2027 {
		t.Errorf("kerst: %s, volgend begin %v", nu, begin)
	}
}

func TestJaarfeesten(t *testing.T) {
	f := volgendeFeesten(time.Date(2026, 11, 11, 15, 0, 0, 0, time.Local), 2)
	if f[0].sleutel != "martinmas" || dagenTot(time.Date(2026, 11, 11, 15, 0, 0, 0, time.Local), f[0].datum) != 0 || f[1].sleutel != "advent" {
		t.Errorf("11 november: %+v", f)
	}
	if a := jaarfeesten(2026)[8]; a.sleutel != "advent" || a.datum.Format("2006-01-02") != "2026-11-29" {
		t.Errorf("eerste advent 2026 hoort 29 november te zijn: %v", a.datum)
	}
	if f := volgendeFeesten(time.Date(2026, 12, 28, 0, 0, 0, 0, time.Local), 1); f[0].sleutel != "epiphany" || f[0].datum.Year() != 2027 {
		t.Errorf("na Kerst komt Driekoningen van volgend jaar: %+v", f)
	}
}

const nepWeer = `{"current":{"temperature_2m":4.2,"apparent_temperature":0.5,"weather_code":61,"wind_speed_10m":45,"is_day":1},
"hourly":{"time":["2026-11-12T13:00","2026-11-12T14:00","2026-11-12T15:00"],"temperature_2m":[4,4,3],"precipitation":[0,0.8,2.1],"precipitation_probability":[20,70,90]},
"daily":{"time":["2026-11-12","2026-11-13"],"weather_code":[61,3],"temperature_2m_max":[6,9],"temperature_2m_min":[2,4],"precipitation_sum":[3,0],
"precipitation_probability_max":[90,10],"sunrise":["2026-11-12T07:58","2026-11-13T08:00"],"sunset":["2026-11-12T16:52","2026-11-13T16:50"],"daylight_duration":[31940,31800]}}`

func TestVandaagScherm(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var vraag string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vraag = r.URL.RawQuery
		_, _ = w.Write([]byte(nepWeer))
	}))
	defer srv.Close()
	oud := weerBasis
	weerBasis = srv.URL
	defer func() { weerBasis = oud }()
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 110, 34
	m.ouders.Vandaag.Plaats, m.ouders.Vandaag.Breedtegraad, m.ouders.Vandaag.Lengtegraad = "Utrecht", 52.09, 5.12
	m = toets(m, "7")
	m.nu = time.Date(2026, 11, 12, 13, 30, 0, 0, time.Local)
	v := m.View()
	for _, moet := range []string{"Donderdag 12 november", "Utrecht", "4°", "regen", "Neem een regenjas mee", "muts en handschoenen",
		"pas op op de fiets", "zon op 07:58", "2 minuten korter licht", "Advent", "Sinterklaas", "Morgen:", "Open-Meteo"} {
		if !strings.Contains(v, moet) {
			t.Errorf("%q ontbreekt:\n%s", moet, v)
		}
	}
	if !strings.Contains(vraag, "latitude=52.09") {
		t.Errorf("verkeerde plek gevraagd: %s", vraag)
	}
	// zonder verbinding: het laatst opgehaalde weer
	srv.Close()
	m2, _ := testModel(t, Geheimen{})
	m2.breed, m2.hoog = 110, 34
	m2 = toets(m2, "7")
	if !strings.Contains(m2.View(), "4°") {
		t.Errorf("bewaard weer niet te zien zonder verbinding:\n%s", m2.View())
	}
}
