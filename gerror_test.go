package genelet

import (
	"testing"
)

func TestGerror(t *testing.T) {
	e0 := Err(123)
	e1 := Gerror{123, "abc"}
	e2 := Gerror{1113, ""}
	e3 := Gerror{1113, "special"}
	e4 := Gerror{0, "abcde"}
	e5p := new(Gerror)
	e5p.Errstr = "xxxxx"
	if e0.Error() != "123" {
		t.Errorf("%s wanted", e0.Error())
	}
	if e1.Error() != "abc" {
		t.Errorf("%s wanted", e1.Error())
	}
	if e2.Error() != "Invalid email request." {
		t.Errorf("%s wanted", e2.Error())
	}
	if e3.Error() != "special" {
		t.Errorf("%s wanted", e3.Error())
	}
	if e4.Error() != "abcde" {
		t.Errorf("%s wanted", e4.Error())
	}
	if e5p.Code != 0 {
		t.Errorf("%d wanted", e5p.Code)
	}
	if e5p.Errstr != "xxxxx" {
		t.Errorf("%s wanted", e5p.Errstr)
	}
	if e5p.Error() != "xxxxx" {
		t.Errorf("%s wanted", e5p.Error())
	}
}
