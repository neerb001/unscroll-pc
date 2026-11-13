package main

import (
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Updates zonder herstart. De updater zet een nieuwe versie klaar in /opt/unscroll/releases/<versie>
// en laat /usr/local/bin/unscroll ernaar wijzen. De draaiende app merkt dat het geïnstalleerde
// programma een ander bestand is dan het programma dat nu draait. Op een rustig moment stopt de app
// netjes; unscroll-sessie start hem meteen opnieuw, nu met de nieuwe versie. In de ouderstand kan
// een ouder de herstart ook meteen doen.

// geinstalleerdPad is waar de updater de nieuwste versie neerzet.
var geinstalleerdPad = "/usr/local/bin/unscroll"

// nieuweVersieKlaar geeft de naam van een geïnstalleerde versie die nog niet draait, of "".
// Een variabele zodat de tests hem kunnen vervangen.
var nieuweVersieKlaar = func() string {
	if versie == "dev" {
		return "" // een ontwikkelversie herstart zichzelf nooit
	}
	nu, err := os.Executable()
	if err != nil {
		return ""
	}
	if echt, err := filepath.EvalSymlinks(nu); err == nil {
		nu = echt
	}
	klaar, err := filepath.EvalSymlinks(geinstalleerdPad)
	if err != nil || klaar == nu {
		return ""
	}
	if _, err := os.Stat(klaar); err != nil {
		return ""
	}
	naam := filepath.Base(filepath.Dir(klaar))
	if naam == versie {
		return ""
	}
	return naam
}

// rustigeHerstartStilte: zo lang moet niemand een toets hebben aangeraakt.
const rustigeHerstartStilte = 15 * time.Minute

// rustigMoment zegt of de app nu ongemerkt opnieuw kan starten: avondslot of midden in de nacht,
// een kwartier niemand aan het toetsenbord, niets dat speelt, geen ouder bezig, en geen scherm
// waar misschien nog niet bewaard werk open staat.
func (m model) rustigMoment() bool {
	if m.ouderModus || m.pinOpen || m.oi.open {
		return false
	}
	if m.nu.Sub(m.laatsteToets) < rustigeHerstartStilte {
		return false
	}
	if m.l.nu != nil && !m.l.nu.stand.Pauze {
		return false
	}
	uur := m.nu.Hour()
	if !m.dicht() && (uur < 2 || uur >= 5) {
		return false
	}
	switch m.tab {
	case tabStart, tabLuisteren, tabLezen, tabVandaag, tabSysteem:
		return true
	}
	return false
}

// herstart stopt wat er speelt (het luisterboek onthoudt zijn plek) en sluit de app af;
// unscroll-sessie start hem daarna opnieuw.
func (m *model) herstart() tea.Cmd {
	return tea.Sequence(m.stopHuidig(true), tea.Quit)
}

// versieExtra: achter het versienummer op het systeemscherm.
func (m model) versieExtra(dim lipgloss.Style) string {
	if m.nieuw != "" {
		return dim.Render(m.t("sys.updateready", m.nieuw))
	}
	return dim.Render(m.t("sys.autoupdate"))
}
