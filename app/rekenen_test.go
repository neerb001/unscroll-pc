package main

import "testing"

func TestReken(t *testing.T) {
	gevallen := map[string]string{
		"1/3 + 1/6":    "1/2  = 0,5",
		"12 × (3 + 4)": "84",
		"7 : 2":        "3 1/2  = 3,5",
		"2(3+4)":       "14",
		"2^10":         "1024",
		"5²":           "25",
		"15% van 80":   "12",
		"2,5 * 4":      "10",
		"1/3":          "1/3  ≈ 0,333333",
		"-3 - -2":      "-1",
		"10 / 4":       "2 1/2  = 2,5",
		"100 ÷ 8":      "12 1/2  = 12,5",
	}
	for som, verwacht := range gevallen {
		v, err := reken(som)
		if err != nil {
			t.Errorf("%s: %v", som, err)
			continue
		}
		if got := toonGetal(v, ","); got != verwacht {
			t.Errorf("%s = %q, verwacht %q", som, got, verwacht)
		}
	}
	for som, fout := range map[string]error{"5 / 0": errDeelNul, "3 +": errSom, "(2+3": errSom, "hallo": errSom, "2^1000": errTeGroot} {
		if _, err := reken(som); err != fout {
			t.Errorf("%s: fout %v, verwacht %v", som, err, fout)
		}
	}
}
