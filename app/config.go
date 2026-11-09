package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Instellingen kan het kind zelf aanpassen in het tabblad Systeem.
// Ze staan in ~/.config/eee/instellingen.json.
type Instellingen struct {
	Computer string `json:"computer"`
	Naam     string `json:"naam"`
	Maatje   string `json:"maatje"`
	Gezicht  int    `json:"gezicht"`
	Thema    string `json:"thema"`
}

func standaardInstellingen() Instellingen {
	return Instellingen{Computer: "EEE", Maatje: "Bliep", Thema: "nacht"}
}

func instellingenPad() string {
	if p := os.Getenv("EEE_INSTELLINGEN"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "eee", "instellingen.json")
}

func laadInstellingen() Instellingen {
	i := standaardInstellingen()
	data, err := os.ReadFile(instellingenPad())
	if err == nil {
		_ = json.Unmarshal(data, &i)
	}
	if strings.TrimSpace(i.Maatje) == "" {
		i.Maatje = "Bliep"
	}
	if _, ok := themas[i.Thema]; !ok {
		i.Thema = "nacht"
	}
	if i.Gezicht < 0 || i.Gezicht >= len(gezichten) {
		i.Gezicht = 0
	}
	return i
}

func (i Instellingen) bewaar() error {
	pad := instellingenPad()
	if err := os.MkdirAll(filepath.Dir(pad), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(i, "", "  ")
	tmp := pad + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, pad)
}

// Ouders bevat de instellingen van de ouders. Dit bestand is van root,
// het kind kan het lezen maar niet wijzigen: /etc/eee/ouders.yaml.
type Ouders struct {
	Avondslot struct {
		Schooldagen    string `yaml:"schooldagen"`
		Weekend        string `yaml:"weekend"`
		Open           string `yaml:"open"`
		LezenMag       bool   `yaml:"lezen_mag"`
		LuisterboekMag bool   `yaml:"luisterboek_mag"`
	} `yaml:"avondslot"`
	PinSHA256 string `yaml:"pin_sha256"`
	MaxVolume int    `yaml:"max_volume"`
	Ntfy      struct {
		URL   string `yaml:"url"`
		Topic string `yaml:"topic"`
	} `yaml:"ntfy"`
}

func standaardOuders() Ouders {
	var o Ouders
	o.Avondslot.Schooldagen = "20:00"
	o.Avondslot.Weekend = "20:30"
	o.Avondslot.Open = "07:00"
	o.Avondslot.LezenMag = true
	o.Avondslot.LuisterboekMag = true
	o.MaxVolume = 70
	o.Ntfy.URL = "https://ntfy.sh"
	return o
}

func oudersPad() string {
	if p := os.Getenv("EEE_OUDERS"); p != "" {
		return p
	}
	return "/etc/eee/ouders.yaml"
}

func laadOuders() (Ouders, error) {
	o := standaardOuders()
	data, err := os.ReadFile(oudersPad())
	if err != nil {
		return o, err
	}
	err = yaml.Unmarshal(data, &o)
	if o.MaxVolume <= 0 || o.MaxVolume > 100 {
		o.MaxVolume = 70
	}
	return o, err
}

func pinHash(pin string) string {
	h := sha256.Sum256([]byte("eee-kinderkamer:" + strings.TrimSpace(pin)))
	return hex.EncodeToString(h[:])
}

func (o Ouders) pinKlopt(pin string) bool {
	return o.PinSHA256 != "" && strings.EqualFold(o.PinSHA256, pinHash(pin))
}
