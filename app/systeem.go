package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Kleine hulpjes rond het apparaat. Alles mag mislukken zonder dat de app stopt.

func draai(timeout time.Duration, naam string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, naam, args...).Output()
	return strings.TrimSpace(string(out)), err
}

func wifiStatus(taal string) string {
	out, err := draai(2*time.Second, "nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL", "dev", "wifi")
	if err != nil {
		return tr(taal, "wifi.unknown")
	}
	for _, regel := range strings.Split(out, "\n") {
		delen := strings.Split(regel, ":")
		if len(delen) >= 3 && delen[0] == "yes" {
			return fmt.Sprintf("%s · %s%%", delen[1], delen[2])
		}
	}
	return tr(taal, "wifi.off")
}

var procent = regexp.MustCompile(`\[(\d+)%\]`)

func volume() int {
	out, err := draai(2*time.Second, "amixer", "get", "Master")
	if err != nil {
		return -1
	}
	if m := procent.FindStringSubmatch(out); m != nil {
		v, _ := strconv.Atoi(m[1])
		return v
	}
	return -1
}

// zetVolume past het volume aan, maar nooit boven het maximum van de ouders.
func zetVolume(stap, max int) int {
	v := volume()
	if v < 0 {
		return v
	}
	n := v + stap
	if n > max {
		n = max
	}
	if n < 0 {
		n = 0
	}
	_, _ = draai(2*time.Second, "amixer", "-q", "set", "Master", fmt.Sprintf("%d%%", n), "unmute")
	return volume()
}

func accu(taal string) string {
	out, err := draai(2*time.Second, "cat", "/sys/class/power_supply/BAT0/capacity")
	if err != nil || out == "" {
		return tr(taal, "battery.none")
	}
	status, _ := draai(2*time.Second, "cat", "/sys/class/power_supply/BAT0/status")
	if status == "Charging" || status == "Full" {
		return tr(taal, "battery.charging", out)
	}
	return out + "%"
}

type meldingMsg string

// hulpVragen stuurt een berichtje naar de telefoons van de ouders via ntfy.
func hulpVragen(o Ouders, naam, taal string) tea.Cmd {
	return func() tea.Msg {
		if o.Ntfy.Topic == "" {
			return meldingMsg(tr(taal, "help.notset"))
		}
		wie := naam
		if wie == "" {
			wie = tr(taal, "help.someone")
		}
		url := strings.TrimRight(o.Ntfy.URL, "/") + "/" + o.Ntfy.Topic
		req, _ := http.NewRequest("POST", url, strings.NewReader(tr(taal, "help.body", wie)))
		req.Header.Set("Title", tr(taal, "help.title"))
		req.Header.Set("Tags", "wave")
		c := http.Client{Timeout: 6 * time.Second}
		resp, err := c.Do(req)
		if err != nil {
			return meldingMsg(tr(taal, "help.offline"))
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return meldingMsg(tr(taal, "help.failed", resp.Status))
		}
		return meldingMsg(tr(taal, "help.sent"))
	}
}
