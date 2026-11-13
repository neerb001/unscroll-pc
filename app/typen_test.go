package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const nepTyp = `# test
G:MENU
*:S_LESSON1
B:                           Lesson 1: j f
*:LESSON1_D1
I:(1)
D:jjj fff
 :jf fj
*:LESSON1_D2
D:fjf
G:E_LESSON1
*:S_LESSON2
B:                           Lesson 2: g h
D:gh hg
`

func TestTypcursus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	_ = os.WriteFile(filepath.Join(dir, "ktnl_junior.typ"), []byte(nepTyp), 0o644)
	oud := typMap
	typMap = dir
	defer func() { typMap = oud }()

	lessen, err := leesTypLessen(filepath.Join(dir, "ktnl_junior.typ"))
	if err != nil || len(lessen) != 2 || lessen[0].Titel != "j f" || len(lessen[0].Oefeningen) != 2 || lessen[0].Oefeningen[0][1] != "jf fj" {
		t.Fatalf("lessen: %+v %v", lessen, err)
	}
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 100, 34
	m = toets(m, "6")
	for i := 0; i < 3; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // typcursus
	if !strings.Contains(m.View(), "les 1: j f") {
		t.Fatalf("lessenlijst:\n%s", m.View())
	}
	m = toets(m, "enter") // les 1
	if !strings.Contains(m.View(), "rechter wijsvinger") {
		t.Fatalf("vinger voor j:\n%s", m.View())
	}
	m = toets(m, "jjk") // één fout: k in plaats van j
	m = toets(m, "backspace")
	m = toets(m, "j fff")
	m = toets(m, "jf fj")
	if m.sh.ty.oef != 0 || !m.sh.ty.klaar || !strings.Contains(m.View(), "Deze toetsen waren nog lastig") {
		t.Fatalf("na oefening 1:\n%s", m.View())
	}
	m = toets(m, "enter")
	m = toets(m, "fjf")
	m = toets(m, "enter") // les 1 klaar → les 2
	if m.sh.ty.les != 1 || !laadTypVoortgang().Klaar["nl-0"] {
		t.Fatalf("door naar les 2: les %d, voortgang %+v", m.sh.ty.les, laadTypVoortgang())
	}
	m = toets(m, "esc")
	if !strings.Contains(m.View(), "✓ les 1") {
		t.Errorf("vinkje bij les 1:\n%s", m.View())
	}
	// eigen zinnen
	m.sh.ty.sel = len(m.sh.ty.lessen)
	m = toets(m, "enter")
	if m.sh.ty.les != -1 || len(m.sh.ty.regels) == 0 {
		t.Error("eigen zinnen")
	}
}
