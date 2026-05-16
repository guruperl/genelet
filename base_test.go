package genelet

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func newBase(c *Config, rv string, cv string, w http.ResponseWriter, r *http.Request) *Base {
	return &Base{C: c, W: w, R: r, RoleValue: rv, ChartagValue: cv}
}

func TestBase(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	//base := new_Base(configure, "", "", nil, nil)

	d := Digest64("1234567", fmt.Sprint("root", "script", "tmpl"))
	if d != "v7AeDE9+Z6iXuxheHt2fEfEpejI=" {
		t.Errorf("digest64 wrong %s", d)
	}

	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		log.Fatal(err)
	}
	w := httptest.NewRecorder()
	b := newBase(configure, "", "", w, req)
	b.SetCookie("c", "v", 100)
	h := w.Header()
	matched, err := regexp.MatchString("^c=v; Path=/; Expires=.*; HttpOnly; SameSite=Lax$", h.Get("Set-Cookie"))
	if err != nil || !matched {
		t.Errorf("%s gotten", h.Get("Set-Cookie"))
	}

	req, err = http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		log.Fatal(err)
	}
	w = httptest.NewRecorder()
	b = newBase(configure, "", "", w, req)
	b.SetCookieExpire("c")
	b.SendPage("okokok")
	h = w.Header()
	matched, err = regexp.MatchString("^c=0; Path=/; Expires=.*; Max-Age=0; HttpOnly; SameSite=Lax$", h.Get("Set-Cookie"))
	if err != nil || !matched {
		t.Errorf("%s gotten", h.Get("Set-Cookie"))
	}
	if w.Body.String() != "okokok" {
		t.Errorf("%s wanted", "okokok")
	}

	err = req.ParseForm()
	if err != nil {
		panic(err)
	}
	req.Form.Add("go_uri", "bbAAA/BBB/CCC?action=x")
	err = b.Fulfill()
	if err == nil || err.(Gerror).Errstr != "Redirected role not found" {
		t.Errorf("%d code for %s\n", err.(Gerror).Code, b.RoleValue)
	}
	b.C.Script = "/bb"
	req.Form.Set("go_uri", "/bb/m/BBB/CCC?action=x")
	err = b.Fulfill()
	if err != nil {
		t.Errorf("%d code for %s\n", err.(Gerror).Code, b.RoleValue)
	}

	if b.RoleValue != "m" {
		t.Errorf("role is %s", b.RoleValue)
	}
	if b.ChartagValue != "BBB" {
		t.Errorf("chartag is %s", b.ChartagValue)
	}
	if b.GetProvider() != "db" {
		t.Errorf("provider is %s", b.GetProvider())
	}
}
