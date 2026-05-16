package genelet

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type CGI interface {
	SetIP() string
	SetWhen() int
	Authenticate(string, string) error
}

type Access struct {
	Base
	CGI
}

func (self *Access) SetIP() string {
	ip := ""
	role := self.C.Roles[self.RoleValue]
	if role.Length != 0 {
		a := strings.Split(self.GetIP(), ".")
		full := fmt.Sprintf("%02X%02X%02X%02X", a[0], a[1], a[2], a[3])
		ip = full[0 : role.Length-1]
	}
	return ip
}

func (self *Access) SetWhen() int {
	return Unix_timestamp()
}

func NewAccess(base Base) *Access {
	a := new(Access)
	a.CGI = a
	a.Base = base
	return a
}

func (self *Access) Signature(fields ...string) string {
	login := fields[0]
	fields = append(fields[:0], fields[1:]...)
	if len(fields) == 0 {
		return self.sign(login, "")
	}
	return self.sign(login, strings.Join(fields, "|"))
}

func (self *Access) sign(login, group string) string {
	role := self.C.Roles[self.RoleValue]
	when := strconv.Itoa(self.CGI.SetWhen() + role.Duration)
	ip := self.CGI.SetIP()
	hash := Digest(role.Secret, ip, login, group, when)

	return EncodeScoder(strings.Join([]string{ip, login, url.PathEscape(group), when, hash}, "/"), role.Coding)
}

func (self *Access) getCookie(raws ...string) (string, string, string, string, string, error) {
	role, ok := self.C.Roles[self.RoleValue]
	if !ok {
		return "", "", "", "", "", Err(1029)
	}

	raw := ""
	if raws == nil {
		coo, err := self.R.Cookie(role.Surface)
		if err != nil {
			return "", "", "", "", "", Err(1030)
		}
		raw = coo.Value
	} else {
		raw = raws[0]
	}

	value := DecodeScoder(raw, role.Coding)
	x := strings.Split(value, "/")
	if len(x) < 5 {
		return "", "", "", "", "", Err(1020)
	}
	ip, login, group, when, hash := x[0], x[1], x[2], x[3], x[4]
	tmp, err := url.PathUnescape(group)
	if err != nil {
		return "", "", "", "", "", err
	}
	group = tmp
	if self.CGI.SetIP() != ip {
		return "", "", "", "", "", Err(1023)
	}
	w, err := strconv.Atoi(when)
	if err != nil {
		return "", "", "", "", "", Err(1026, err.Error())
	}
	requesttime := self.CGI.SetWhen()
	if requesttime > w {
		return "", "", "", "", "", Err(1022)
	}
	if role.Grouplist != nil && !Grep(role.Grouplist, group) {
		return "", "", "", "", "", Err(1021)
	}
	if role.Userlist != nil && !Grep(role.Userlist, login) {
		return "", "", "", "", "", Err(1021)
	}

	if Digest(role.Secret, ip, login, group, when) != hash {
		return "", "", "", "", "", Err(1024)
	}
	return ip, login, group, when, hash, nil
}

func (self *Access) VerifyCookie(raw string) error {
	_, login, group, when, hash, err := self.getCookie(raw)
	if err != nil {
		return err
	}

	self.R.Header.Add("X-Forwarded-Time", when)
	self.R.Header.Add("X-Forwarded-User", login)
	self.R.Header.Add("X-Forwarded-Group", group)
	self.R.Header.Add("X-Forwarded-Raw", raw)
	self.R.Header.Add("X-Forwarded-Hash", hash)
	role := self.C.Roles[self.RoleValue]
	self.R.Header.Add("X-Forwarded-Duration", strconv.Itoa(role.Duration))
	//	self.R.Header.Add("X-Forwarded-Request_Time", strconv.Itoa(self.CGI.SetWhen()))

	return nil
}
