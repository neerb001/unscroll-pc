package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Nu bijwerken vanuit de ouderstand (U), zonder terminal. De updater moet als root draaien; de app
// draait als het kind. Net als bij de ouderinstellingen mag de app daarom precies één vaste
// opdracht als root uitvoeren (een sudo-regel): "unscroll --ouder-update". Die controleert zelf
// eerst de pincode en start dan hetzelfde script als de timer, met --nu.

var updaterPad = "/usr/local/sbin/unscroll-update"

// ouderUpdate draait als root. De pincode komt via stdin, zodat hij niet in de proceslijst staat.
func ouderUpdate(in io.Reader, uit io.Writer) error {
	var v struct {
		Pin string `json:"pin"`
	}
	if err := json.NewDecoder(io.LimitReader(in, 4<<10)).Decode(&v); err != nil {
		return err
	}
	o, err := laadOuders()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !o.pinKlopt(v.Pin) {
		return errors.New("wrong PIN")
	}
	cmd := exec.Command(updaterPad, "--nu")
	cmd.Stdout, cmd.Stderr = uit, uit
	return cmd.Run()
}

// ouderUpdateUitvoeren vraagt de root-helper om nu bij te werken (via sudo, zonder wachtwoord).
var ouderUpdateUitvoeren = func(verzoek []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sudo", "-n", "/usr/local/bin/unscroll", "--ouder-update")
	cmd.Stdin = bytes.NewReader(verzoek)
	return cmd.CombinedOutput()
}

type updateMsg struct {
	uit string
	err error
}

func (m *model) nuBijwerken() tea.Cmd {
	if m.bijwerken {
		return nil
	}
	m.bijwerken = true
	m.zegLang(m.t("msg.updating"))
	data, _ := json.Marshal(struct {
		Pin string `json:"pin"`
	}{m.ouderPin})
	return func() tea.Msg {
		uit, err := ouderUpdateUitvoeren(data)
		return updateMsg{strings.TrimSpace(string(uit)), err}
	}
}

// updateBericht vertelt de ouder hoe het ging, in gewone woorden.
func (m *model) updateBericht(msg updateMsg) {
	m.bijwerken = false
	m.nieuw = nieuweVersieKlaar()
	laatste := msg.uit
	if i := strings.LastIndex(laatste, "\n"); i >= 0 {
		laatste = laatste[i+1:]
	}
	switch {
	case m.nieuw != "":
		m.zegLang(m.t("msg.updateready", m.nieuw))
	case strings.Contains(msg.uit, "wrong PIN"):
		m.zeg(m.t("msg.badpin"))
	case msg.err != nil && (strings.Contains(msg.uit, "password") || strings.Contains(msg.uit, "not allowed") || strings.Contains(msg.uit, "sudoers")):
		// de sudo-regel komt met de updater zelf; op een apparaat dat net is bijgewerkt kan hij nog ontbreken
		m.zegLang(m.t("msg.updatenosudo"))
	case strings.Contains(msg.uit, "GitHub not reachable"):
		m.zegLang(m.t("msg.updateoffline"))
	case msg.err != nil:
		if laatste == "" {
			laatste = msg.err.Error()
		}
		m.zegLang(m.t("msg.updatefail", laatste))
	case strings.Contains(msg.uit, "Already up to date"):
		m.zeg(m.t("msg.uptodate", versie))
	default:
		m.zegLang(laatste)
	}
}

// zegLang: een melding die wat langer blijft staan, omdat er iets te lezen of te doen is.
func (m *model) zegLang(s string) { m.melding = s; m.meldingTot = time.Now().Add(15 * time.Second) }
