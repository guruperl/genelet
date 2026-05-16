package genelet

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TTicket struct {
	Ticket
}

func NewTTicket(base Base, uri string, provider string) *TTicket {
	a := new(TTicket)
	a.CGI = a
	a.Base = base
	a.Uri = uri
	a.Provider = provider
	return a
}
func (self *TTicket) SetIP() string {
	return "123.123.123.123"
}
func (self *TTicket) SetWhen() int {
	return 1
}
func (self *TTicket) Authenticate(login, password string) error {
	if login == "" || password == "" {
		return Err(1037)
	}
	role := self.C.Roles[self.RoleValue]
	issuer := role.Issuers[self.Provider]
	if login != issuer.ProviderPars["Def_login"] || password != issuer.ProviderPars["Def_password"] {
		return Err(1031)
	}

	self.Out_hash = map[string]interface{}{"email": issuer.ProviderPars["Def_login"], "user": "x2"}

	return nil
}

func TestTicket(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	//b := newBase(configure, "m", "json", w, req)
	// test login page with json
	//ticket := NewTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	//str := ticket.Login_page(1001)
	//if str != "{\"data\":\"failed\"}" {
	//	t.Errorf("%s wanted", str)
	//}

	// test login page with mime foo and error code 1001
	//b = newBase(configure, "m", "foo", w, req)
	//ticket = NewTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	//str = ticket.Login_page(1001)
	//matched, err := regexp.MatchString("Google authorization required", str)
	//if !matched {
	//	t.Errorf("%s wanted", str)
	//}
	//matched, err = regexp.MatchString("email.*passwd", str)
	//if !matched {
	//		t.Errorf("%s wanted", str)
	//	}
	// the location of on-disk template, that does not exist
	//if (ticket.C.Template+"/"+ticket.RoleValue+"/"+ticket.C.Roles["m"].Login+"."+ticket.ChartagValue != "ee/m/login.foo") {
	//		t.Errorf("%s\t%s\t%s\t%s\n", ticket.C.Template, ticket.RoleValue, ticket.C.Roles["m"].Login, ticket.ChartagValue)
	//}

	// test authentication
	b := newBase(configure, "m", "json", w, req)
	tticket := NewTTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	tticket.Uri = "asw"
	ret := tticket.Authenticate("", "w")
	if ret.(Gerror).Code != 1037 {
		t.Errorf("%s returned", ret.Error())
	}
	ret = tticket.Authenticate("h", "w")
	if ret.(Gerror).Code != 1031 {
		t.Errorf("%s returned", ret.Error())
	}
	ret = tticket.Authenticate("hello", "world")
	if ret != nil {
		t.Errorf("%s returned", ret.Error())
	}
	if tticket.C.Roles["m"].Attributes[0] != "email" {
		t.Errorf("%s wanted", tticket.C.Roles["m"].Attributes[0])
	}
	if tticket.C.Roles["m"].Attributes[1] != "m_id" {
		t.Errorf("%s wanted", tticket.C.Roles["m"].Attributes[1])
	}
	if tticket.Out_hash["email"] != "hello" {
		t.Errorf("%s wanted", tticket.Out_hash["login"])
	}
	if tticket.Out_hash["user"] != "x2" {
		t.Errorf("%s wanted", tticket.Out_hash["x2"])
	}
	ret = tticket.HandlerFields()
	if ret != nil {
		t.Errorf("%s returned", ret.Error())
	}

	// test Handler_login which also test authentication and login page
	w = httptest.NewRecorder()
	b = newBase(configure, "m", "e", w, req)
	tticket = NewTTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler_login()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s returned", ret.Error())
	}
	h := w.Header()
	setCookie := h.Get("Set-Cookie")
	if !strings.HasPrefix(setCookie, "mc=") {
		t.Errorf("%s wanted", setCookie)
	}
	cookieValue := strings.SplitN(strings.TrimPrefix(setCookie, "mc="), ";", 2)[0]
	ret = tticket.VerifyCookie(cookieValue)
	if ret != nil {
		t.Errorf("%s wanted", ret.Error())
	}
	h = tticket.R.Header
	if h["X-Forwarded-User"][0] != "hello" {
		t.Errorf("%s wanted", h["X-Forwarded-User"])
	}
	if h["X-Forwarded-Group"][0] != `||||` {
		t.Errorf("%s wanted", h["X-Forwarded-Group"])
	}

	// test Handler with direct login
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world&direct=1", nil)
	w = httptest.NewRecorder()
	b = newBase(configure, "m", "e", w, req)
	tticket = NewTTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler_login()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s wanted", ret.Error())
	}
	h = w.Header()
	if !strings.HasPrefix(h.Get("Set-Cookie"), "mc=") {
		t.Errorf("%s wanted", h.Get("Set-Cookie"))
	}

	// test Handler with error code but no cookie
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world&go_err=1020", nil)
	w = httptest.NewRecorder()
	b = newBase(configure, "m", "e", w, req)
	tticket = NewTTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler()
	if ret.(Gerror).Code != 1036 {
		t.Errorf("%#v found", ret)
	}

	// test Handler with error code and cookie, the most common case for redirect
	req, _ = http.NewRequest("GET", "http://example.com/foo?go_uri=xxxx&go_err=1020", nil)
	w = httptest.NewRecorder()
	b = newBase(configure, "m", "e", w, req)
	tticket = NewTTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	cookie := &http.Cookie{Name: "go_probe", Value: "http://xxx.yyy.zzz/foo/bar", Path: "/", Domain: "genelet.com"}
	req.AddCookie(cookie)
	ret = tticket.Handler()
	if ret.(Gerror).Code != 1020 {
		t.Errorf("%#v found", ret)
	}

	// test Handler with login and password, as well as cookie
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world", nil)
	w = httptest.NewRecorder()
	b = newBase(configure, "m", "e", w, req)
	tticket = NewTTicket(*b, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	cookie = &http.Cookie{Name: "go_probe", Value: "http://xxx.yyy.zzz/foo/bar", Path: "/", Domain: "genelet.com"}
	req.AddCookie(cookie)
	ret = tticket.Handler()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s wanted", ret.Error())
	}
	h = w.Header()
	if !strings.HasPrefix(h.Get("Set-Cookie"), "mc=") {
		t.Errorf("%s wanted", h.Get("Set-Cookie"))
	}
	if ret.(Gerror).Errstr != "http://xxx.yyy.zzz/foo/bar" {
		t.Errorf("%s wanted", ret.(Gerror).Errstr)
	}
}
