package genelet

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type Oauth2 struct {
	Procedure
	DefaultPars map[string]string
	AccessToken string
}

func NewOauth2(base Base, db *sql.DB, uri string, provider string) *Oauth2 {
	a := new(Oauth2)
	a.CGI = a
	a.Base = base
	a.DB = db
	a.Uri = uri
	a.Provider = provider
	a.DefaultPars = make(map[string]string)
	//a.DefaultPars["endpoint"]       = "https://api.linkedin.com/v2/me"
	switch provider {
	case "google":
		a.DefaultPars["scope"] = "profile"
		a.DefaultPars["response_type"] = "code"
		a.DefaultPars["grant_type"] = "authorization_code"
		a.DefaultPars["authorize_url"] = "https://accounts.google.com/o/oauth2/auth"
		a.DefaultPars["access_token_url"] = "https://accounts.google.com/o/oauth2/token"
		a.DefaultPars["endpoint"] = "https://www.googleapis.com/oauth2/v1/userinfo"
	case "facebook":
		a.DefaultPars["scope"] = "public_profile%20email"
		a.DefaultPars["authorize_url"] = "https://www.facebook.com/dialog/oauth"
		a.DefaultPars["access_token_url"] = "https://graph.facebook.com/oauth/access_token"
		a.DefaultPars["endpoint"] = "https://graph.facebook.com/me"
		a.DefaultPars["fields"] = "id,email,name,first_name,last_name,age_range,gender"
	case "linkedin":
		a.DefaultPars["scope"] = "r_basicprofile"
		a.DefaultPars["authorize_url"] = "https://www.linkedin.com/oauth/v2/authorization"
		a.DefaultPars["access_token_url"] = "https://www.linkedin.com/oauth/v2/accessToken"
		a.DefaultPars["grant_type"] = "authorization_code"
		a.DefaultPars["endpoint"] = "https://api.linkedin.com/v1/people/~"
	case "qq":
		a.DefaultPars["scope"] = "get_user_info"
		a.DefaultPars["authorize_url"] = "https://graph.qq.com/oauth2.0/authorize"
		a.DefaultPars["access_token_url"] = "https://graph.qq.com/oauth2.0/token"
		a.DefaultPars["grant_type"] = "authorization_code"
		a.DefaultPars["endpoint"] = "https://graph.qq.com/user/get_user_info"
		a.DefaultPars["fields"] = "nickname, gender"
	case "microsoft":
		a.DefaultPars["response_type"] = "code"
		a.DefaultPars["scope"] = "wl.basic%20wl.offline_access%20wl.emails%20wl.skydrive"
		a.DefaultPars["authorize_url"] = "https://oauth.live.com/authorize"
		a.DefaultPars["access_token_url"] = "https://oauth.live.com/token"
		a.DefaultPars["grant_type"] = "authorization_code"
		a.DefaultPars["token_method_get"] = "1"
		a.DefaultPars["endpoint"] = "https://apis.live.net/v5.0/me"
	case "salesforce":
		a.DefaultPars["response_type"] = "code"
		a.DefaultPars["grant_typ"] = "authorization_code"
		a.DefaultPars["authorize_url"] = "https://login.salesforce.com/services/oauth2/authorize"
		a.DefaultPars["access_token_url"] = "https://login.salesforce.com/services/oauth2/token"
		a.DefaultPars["endpoint"] = "https://login.salesforce.com/id/"
	}

	role := base.C.Roles[base.RoleValue]
	issuer := role.Issuers[provider]
	for k, v := range issuer.ProviderPars {
		a.DefaultPars[k] = v
	}

	return a
}

