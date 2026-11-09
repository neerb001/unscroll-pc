package main

// Zender is een radiozender. Ouders kunnen hun eigen lijst zetten onder "radio:" in ouders.yaml;
// staat daar niets, dan gebruikt de app deze lijst.
type Zender struct {
	Naam string `yaml:"naam"`
	URL  string `yaml:"url"`
}

var standaardZenders = []Zender{
	{"NPO Radio 1", "https://icecast.omroep.nl/radio1-bb-mp3"},
	{"NPO Radio 2", "https://icecast.omroep.nl/radio2-bb-mp3"},
	{"NPO 3FM", "https://icecast.omroep.nl/3fm-bb-mp3"},
	{"NPO Klassiek", "https://icecast.omroep.nl/radio4-bb-mp3"},
	{"BBC World Service", "https://stream.live.vc.bbcmedia.co.uk/bbc_world_service"},
}

func (o Ouders) zenders() []Zender {
	var z []Zender
	for _, x := range o.Radio {
		if x.Naam != "" && x.URL != "" {
			z = append(z, x)
		}
	}
	if len(z) == 0 {
		return standaardZenders
	}
	return z
}
