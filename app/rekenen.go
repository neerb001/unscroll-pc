package main

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

// Rekenmachine met exacte breuken: 1/3 + 1/6 is precies 1/2. Begrijpt ook × ÷ : en een komma
// als decimaalteken, haakjes, machten (^ of ²) en procenten ("15% van 80").

var (
	errDeelNul   = errors.New("deelnul")
	errSom       = errors.New("som")
	errTeGroot   = errors.New("tegroot")
	procentVan   = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*%\s*(van|of)\s*`)
	procentLos   = regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*%`)
	maalVoorHaak = regexp.MustCompile(`(\d|\))\s*\(`)
)

type rekenaar struct {
	t   []rune
	pos int
}

func reken(som string) (*big.Rat, error) {
	s := strings.TrimSpace(som)
	s = procentVan.ReplaceAllString(s, "($1/100)*")
	s = procentLos.ReplaceAllString(s, "($1/100)")
	r := strings.NewReplacer("×", "*", "x", "*", "X", "*", "·", "*", "÷", "/", ":", "/", "−", "-", "²", "^2", "³", "^3", ",", ".")
	s = r.Replace(s)
	s = maalVoorHaak.ReplaceAllString(s, "$1*(") // 2(3+4) betekent 2·(3+4)
	if s == "" {
		return nil, errSom
	}
	p := &rekenaar{t: []rune(s)}
	v, err := p.plus()
	if err != nil {
		return nil, err
	}
	p.spaties()
	if p.pos != len(p.t) {
		return nil, errSom
	}
	return v, nil
}

func (p *rekenaar) spaties() {
	for p.pos < len(p.t) && unicode.IsSpace(p.t[p.pos]) {
		p.pos++
	}
}

func (p *rekenaar) kijk() rune {
	p.spaties()
	if p.pos < len(p.t) {
		return p.t[p.pos]
	}
	return 0
}

func (p *rekenaar) plus() (*big.Rat, error) {
	v, err := p.maal()
	if err != nil {
		return nil, err
	}
	for {
		switch p.kijk() {
		case '+', '-':
			op := p.t[p.pos]
			p.pos++
			w, err := p.maal()
			if err != nil {
				return nil, err
			}
			if op == '+' {
				v = new(big.Rat).Add(v, w)
			} else {
				v = new(big.Rat).Sub(v, w)
			}
		default:
			return v, nil
		}
	}
}

func (p *rekenaar) maal() (*big.Rat, error) {
	v, err := p.macht()
	if err != nil {
		return nil, err
	}
	for {
		switch p.kijk() {
		case '*', '/':
			op := p.t[p.pos]
			p.pos++
			w, err := p.macht()
			if err != nil {
				return nil, err
			}
			if op == '*' {
				v = new(big.Rat).Mul(v, w)
			} else {
				if w.Sign() == 0 {
					return nil, errDeelNul
				}
				v = new(big.Rat).Quo(v, w)
			}
		default:
			return v, nil
		}
	}
}

func (p *rekenaar) macht() (*big.Rat, error) {
	v, err := p.enkel()
	if err != nil {
		return nil, err
	}
	if p.kijk() == '^' {
		p.pos++
		e, err := p.macht()
		if err != nil {
			return nil, err
		}
		if !e.IsInt() || e.Num().CmpAbs(big.NewInt(64)) > 0 {
			return nil, errTeGroot
		}
		n := e.Num().Int64()
		uit := big.NewRat(1, 1)
		for i := int64(0); i < abs64(n); i++ {
			uit.Mul(uit, v)
		}
		if n < 0 {
			if uit.Sign() == 0 {
				return nil, errDeelNul
			}
			uit.Inv(uit)
		}
		return uit, nil
	}
	return v, nil
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func (p *rekenaar) enkel() (*big.Rat, error) {
	switch c := p.kijk(); {
	case c == '-':
		p.pos++
		v, err := p.enkel()
		if err != nil {
			return nil, err
		}
		return new(big.Rat).Neg(v), nil
	case c == '+':
		p.pos++
		return p.enkel()
	case c == '(':
		p.pos++
		v, err := p.plus()
		if err != nil {
			return nil, err
		}
		if p.kijk() != ')' {
			return nil, errSom
		}
		p.pos++
		return v, nil
	case c >= '0' && c <= '9' || c == '.':
		begin := p.pos
		for p.pos < len(p.t) && (p.t[p.pos] >= '0' && p.t[p.pos] <= '9' || p.t[p.pos] == '.') {
			p.pos++
		}
		if p.pos-begin > 30 {
			return nil, errTeGroot
		}
		v, ok := new(big.Rat).SetString(string(p.t[begin:p.pos]))
		if !ok {
			return nil, errSom
		}
		return v, nil
	}
	return nil, errSom
}

// toonGetal: geheel getal, of breuk met het kommagetal erachter. komma: "," of ".".
func toonGetal(v *big.Rat, komma string) string {
	if v.Num().BitLen() > 200 || v.Denom().BitLen() > 200 {
		f, _ := v.Float64()
		return strings.Replace(fmt.Sprintf("%.6g", f), ".", komma, 1)
	}
	if v.IsInt() {
		return v.Num().String()
	}
	dec := strings.TrimRight(strings.TrimRight(v.FloatString(6), "0"), ".")
	dec = strings.Replace(dec, ".", komma, 1)
	breuk := v.Num().String() + "/" + v.Denom().String()
	// gemengd getal als het meer dan 1 is: 7/2 = 3 1/2
	heel := new(big.Int).Quo(v.Num(), v.Denom())
	if heel.Sign() != 0 {
		rest := new(big.Int).Rem(new(big.Int).Abs(v.Num()), v.Denom())
		breuk = heel.String() + " " + rest.String() + "/" + v.Denom().String()
	}
	if v.Denom().BitLen() > 30 { // lange breuk: alleen het kommagetal
		return "≈ " + dec
	}
	teken := "≈" // afgerond, tenzij het kommagetal precies klopt (zoals 1/4 = 0,25)
	if new(big.Rat).Mul(v, big.NewRat(1000000, 1)).IsInt() {
		teken = "="
	}
	return breuk + "  " + teken + " " + dec
}
