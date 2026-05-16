package genelet

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

type Beacon struct {
	Controller
	RoleValue    string
	ChartagValue string
	Header       http.Header

	u   *url.URL
	jar *cookiejar.Jar
	ua  string

	Code     int
	Redirect string
	Content  string
}

func NewBeacon(controller Controller, role, tag string, header http.Header) (*Beacon, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		panic(err)
	}
	u, err := url.Parse(controller.C.ServerURL)
	if err != nil {
		panic(err)
	}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"

	return &Beacon{controller, role, tag, header, u, jar, ua, 0, "", ""}, nil
}

func (self *Beacon) Cookies() []*http.Cookie {
	return self.jar.Cookies(self.u)
}

func (self *Beacon) setCommon(r *http.Request) {
	r.Header.Set("User-Agent", self.ua)
	for _, cookie := range self.jar.Cookies(self.u) {
		r.AddCookie(cookie)
	}
	for k, vs := range self.Header {
		for _, v := range vs {
			r.Header.Add(k, v)
		}
	}
}

func (self *Beacon) getURL(pars ...string) string {
	c := self.C
	in := c.ServerURL + c.Script + "/" + self.RoleValue + "/" + self.ChartagValue + "/" + pars[0]
	if len(pars) > 1 {
		in += "?" + pars[1]
	}
	return in
}

func (self *Beacon) GetDirect(in string) error {
	r := httptest.NewRequest("GET", in, nil)
	self.setCommon(r)
	self.run(r)
	return nil
}

func (self *Beacon) GetMock(pars ...string) error {
	r := httptest.NewRequest("GET", self.getURL(pars...), nil)
	self.setCommon(r)
	self.run(r)
	return nil
}

func (self *Beacon) PostMock(obj string, args url.Values) error {
	req := httptest.NewRequest("POST", self.getURL(obj), strings.NewReader(args.Encode()))
	self.setCommon(req)
	self.run(req)
	return nil
}

func (self *Beacon) LOGIN(args url.Values) error {
	return self.PostMock(self.C.LoginName, args)
}

func (self *Beacon) run(r *http.Request) {
	w := httptest.NewRecorder()
	self.ServeHTTP(w, r)
	resp := w.Result()
	if cookies := resp.Cookies(); cookies != nil {
		self.jar.SetCookies(self.u, cookies)
	}
	if w.Code == 301 || w.Code == 303 {
		loc, err := resp.Location()
		if err != nil {
			panic(err)
		}
		self.Redirect = loc.String()
	}
	if w.Code >= 300 {
		self.Content = ""
	} else {
		self.Content = w.Body.String()
	}
	self.Code = w.Code
}
