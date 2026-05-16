package genelet

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TAccess struct {
	Access
}

func (self *TAccess) SetIP() string {
	return "123.123.123.123"
}
func (self *TAccess) SetWhen() int {
	return 1
}
func NewTAccess(base Base) *TAccess {
	a := new(TAccess)
	a.CGI = a
	a.Base = base
	return a
}

func TestAccess(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	role := configure.Roles["m"]
	ac := role
	if ac.Surface != "mc" {
		t.Errorf("%s wanted", ac.Surface)
	}
	if ac.Coding != "member-local-coding" {
		t.Errorf("%s wanted", ac.Coding)
	}
	if ac.Secret != "member-local-secret" {
		t.Errorf("%s wanted", ac.Secret)
	}
	if ac.Domain != "genelet.com" {
		t.Errorf("%s wanted", ac.Domain)
	}
	if ac.Duration != 360000 {
		t.Errorf("%d wanted", ac.Duration)
	}
	if ac.Userlist[0] != "x1" {
		t.Errorf("%s wanted", ac.Userlist[0])
	}
	if ac.Userlist[2] != "x3" {
		t.Errorf("%s wanted", ac.Userlist[2])
	}
	if ac.Grouplist != nil {
		t.Errorf("%v wanted", ac.Grouplist)
	}

	r, err := http.NewRequest("GET", "http://xxx.yyy", bytes.NewBuffer([]byte("sss")))
	if err != nil {
		panic(err)
	}
	w := httptest.NewRecorder()
	base := newBase(configure, "m", "json", w, r)
	access := NewTAccess(*base)

	sig := access.Signature("x2", "g1", "g2", "g3")
	ret := access.VerifyCookie(sig) // within 1 second
	if ret != nil {
		t.Errorf("%s wanted", ret.Error())
	}
	if r.Header["X-Forwarded-User"][0] != "x2" {
		t.Errorf("%s wanted", "x2")
	}
	if r.Header["X-Forwarded-Group"][0] != "g1|g2|g3" {
		t.Errorf("%s wanted", "g1|g2|g3")
	}
	if r.Header["X-Forwarded-Time"][0] != "360001" {
		t.Errorf("%d wanted", 360001)
	}
	if r.Header["X-Forwarded-Duration"][0] != "360000" {
		t.Errorf("%d wanted", 360000)
	}
	if r.Header["X-Forwarded-Request_time"] != nil && r.Header["X-Forwarded-Request_time"][0] != "1" {
		t.Errorf("%d wanted", 1)
	}
	/*
		if r.Header["X-Forwarded-Raw"][0] != "Ec9rwEEzh1/0UTuoE7dvi+lv1yvnF2niRA2N4yYNbq3RoCp4b/Par3EUdNGGqUwsHfg1O6jtk3IvLaPSuW8/" {
			t.Errorf("%s wanted but we got %v", "Ec9rwEEzh1/0UTuoE7dvi+lv1yvnF2niRA2N4yYNbq3RoCp4b/Par3EUdNGGqUwsHfg1O6jtk3IvLaPSuW8/",  r.Header["X-Forwarded-Raw"])
		}
	*/
	wantHash := Digest(role.Secret, access.SetIP(), "x2", "g1|g2|g3", "360001")
	if r.Header["X-Forwarded-Hash"][0] != wantHash {
		t.Errorf("%s wanted", wantHash)
	}

	sig = access.Signature("x3", "g2", "g3", "g4")
	ret = access.VerifyCookie(sig) // within 1 second
	if ret != nil {
		t.Errorf("%s wanted", ret.Error())
	}
	if r.Header["X-Forwarded-User"][0] != "x2" {
		t.Errorf("%s wanted", "x2")
	}
	if r.Header["X-Forwarded-User"][1] != "x3" {
		t.Errorf("%s wanted", "x3")
	}
	if r.Header["X-Forwarded-Group"][0] != "g1|g2|g3" {
		t.Errorf("%s wanted", "g1|g2|g3")
	}
	if r.Header["X-Forwarded-Group"][1] != "g2|g3|g4" {
		t.Errorf("%s wanted", "g2|g3|g4")
	}

	sig = access.Signature("bad_guy", "g2", "g3", "g4")
	ret = access.VerifyCookie(sig) // within 1 second
	if ret.(Gerror).Code != 1021 {
		t.Errorf("%s wanted", ret.Error())
	}

	access.SendNocache("okok")
	h := w.Header()
	if h.Get("Cache-Control") != "no-cache, no-store, max-age=0, must-revalidate" {
		t.Errorf("%s gotten", h.Get("Cache-Control"))
	}
	if h.Get("Pragma") != "no-cache" {
		t.Errorf("%s gotten", h.Get("Pragma"))
	}
	if w.Body.String() != "okok" {
		t.Errorf("%s wanted", "okok")
	}
}