func (self *Oauth2) Authenticate(login, password string) error {
	defaults := self.DefaultPars
	cbk := self.Callback_address()
	if login == "" {
		if password != "" {
			return Err(400)
		}
		state, err := oauthState()
		if err != nil {
			return err
		}
		defaults["state"] = state
		self.setStateCookie(state, 600)
		dest := defaults["authorize_url"] + "?client_id=" + defaults["client_id"] + "&redirect_uri=" + url.QueryEscape(cbk)
		for _, k := range []string{"scope", "display", "state", "response_type", "access_type", "approval_prompt"} {
			if v, ok := defaults[k]; ok {
				dest += "&" + k + "=" + v
			}
		}
		return Err(303, dest)
	}
	if err := self.verifyState(); err != nil {
		return err
	}

	form := make(url.Values)
	form.Set("code", login)
	form.Set("client_id", defaults["client_id"])
	form.Set("client_secret", defaults["client_secret"])
	form.Set("redirect_uri", cbk)
	if defaults["grant_type"] != "" {
		form.Set("grant_type", defaults["grant_type"])
	}

	var res *http.Response
	var err error
	if _, ok := defaults["token_method_get"]; ok {
		res, err = HTTPClient.Get(defaults["access_token_url"] + "?" + form.Encode())
	} else {
		res, err = HTTPClient.PostForm(defaults["access_token_url"], form)
	}
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return Err(res.StatusCode)
	}

	back := make(map[string]interface{})
	switch self.Provider {
	case "facebook":
		m, err := url.ParseQuery(string(body))
		if err != nil {
			return Err(1400)
		}
		back["access_token"] = m.Get("access_token")
		back["expires"] = m.Get("expires")
	default:
		err := json.Unmarshal(body, &back)
		if err != nil {
			return err
		}
	}
	if accessToken, ok := back["access_token"]; ok {
		self.AccessToken = accessToken.(string)
	} else {
		return Err(1401)
	}

	if endpoint, ok := defaults["endpoint"]; ok {
		form = make(url.Values)
		for k, v := range back {
			if k == "access_token" {
				continue
			}
			form.Set(k, Interface2String(v))
		}
		h := make(map[string]string)
		if self.Provider == "salesforce" {
			endpoint = form.Get("id")
		}
		if self.Provider == "facebook" {
			form.Set("fields", defaults["fields"])
		}
		if self.Provider == "linkedin" {
			h["x-li-format"] = "json"
		}
		if back1, err := self.Oauth2_api("GET", endpoint, form, h); err == nil {
			for k, v := range back1 {
				back[k] = v
			}
		} else {
			return err
		}
	}
	for k, v := range defaults {
		back[k] = v
	}

	return self.Fill_provider(back)
}

func oauthState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (self *Oauth2) stateCookieName() string {
	return "_goauth2_state_" + self.RoleValue + "_" + self.Provider
}

func (self *Oauth2) setStateCookie(value string, maxAge int) {
	http.SetCookie(self.W, &http.Cookie{
		Name:     self.stateCookieName(),
		Value:    value,
		Path:     self.C.Script + "/" + self.RoleValue + "/" + self.ChartagValue + "/" + self.Provider,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   self.R.TLS != nil,
	})
}

func (self *Oauth2) verifyState() error {
	got := self.R.Form.Get("state")
	cookie, err := self.R.Cookie(self.stateCookieName())
	self.setStateCookie("", -1)
	if err != nil || got == "" || cookie.Value == "" || got != cookie.Value {
		return Err(400, "invalid oauth2 state")
	}
	return nil
}

func (self *Oauth2) oauth2Request(method string, uri string, form url.Values, h map[string]string) ([]byte, error) {
	if h == nil {
		h = make(map[string]string)
	}
	if self.DefaultPars["grant_type"] == "authorization_code" {
		h["Authorization"] = "Bearer " + self.AccessToken
		return Do(method, uri, nil, h)
	}
	if form == nil {
		form = make(url.Values)
	}
	form.Set("access_token", self.AccessToken)
	return Do(method, uri, form, h)
}

func (self *Oauth2) Oauth2_api(method string, uri string, form url.Values, h map[string]string) (map[string]interface{}, error) {
	body, err := self.oauth2Request(method, uri, form, h)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func (self *Oauth2) Oauth2_apis(method string, uri string, form url.Values, h map[string]string) ([]map[string]interface{}, error) {
	body, err := self.oauth2Request(method, uri, form, h)
	if err != nil {
		return nil, err
	}

	return To_slice(body)
}

func (self *Oauth2) multipart_request(uri string, form url.Values, paramName, path string) ([]byte, error) {
	hash := self.DefaultPars
	if hash["grant_type"] == "authorization_code" {
		h := make(map[string]string)
		h["Authorization"] = "Bearer " + self.AccessToken
		return MultipartUpload(uri, form, h, paramName, path)
	}

	if form == nil {
		form = make(url.Values)
	}
	form.Set("access_token", self.AccessToken)
	return MultipartUpload(uri, form, nil, paramName, path)
}

func (self *Oauth2) Multipart_upload(uri string, form url.Values, paramName, path string) (map[string]interface{}, error) {
	body, err := self.multipart_request(uri, form, paramName, path)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func (self *Oauth2) Multipart_uploads(uri string, form url.Values, paramName, path string) ([]map[string]interface{}, error) {
	body, err := self.multipart_request(uri, form, paramName, path)
	if err != nil {
		return nil, err
	}

	return To_slice(body)
}
