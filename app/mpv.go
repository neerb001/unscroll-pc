package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Speler bestuurt één mpv-proces via de JSON IPC-socket. mpv draait zonder beeld en zonder
// terminal op de achtergrond en wordt pas gestart als er iets moet spelen.
type Speler struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	sock   string
	conn   net.Conn
	rd     *bufio.Reader
	id     int
	extern bool // tests: niet zelf mpv starten, alleen met sock verbinden
}

var errGeenMpv = errors.New("mpv not installed")

func nieuweSpeler() *Speler {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return &Speler{sock: filepath.Join(dir, fmt.Sprintf("unscroll-mpv-%d.sock", os.Getpid()))}
}

func (s *Speler) start() error {
	if s.conn != nil {
		return nil
	}
	if !s.extern {
		pad, err := exec.LookPath("mpv")
		if err != nil {
			return errGeenMpv
		}
		_ = os.Remove(s.sock)
		s.cmd = exec.Command(pad, "--idle=yes", "--no-video", "--no-terminal", "--config=no", "--ytdl=no",
			"--load-scripts=no", "--audio-display=no", "--keep-open=no", "--volume=100",
			"--cache=yes", "--demuxer-max-bytes=8MiB", "--demuxer-max-back-bytes=2MiB",
			"--input-ipc-server="+s.sock)
		stopMetOuder(s.cmd)
		if err := s.cmd.Start(); err != nil {
			s.cmd = nil
			return err
		}
		cmd := s.cmd
		go func() { _ = cmd.Wait() }()
	}
	var laatste error
	for i := 0; i < 60; i++ {
		c, err := net.Dial("unix", s.sock)
		if err == nil {
			s.conn, s.rd = c, bufio.NewReader(c)
			return nil
		}
		laatste = err
		time.Sleep(50 * time.Millisecond)
	}
	s.reset()
	return laatste
}

// reset sluit de verbinding en stopt mpv; de volgende opdracht start hem opnieuw.
func (s *Speler) reset() {
	if s.conn != nil {
		s.conn.Close()
	}
	s.conn, s.rd = nil, nil
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.cmd = nil
}

// Opdracht stuurt een opdracht naar mpv en geeft het antwoord ("data") terug.
func (s *Speler) Opdracht(args ...any) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.start(); err != nil {
		return nil, err
	}
	s.id++
	verzoek, _ := json.Marshal(map[string]any{"command": args, "request_id": s.id})
	_ = s.conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := s.conn.Write(append(verzoek, '\n')); err != nil {
		s.reset()
		return nil, err
	}
	for {
		regel, err := s.rd.ReadBytes('\n')
		if err != nil {
			s.reset()
			return nil, err
		}
		var a struct {
			Error     string          `json:"error"`
			Data      json.RawMessage `json:"data"`
			RequestID *int            `json:"request_id"`
		}
		if json.Unmarshal(regel, &a) != nil || a.RequestID == nil || *a.RequestID != s.id {
			continue // gebeurtenissen en oude antwoorden overslaan
		}
		if a.Error != "" && a.Error != "success" {
			return nil, errors.New(a.Error)
		}
		return a.Data, nil
	}
}

// Speel laadt een lijst bestanden en begint bij het eerste, op positie start (seconden).
// headers gaan alleen mee met deze lijst (bijvoorbeeld de sleutel van Audiobookshelf).
func (s *Speler) Speel(urls []string, start float64, headers string) error {
	if len(urls) == 0 {
		return errors.New("nothing to play")
	}
	if _, err := s.Opdracht("set_property", "http-header-fields", headers); err != nil {
		return err
	}
	beginWaarde := "none"
	if start > 0 {
		beginWaarde = fmt.Sprintf("%.1f", start)
	}
	if _, err := s.Opdracht("set_property", "start", beginWaarde); err != nil {
		return err
	}
	if _, err := s.Opdracht("loadfile", urls[0], "replace"); err != nil {
		return err
	}
	for _, u := range urls[1:] {
		if _, err := s.Opdracht("loadfile", u, "append"); err != nil {
			return err
		}
	}
	_, err := s.Opdracht("set_property", "pause", false)
	return err
}

// StartVrij zet de beginpositie terug, zodat het volgende bestand gewoon bij 0 begint.
func (s *Speler) StartVrij() { _, _ = s.Opdracht("set_property", "start", "none") }

func (s *Speler) Pauze(aan bool) { _, _ = s.Opdracht("set_property", "pause", aan) }
func (s *Speler) Spoel(sec float64) {
	_, _ = s.Opdracht("seek", sec, "relative")
}
func (s *Speler) Volgende() { _, _ = s.Opdracht("playlist-next") }
func (s *Speler) Vorige()   { _, _ = s.Opdracht("playlist-prev") }
func (s *Speler) Leeg()     { _, _ = s.Opdracht("stop") }

// Afsluiten stopt mpv helemaal (bij het afsluiten van de app).
func (s *Speler) Afsluiten() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_, _ = s.conn.Write([]byte(`{"command":["quit"]}` + "\n"))
	}
	s.reset()
	if !s.extern {
		_ = os.Remove(s.sock)
	}
}

// SpelerStand is wat de app elke seconde van mpv wil weten.
type SpelerStand struct {
	Leeg, Pauze bool
	Pos, Duur   float64
	Index       int
	Titel       string
}

func (s *Speler) Stand() (SpelerStand, error) {
	var st SpelerStand
	lees := func(naam string, doel any) error {
		d, err := s.Opdracht("get_property", naam)
		if err != nil {
			return err
		}
		return json.Unmarshal(d, doel)
	}
	if err := lees("idle-active", &st.Leeg); err != nil {
		return st, err
	}
	_ = lees("pause", &st.Pauze)
	_ = lees("time-pos", &st.Pos)
	_ = lees("duration", &st.Duur)
	_ = lees("playlist-pos", &st.Index)
	if lees("metadata/by-key/icy-title", &st.Titel) != nil {
		st.Titel = ""
	}
	return st, nil
}
