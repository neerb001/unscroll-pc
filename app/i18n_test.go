package main

import (
	"sort"
	"testing"
)

// Elke taal moet precies dezelfde sleutels hebben als Engels.
func TestVertalingenCompleet(t *testing.T) {
	for _, taal := range talen {
		if _, ok := taalNamen[taal]; !ok {
			t.Errorf("taal %q heeft geen naam in taalNamen", taal)
		}
		for k := range teksten["en"] {
			if _, ok := teksten[taal][k]; !ok {
				t.Errorf("%s mist sleutel %q", taal, k)
			}
		}
		var extra []string
		for k := range teksten[taal] {
			if _, ok := teksten["en"][k]; !ok {
				extra = append(extra, k)
			}
		}
		sort.Strings(extra)
		if len(extra) > 0 {
			t.Errorf("%s heeft sleutels die Engels niet heeft: %v", taal, extra)
		}
	}
	for _, n := range themaVolgorde {
		if tr("en", "theme."+n) == "theme."+n {
			t.Errorf("thema %q heeft geen vertaling", n)
		}
	}
}

func TestStandaardTaalIsEngels(t *testing.T) {
	if standaardTaal() != "en" {
		t.Error("standaardtaal moet Engels zijn")
	}
	if tr("xx", "yes") != "yes" {
		t.Error("onbekende taal moet terugvallen op Engels")
	}
}
