package genelet

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

type TGate struct {
	Gate
}

func (self *TGate) SetIP() string {
	return "123.123.123.123"
}
func (self *TGate) SetWhen() int {
	return 1
}
func NewTGate(base Base) *TGate {
	a := new(TGate)
	a.CGI = a
	a.Base = base
	return a
}

func TestGate(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	b := newBase(configure, "m", "json", w, req)
	g := NewGate(*b)
	err = g.HandleLogout()
	if gerr, ok := err.(Gerror); !ok || gerr.Code != 200 {
		t.Fatalf("HandleLogout = %#v, want 200 Gerror", err)
	}

	h := w.Header()["Set-Cookie"]
	matched, err := regexp.MatchString("^mc=0; Path=/; Domain=genelet", h[0])
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("%s wanted", h[0])
	}
	matched, err = regexp.MatchString("^mc_=0; Path=/; Domain=genelet", h[1])
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("%s wanted", h[1])
	}
	matched, err = regexp.MatchString("^go_probe=0; Path=/; Domain=genelet", h[2])
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("%s wanted", h[2])
	}

	access := NewTGate(*b)
	cookie := &http.Cookie{Name: "mc", Value: access.Signature("x2", "1", "first", "last"), Path: "/", Domain: "genelet.com"}
	b.R.AddCookie(cookie)
	err = access.Forbid()
	if err != nil {
		t.Errorf("%s got\n", err.Error())
	}

	access.SetAttributes(map[string]string{"last_name": "aaa", "address": "bbb", "company": "ccc"})
	x := w.Header()
	c := x["Set-Cookie"][3]
	if !strings.HasPrefix(c, "mc=") {
		t.Errorf("%s wanted", c)
	}

	b.R.Header.Del("Cookie")
	cookie = &http.Cookie{Name: "mc", Value: strings.SplitN(strings.TrimPrefix(c, "mc="), ";", 2)[0], Path: "/", Domain: "genelet.com"}
	b.R.AddCookie(cookie)
	access = NewTGate(*b)
	err = access.Forbid()
	if err != nil {
		t.Errorf("%s got\n", err.Error())
	}

	b.R.Header.Del("Cookie")
	cookie = &http.Cookie{Name: "mc", Value: "xxEc9rwEEzh1/0UTuoE7dvi/k4lCC5RHm/SgbasG0Jca7XoTUFbKrrnkpWOcmZ8UQUEPAMPeLsi0pteOPNl2s1TO2I", Path: "/", Domain: "genelet.com"}
	b.R.AddCookie(cookie)
	err = access.Forbid()
	if err.Error() != `{"data":"challenge"}` {
		t.Errorf("%s got\n", err.Error())
	}
	access.ChartagValue = "e"
	err = access.Forbid()
	if err.Error() != `bb/m/e/login?go_uri=&go_err=1025&role=m&tag=e&provider=db` {
		t.Errorf("%s got\n", err.Error())
	}
}
