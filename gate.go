package genelet

import (
	"net/url"
	"strings"
)

type Gate struct {
	Access
}

func NewGate(base Base) *Gate {
	a := new(Gate)
	a.CGI = a
	a.Base = base
	return a
}

func (self *Gate) Forbid() error {
	c := self.C
	role, ok := c.Roles[self.RoleValue]
	if !ok {
		return nil
	}

	coo, err := self.R.Cookie(role.Surface)
	if err == nil {
		err = self.VerifyCookie(coo.Value)
		if err == nil {
			return nil
		}
	}

	chartag, ok := c.Chartags[self.ChartagValue]
	if ok && chartag.Case > 0 {
		return Err(200, chartag.CallChallenge())
	}

	escaped := url.QueryEscape(self.R.RequestURI)
	self.SetCookieSession(c.GoProbeName, escaped)
	self.SetCookieExpire(role.Surface)

	var k string
	for k = range role.Issuers {
		if !Grep(c.Oauth2s, k) && !Grep(c.Oauth1s, k) {
			redirect := c.Script + "/" + self.RoleValue + "/" + self.ChartagValue + "/" + c.LoginName + "?" + c.GoURIName + "=" + escaped + "&" + c.GoErrName + "=1025&" + c.RoleName + "=" + self.RoleValue + "&" + c.TagName + "=" + self.ChartagValue + "&" + c.ProviderName + "=" + k
			return Err(303, redirect)
		}
	}
	redirect := c.Script + "/" + self.RoleValue + "/" + self.ChartagValue + "/" + k + "?" + c.GoURIName + "=" + escaped + "&" + c.GoErrName + "=1025&" + c.TagName + "=" + self.ChartagValue
	return Err(303, redirect)
}

func (self *Gate) HandleLogout() error {
	role, ok := self.C.Roles[self.RoleValue]
	if !ok {
		return Err(1029)
	}

	self.SetCookieExpire(role.Surface)
	self.SetCookieExpire(role.Surface + "_")
	self.SetCookieExpire(self.C.GoProbeName)

	chartag, ok := self.C.Chartags[self.ChartagValue]
	if ok && chartag.Case > 0 {
		return Err(200, chartag.CallLogout())
	} else {
		return Err(303, role.Logout)
	}
}

/*
func (self *Gate)GetAttribute(ref string) (string, error) {
	role, ok := self.C.Roles[self.RoleValue]
    if (!ok) { return "", Err(1029) }

	_, login, group, _, _, err := self.getCookie()
    if err != nil { return "", err }

	if (ref==role.Attributes[0]) {
		return login, nil
	}

	groups := strings.Split(group, "|")
	for i, a := range role.Attributes {
		if (len(groups) >= i && ref==a) {
			return groups[i-1], nil
		}
	}

	return "", Err(1039)
}
*/

func (self *Gate) GetAttribute(key string) (string, error) {
	ref := make(map[string]string)
	err := self.GetAttributes(ref)
	if err != nil {
		return "", err
	}
	val, ok := ref[key]
	if !ok {
		return "", Err(1039)
	}

	return val, nil
}

func (self *Gate) GetAttributes(ref map[string]string) error {
	role, ok := self.C.Roles[self.RoleValue]
	if !ok {
		return Err(1029)
	}

	_, login, group, _, _, err := self.getCookie()
	if err != nil {
		return err
	}

	groups := strings.Split(group, "|")
	for i, a := range role.Attributes {
		if i == 0 {
			ref[a] = login
		} else if len(groups) > i-1 {
			ref[a] = groups[i-1]
		}
	}
	return nil
}

func (self *Gate) SetAttribute(key string, value string) error {
	ref := make(map[string]string)
	ref[key] = value
	return self.SetAttributes(ref)
}

func (self *Gate) SetAttributes(ref map[string]string) error {
	role, ok := self.C.Roles[self.RoleValue]
	if !ok {
		return Err(1029)
	}

	ip, login, group, when, _, err := self.getCookie()
	if err != nil {
		return err
	}

	n_login, ok := ref[role.Attributes[0]]
	if ok {
		login = n_login
	}

	groups := strings.Split(group, "|")
	new_groups := make([]string, len(role.Attributes)-1)
	for i := 1; i < len(role.Attributes); i++ {
		n_value, ok := ref[role.Attributes[i]]
		if ok {
			new_groups[i-1] = n_value
		} else if len(groups) > i-1 {
			new_groups[i-1] = groups[i-1]
		} else {
			new_groups[i-1] = ""
		}
	}

	new_group := strings.Join(new_groups, "|")
	newHash := Digest(role.Secret, ip, login, new_group, when)
	signed := EncodeScoder(strings.Join([]string{ip, login, new_group, when, newHash}, "/"), role.Coding)

	self.SetCookie(role.Surface, signed, role.MaxAge)
	self.SetCookieSession(role.Surface+"_", signed)

	return nil
}
