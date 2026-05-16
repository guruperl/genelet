package genelet

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Base struct {
	C            *Config
	W            http.ResponseWriter
	R            *http.Request
	RoleValue    string
	ChartagValue string
}

func (self *Base) Fulfill() error {
	ARGS := self.R.Form
	goURI := ARGS.Get(self.C.GoURIName)
	self.RoleValue = ARGS.Get(self.C.RoleName)
	self.ChartagValue = ARGS.Get(self.C.TagName)

	if self.RoleValue != "" && self.ChartagValue != "" {
		return nil
	}

	newURL, err := url.Parse(goURI)
	if err != nil {
		return Err(404, "Redirected URL not found")
	}

	length := len(self.C.Script)
	if len(newURL.Path) <= length {
		return Err(404, "Redirected URL not found")
	}
	u1 := newURL.Path[:length]
	u2 := newURL.Path[length+1:]
	if u1 == self.C.Script && len(u2) > 0 {
		pathInfo := strings.Split(u2, "/")
		if len(pathInfo) < 2 {
			return Err(404, "Redirected URL not found")
		}
		self.RoleValue = pathInfo[0]
		self.ChartagValue = pathInfo[1]
	}

	if self.RoleValue == "" {
		return Err(404, "Redirected role name not found")
	}
	_, ok := self.C.Roles[self.RoleValue]
	if !ok {
		return Err(404, "Redirected role not found")
	}
	return nil
}

func (self *Base) GetRole() Role {
	return self.C.Roles[self.RoleValue]
}

func (self *Base) GetProvider() string {
	if self.RoleValue == "" {
		return ""
	}
	role, ok := self.C.Roles[self.RoleValue]
	if !ok {
		return ""
	}
	one := ""
	for key, val := range role.Issuers {
		if val.Default {
			return key
		}
		one = key
	}
	return one
}

func (self *Base) SendStatusPage(status int, output ...string) {
	chartag, ok := self.C.Chartags[self.ChartagValue]
	ct := "text/html; charset=UTF-8"
	if ok {
		ct = chartag.ContentType
	}
	self.W.Header().Set("Content-Type", ct)

	if status == 303 || status == 302 || status == 301 {
		if output != nil {
			self.W.Header().Set("Location", output[0])
		}
		self.W.WriteHeader(status)
		return
	}

	self.W.WriteHeader(status)
	if output != nil {
		self.W.Write([]byte(output[0]))
	}
}

func (self *Base) SendPage(output string) {
	self.SendStatusPage(200, output)
}

func (self *Base) SendNocache(output string) {
	self.W.Header().Set("Pragma", "no-cache")
	self.W.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	self.SendStatusPage(200, output)
}

func (self *Base) GetIP() string {
	host, _, _ := net.SplitHostPort(self.R.RemoteAddr)
	return host
}

//func (self *Base) GetIP_int() uint32 {
//	x := net.ParseIP(self.GetIP())
//	return binary.BigEndian.Uint32(x.To4())
//}

func (self *Base) SetCookie(name string, value string, maxAge ...int) {
	path := "/"
	domain := ""
	role, ok := self.C.Roles[self.RoleValue]
	if ok && role.Domain != "" {
		domain = role.Domain
	}
	if ok && role.Path != "" {
		path = role.Path
	}

	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		Domain:   domain,
		Path:     path,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   self.secureCookie(),
	}
	if maxAge != nil {
		expiration := time.Now().Add(time.Duration(maxAge[0]) * time.Second)
		cookie.MaxAge = maxAge[0]
		cookie.Expires = expiration
	}
	http.SetCookie(self.W, &cookie)
}

func (self *Base) SetCookieSession(name string, value string) {
	self.SetCookie(name, value)
}

func (self *Base) SetCookieExpire(name string) {
	self.SetCookie(name, "0", -365*24*3600)
}
