package genelet

import (
	"crypto/subtle"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const csrfTokenTTL = 24 * time.Hour

var geneletForwardedHeaders = []string{
	"X-Forwarded-ID",
	"X-Forwarded-Time",
	"X-Forwarded-User",
	"X-Forwarded-Group",
	"X-Forwarded-Raw",
	"X-Forwarded-Hash",
	"X-Forwarded-Duration",
	"X-Forwarded-Request-Time",
	"X-Forwarded-Request_time",
}

func scrubGeneletForwardedHeaders(h http.Header) {
	for _, name := range geneletForwardedHeaders {
		h.Del(name)
	}
}

func (c *Config) uploadLimit() int64 {
	if c == nil || c.UploadMaxBytes <= 0 {
		return 32 << 20
	}
	return c.UploadMaxBytes
}

func (c *Config) csrfName() string {
	if c == nil || c.CSRFName == "" {
		return "go_csrf"
	}
	return c.CSRFName
}

func (c *Config) requestTimeout() time.Duration {
	if c == nil || c.RequestTimeoutSeconds <= 0 {
		return 10 * time.Second
	}
	return time.Duration(c.RequestTimeoutSeconds) * time.Second
}

func (self *Base) secureCookie() bool {
	if self == nil || self.R == nil {
		return false
	}
	if self.R.TLS != nil {
		return true
	}
	if self.C != nil {
		u, err := url.Parse(self.C.ServerURL)
		if err == nil && strings.EqualFold(u.Scheme, "https") {
			return true
		}
	}
	return false
}

func (self *Base) csrfSecret() string {
	if self == nil || self.C == nil {
		return "genelet-csrf"
	}
	if role, ok := self.C.Roles[self.RoleValue]; ok && role.Secret != "" {
		return role.Secret
	}
	if self.C.Secret != "" {
		return self.C.Secret
	}
	return "genelet-csrf"
}

func (self *Base) csrfSubject() string {
	if self == nil || self.R == nil {
		return ""
	}
	if raw := self.R.Header.Get("X-Forwarded-Raw"); raw != "" {
		return raw
	}
	if self.C != nil {
		if role, ok := self.C.Roles[self.RoleValue]; ok && role.Surface != "" {
			if coo, err := self.R.Cookie(role.Surface); err == nil {
				return coo.Value
			}
		}
	}
	return self.RoleValue + "|" + self.ChartagValue + "|" + self.GetIP()
}

func (self *Base) CSRFToken() string {
	when := strconv.FormatInt(time.Now().Unix(), 10)
	return when + ":" + Digest(self.csrfSecret(), when, self.RoleValue, self.ChartagValue, self.csrfSubject())
}

func (self *Base) CSRFInput() template.HTML {
	if self == nil || self.C == nil {
		return ""
	}
	return template.HTML(`<input type="hidden" name="` + template.HTMLEscapeString(self.C.csrfName()) + `" value="` + template.HTMLEscapeString(self.CSRFToken()) + `">`)
}

func (self *Base) ValidateCSRF() error {
	if self == nil || self.C == nil || self.R == nil {
		return Err(http.StatusForbidden)
	}
	token := self.R.Header.Get("X-CSRF-Token")
	if token == "" {
		token = self.R.Header.Get("X-CSRF")
	}
	if token == "" {
		token = self.R.Form.Get(self.C.csrfName())
	}
	parts := strings.Split(token, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Err(http.StatusForbidden, "invalid csrf token")
	}
	when, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Err(http.StatusForbidden, "invalid csrf token")
	}
	age := time.Since(time.Unix(when, 0))
	if age < 0 || age > csrfTokenTTL {
		return Err(http.StatusForbidden, "expired csrf token")
	}
	want := Digest(self.csrfSecret(), parts[0], self.RoleValue, self.ChartagValue, self.csrfSubject())
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(want)) != 1 {
		return Err(http.StatusForbidden, "invalid csrf token")
	}
	return nil
}

func (c *Config) ValidateLocalRedirect(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if unescaped, err := url.QueryUnescape(raw); err == nil && strings.HasPrefix(unescaped, "/") {
		raw = unescaped
	}
	if strings.Contains(raw, `\`) {
		return "", fmt.Errorf("redirect contains a backslash")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "" || u.Host != "" {
		return "", fmt.Errorf("redirect must be local")
	}
	if !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return "", fmt.Errorf("redirect must be a single-slash local path")
	}
	script := ""
	if c != nil {
		script = c.Script
	}
	if script != "" {
		if !strings.HasPrefix(script, "/") {
			script = "/" + script
		}
		if u.Path != script && !strings.HasPrefix(u.Path, script+"/") {
			return "", fmt.Errorf("redirect must stay under %s", script)
		}
	}
	return u.RequestURI(), nil
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func actionRequiresPost(action string) bool {
	switch action {
	case "insert", "update", "insupd", "replace", "delete", "login_as", "upload", "approve", "deleteBidder", "deleteTarget":
		return true
	default:
		return strings.HasPrefix(action, "delete") || strings.HasPrefix(action, "update")
	}
}
