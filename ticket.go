package genelet

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	int_cipher "github.com/delongw/go-int-cipher"
)

type Ticket struct {
	Access
	Uri      string
	Provider string
	Out_hash map[string]interface{}
	// each item in condition uri is: variable name, how, variable value, base uri, other variable names wholse values would be passed in the redirect.
	// how: t--out hash, u--go uri, p--provider; A--equal, B--not equal, C--match; 0--no sign, 1--sign;  0--no authen cookie, 1--yes authen cookie
	// for u and p in 'how', 'variable name' will be go_uri that matches
	ConditionURI [][]string
}

func NewTicket(base Base, uri string, provider string) *Ticket {
	a := new(Ticket)
	a.CGI = a
	a.Base = base
	a.Uri = uri
	a.Provider = provider
	return a
}

func (self *Ticket) Handler() error {
	ARGS := self.R.Form
	found, err := self.R.Cookie(self.C.GoProbeName)
	if err != nil {
		self.SetCookieSession(self.C.GoProbeName, self.R.RequestURI)
		return Err(1036)
	}
	if self.Uri == "" {
		self.Uri = found.Value
	}
	if self.Uri, err = self.validatedRedirect(self.Uri); err != nil {
		return err
	}

	if ARGS.Get(self.C.GoErrName) != "" {
		code, _ := strconv.Atoi(ARGS.Get(self.C.GoErrName))
		return Err(code)
	}

	return self.Handler_login()
}

func (self *Ticket) Handler_login() error {
	ARGS := self.R.Form
	role := self.C.Roles[self.RoleValue]
	issuer := role.Issuers[self.Provider]
	var err error
	if self.Uri, err = self.validatedRedirect(self.Uri); err != nil {
		return err
	}

	if passin := ARGS.Get(role.Surface); passin != "" {
		if role.Surface == issuer.Credential[3] {
			if err := self.VerifyCookie(passin); err != nil {
				return err
			} else {
				self.SetCookie(role.Surface, passin, role.MaxAge)
				self.SetCookieSession(role.Surface+"_", passin)
				return Err(303, self.Uri)
			}
		}
	}

	login := ARGS.Get(issuer.Credential[0])
	password := ARGS.Get(issuer.Credential[1])
	if login != "" && !loginThrottleAllowed(&self.Base, self.Provider, login) {
		return Err(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests))
	}
	if err := self.CGI.Authenticate(login, password); err != nil {
		if login != "" {
			if throttleErr := loginThrottleFailure(&self.Base, self.Provider, login); throttleErr != nil {
				return throttleErr
			}
		}
		return err
	}
	if login != "" {
		loginThrottleSuccess(&self.Base, self.Provider, login)
	}

	return self.HandlerFields()
}

func (self *Ticket) validatedRedirect(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	safe, err := self.C.ValidateLocalRedirect(raw)
	if err != nil {
		return "", Err(http.StatusBadRequest, "invalid redirect")
	}
	return safe, nil
}

func (self *Ticket) GetAttributes() []string {
	role := self.C.Roles[self.RoleValue]
	fields := make([]string, len(role.Attributes))
	for i, v := range role.Attributes {
		if self.Out_hash[v] == nil {
			continue
		}
		out := Interface2String(self.Out_hash[v])
		if v == role.Id_name && role.Id_cipher {
			id64, _ := strconv.ParseInt(out, 10, 64)
			fields[i] = strconv.FormatInt(int64(int_cipher.Encrypt(uint(id64), self.C.Secret)), 10)
		} else {
			fields[i] = out
		}
	}
	return fields
}

func (self *Ticket) HandlerFields() error {
	c := self.C
	role := c.Roles[self.RoleValue]
	fields := self.GetAttributes()
	if fields[0] == "" {
		return Err(1032)
	}

	final_uri := self.Uri
	var err error
	if final_uri, err = self.validatedRedirect(final_uri); err != nil {
		return err
	}

	// how, variable name, value, base uri, others wholse values in redirect
	// how: (t, u, p)+(A, B, C)+(0,1)+(0,1), t:out_hash, u:go_uri, p:provider
	// for u and p, go_uri has to match values[1] (i.e. 2nd match)
	// A:equal, B:not equal, C:match,
	// 0: no stamp nor md5, 1: stamp and md5; 0:no coookie, 1:cookie
	c_uri := (role.Issuers[self.Provider]).ConditionURI
	if c_uri != nil {
		outs := self.Out_hash
		for _, values := range c_uri {
			how := values[0]
			obs := ""
			if how[0:1] == "t" {
				obs = outs[values[1]].(string)
			} else {
				if how[0:1] == "p" {
					obs = self.Provider
				} else {
					obs = final_uri
				}
				u, err := url.Parse(final_uri)
				if err != nil {
					continue
				}
				if len(u.Path) < len(values[1]) {
					continue
				}
				if (u.Path)[0:len(values[1])] != values[1] {
					continue
				}
			}
			if (how[1:2] == "A" && obs == values[2]) ||
				(how[1:2] == "B" && obs != values[2]) ||
				(how[1:2] == "C" && strings.Contains(obs, values[2])) {
				target, err := url.Parse(values[3])
				if err != nil {
					break
				}
				q := target.Query()
				for i := 4; i < len(values); i++ {
					if v, ok := outs[values[i]]; ok {
						q.Add(values[i], Interface2String(v))
					}
				}
				if how[2:3] == "1" {
					q.Add(c.GoStampName, fmt.Sprintf("%d", Unix_timestamp()))
					q.Add(c.GoMD5Name, SortMapMd5(c.Secret, self.C.GoMD5Name, q))
				}
				target.RawQuery = q.Encode()
				final_uri = target.RequestURI()
				if how[3:4] == "0" {
					self.Uri = final_uri
					return Err(303, final_uri)
				}
			}
		}
	}

	signed := self.Signature(fields...)
	self.SetCookie(role.Surface, signed, role.MaxAge)
	self.SetCookieSession(role.Surface+"_", signed)

	chartag, ok := c.Chartags[self.ChartagValue]
	if ok && chartag.Case > 0 {
		self.SendNocache(chartag.CallLogged())
		return nil
	}

	self.Uri = final_uri
	return Err(303, final_uri)
}

func (self *Ticket) Authenticate(login, password string) error {
	if login == "" || password == "" {
		return Err(1037)
	}
	role := self.C.Roles[self.RoleValue]
	issuer := role.Issuers[self.Provider]
	if login != issuer.ProviderPars["Def_login"] || password != issuer.ProviderPars["Def_password"] {
		return Err(1031)
	}

	role.Attributes = []string{"login", "provider"}
	self.Out_hash = map[string]interface{}{"login": issuer.ProviderPars["Def_login"], "provider": self.Provider}

	return nil
}
