package main

import (
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Uitzetten en opnieuw opstarten vanuit Systeem: z en r, met een tweede druk om het zeker te
// weten. Eerst stopt de muziek netjes en staat alles wat bewaard moet worden op schijf.
//
// De app draait als het kind; het mag precies deze twee opdrachten als root (sudo-regel, zie
// unscroll-update). Lukt sudo niet, dan via logind (dat mag de gebruiker van het scherm vaak ook).

const (
	afsluitUit        = "poweroff"
	afsluitOpnieuw    = "reboot"
	afsluitBedenktijd = 10 * time.Second
)

var afsluitOpdracht = func(soort string) ([]byte, error) {
	uit, err := exec.Command("sudo", "-n", "/usr/bin/systemctl", soort).CombinedOutput()
	if err == nil {
		return nil, nil
	}
	if uit2, err2 := exec.Command("systemctl", soort).CombinedOutput(); err2 == nil {
		return nil, nil
	} else {
		uit = append(uit, uit2...)
	}
	return uit, err
}

type afsluitMsg struct {
	uit []byte
	err error
}

// vraagAfsluiten: de eerste keer vragen of het zeker is, de tweede keer (binnen tien seconden) doen.
func (m *model) vraagAfsluiten(soort string) tea.Cmd {
	if m.afsluiten == soort && time.Since(m.afsluitenGevraagd) < afsluitBedenktijd {
		m.afsluiten = ""
		if soort == afsluitUit {
			m.zegLang(m.t("sys.offnow"))
		} else {
			m.zegLang(m.t("sys.rebootnow"))
		}
		return tea.Sequence(m.stopHuidig(true), func() tea.Msg {
			wachtOpSchijf() // eerst alles bewaren
			uit, err := afsluitOpdracht(soort)
			return afsluitMsg{uit, err}
		})
	}
	m.afsluiten, m.afsluitenGevraagd = soort, time.Now()
	if soort == afsluitUit {
		m.zegLang(m.t("sys.offask"))
	} else {
		m.zegLang(m.t("sys.rebootask"))
	}
	return nil
}

func (m *model) afsluitBericht(msg afsluitMsg) {
	if msg.err == nil {
		return // de computer gaat uit
	}
	reden := strings.TrimSpace(string(msg.uit))
	if reden == "" {
		reden = msg.err.Error()
	}
	m.zegLang(m.t("sys.offfail", reden))
}
