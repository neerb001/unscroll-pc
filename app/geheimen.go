package main

import (
	"bufio"
	"os"
	"strings"
)

// Geheimen staan alleen op het apparaat, in /etc/unscroll/geheimen.env (root, leesbaar voor de app).
// Ze worden gezet met sudo unscroll-ouder en komen nooit in git.
type Geheimen map[string]string

// adres maakt van wat een ouder intypt een bruikbaar adres:
// "192.168.1.10:4533" wordt "http://192.168.1.10:4533", spaties en een / aan het eind gaan weg.
func adres(s string) string {
	s = strings.TrimRight(strings.TrimSpace(s), "/")
	if s != "" && !strings.Contains(s, "://") {
		s = "http://" + s
	}
	return s
}

func geheimenPad() string {
	if p := os.Getenv("UNSCROLL_GEHEIMEN"); p != "" {
		return p
	}
	return "/etc/unscroll/geheimen.env"
}

func laadGeheimen() Geheimen {
	f, err := os.Open(geheimenPad())
	if err != nil {
		return Geheimen{}
	}
	defer f.Close()
	return leesEnv(f)
}

// leesEnv leest regels als NAAM=waarde, NAAM="waarde" of NAAM='waarde' (zoals bash ze schrijft).
func leesEnv(f *os.File) Geheimen {
	g := Geheimen{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		r := strings.TrimSpace(sc.Text())
		if r == "" || r[0] == '#' {
			continue
		}
		r = strings.TrimPrefix(r, "export ")
		naam, waarde, ok := strings.Cut(r, "=")
		if !ok {
			continue
		}
		g[strings.TrimSpace(naam)] = shellWaarde(waarde)
	}
	return g
}

// shellWaarde haalt de aanhalingstekens weg zoals een shell dat doet: 'a'\”b' wordt a'b.
func shellWaarde(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\'':
			j := strings.IndexByte(s[i+1:], '\'')
			if j < 0 {
				b.WriteString(s[i+1:])
				return b.String()
			}
			b.WriteString(s[i+1 : i+1+j])
			i += j + 1
		case '"':
			for i++; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' && i+1 < len(s) && strings.IndexByte("\"\\$`", s[i+1]) >= 0 {
					i++
				}
				b.WriteByte(s[i])
			}
		case '\\':
			if i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			}
		case ' ', '\t':
			return b.String() // rest van de regel is commentaar
		case '#':
			if i == 0 {
				return ""
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
