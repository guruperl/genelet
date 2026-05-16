package genelet

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const oauthConsumerSecretKey = "oauth_consumer_" + "secret"

func Oauth1Sign(method string, uri string, hash map[string]string, items []string, combined []string, form url.Values) string {
	tmps := make([]string, len(items))
	copy(tmps, items)
	for k := range form {
		tmps = append(tmps, k)
	}
	sort.Strings(tmps)
	xItems := make([]string, len(tmps))
	for i, v := range tmps {
		x := hash[v]
		if x == "" && form != nil {
			x = form.Get(v)
		}
		xItems[i] = v + "%3D" + url.QueryEscape(x)
	}
	str := method + "&" + url.QueryEscape(uri) + "&" + strings.Join(xItems, "%26")
	key := combined[0] + "&"
	if len(combined) > 1 {
		key += combined[1]
	}

	return url.QueryEscape(Digest64(key, str))
}

func Oauth1Request(method string, uri string, hash map[string]string, items []string, combined []string, x_li_format string, form url.Values) ([]byte, error) {
	oauthSignature := Oauth1Sign(method, uri, hash, items, combined, form)

	xItems := make([]string, len(items))
	for i, key := range items {
		xItems[i] = key + "=\"" + hash[key] + "\""
	}
	h := make(map[string]string)
	h["Authorization"] = "OAuth oauth_signature=\"" + oauthSignature + "\", " + strings.Join(xItems, ", ")
	if x_li_format != "" {
		h["x-li-format"] = x_li_format
	}

	return Do(method, uri, form, h)
}

func get_body(method string, uri string, hash map[string]string, items []string, combined []string) (map[string]interface{}, error) {
	items = append(items, []string{"oauth_consumer_key", "oauth_nonce", "oauth_signature_method", "oauth_timestamp", "oauth_version"}...)
	body, err := Oauth1Request(method, uri, hash, items, combined, "", nil)
	if err != nil {
		return nil, err
	}

	back := make(map[string]interface{})
	a := strings.Split(string(body), "&")
	for _, v := range a {
		b := strings.Split(v, "=")
		back[b[0]] = b[1]
	}

	return back, nil
}

type Oauth1 struct {
	Procedure
	DefaultPars map[string]string
	Combined    []string
	X_li_format string
}

func NewOauth1(base Base, db *sql.DB, uri string, provider string) *Oauth1 {
	a := new(Oauth1)
	a.CGI = a
	a.Base = base
	a.DB = db
	a.Uri = uri
	a.Provider = provider
	a.DefaultPars = make(map[string]string)
	a.Combined = make([]string, 0)
	a.DefaultPars["oauth_signature_method"] = "HMAC-SHA1"
	a.DefaultPars["oauth_version"] = "1.0"
	switch provider {
	case "twitter":
		a.DefaultPars["oauth_request_token"] = "https://api.twitter.com/oauth/request_token"
		a.DefaultPars["oauth_authorize_uri"] = "https://api.twitter.com/oauth/authorize"
		a.DefaultPars["oauth_access_token"] = "https://api.twitter.com/oauth/access_token"
	//a.DefaultPars["oauth_endpoint"]			= "https://api.twitter.com/1.1/account/settings.json"
	case "linkedin":
		a.DefaultPars["oauth_request_token"] = "https://api.linkedin.com/uas/oauth/requestToken"
		a.DefaultPars["oauth_authorize_uri"] = "https://api.linkedin.com/uas/oauth/authenticate"
		a.DefaultPars["oauth_access_token"] = "https://api.linkedin.com/uas/oauth/accessToken"
		a.DefaultPars["oauth_endpoint"] = "https://api.linkedin.com/v1/people/~:(id,first-name,last-name,publicProfileUrl,pictureUrl)"
		a.DefaultPars["fields"] = "id,email,name,first_name,last_name,age_range,gender"
	}

	role := base.C.Roles[base.RoleValue]
	issuer := role.Issuers[provider]
	for k, v := range issuer.ProviderPars {
		a.DefaultPars[k] = v
	}

	return a
}

func (self *Oauth1) Authenticate(login, password string) error {
	role := self.C.Roles[self.RoleValue]
	hash := self.DefaultPars
	hash["oauth_callback"] = url.QueryEscape(self.Callback_address())

	now := int32(time.Now().Unix())
	hash["oauth_timestamp"] = fmt.Sprintf("%d", now)
	hash["oauth_nonce"] = fmt.Sprintf("%x%8x", os.Getpid(), now)

	if login == "" {
		self.Combined = []string{hash[oauthConsumerSecretKey]}
		back, err := get_body("GET", hash["oauth_request_token"], hash, []string{"oauth_callback"}, self.Combined)
		if err != nil {
			return err
		}
		if back["oauth_callback_confirmed"].(string) != "true" {
			return Err(404)
		}
		self.SetCookie(self.Provider, EncodeScoder(back["oauth_token_secret"].(string), role.Coding))
		return Err(303, hash["oauth_authorize_uri"]+"?"+"oauth_token="+back["oauth_token"].(string)+"&oauth_callback="+hash["oauth_callback"])
	}

	hash["oauth_token"] = login
	hash["oauth_verifier"] = password
	oauthTokenSecret, err := self.R.Cookie(self.Provider)
	if err != nil {
		return err
	}
	if oauthTokenSecret == nil {
		return Err(404)
	}

	hash["oauth_token_secret"] = oauthTokenSecret.Value
	hash["oauth_token_secret"] = DecodeScoder(hash["oauth_token_secret"], role.Coding)
	self.Combined = []string{hash[oauthConsumerSecretKey], hash["oauth_token_secret"]}
	back, err := get_body("GET", hash["oauth_access_token"], hash, []string{"oauth_token", "oauth_verifier"}, self.Combined)
	if err != nil {
		return err
	}
	for k, v := range back {
		hash[k] = v.(string)
	}
	// we get back oauth_token oauth_token_secret user_id screen_name x_auth_expires
	// need to re-assigned to Combined
	self.Combined = []string{hash[oauthConsumerSecretKey], hash["oauth_token_secret"]}

	if hash["oauth_endpoint"] != "" {
		back1, err := self.Oauth1_api("GET", hash["oauth_endpoint"], nil)
		if err != nil {
			return err
		}
		for k, v := range back1 {
			back[k] = v
		}
	}

	for k, v := range hash {
		back[k] = v
	}

	// oauth_token oauth_token_secret user_id screen_name x_auth_expires
	// OAuth consumer key and secret are already present.
	return self.Fill_provider(back)
}

func (self *Oauth1) oauth1Request(method string, uri string, form url.Values) ([]byte, error) {
	hash := self.DefaultPars
	now := int32(time.Now().Unix())
	hash["oauth_timestamp"] = fmt.Sprintf("%d", now)
	hash["oauth_nonce"] = fmt.Sprintf("%x%8x", os.Getpid(), now)
	items := []string{"oauth_consumer_key", "oauth_nonce", "oauth_signature_method", "oauth_token", "oauth_timestamp", "oauth_version"}
	return Oauth1Request(method, uri, hash, items, self.Combined, "json", form)
}

func (self *Oauth1) Oauth1_api(method string, uri string, form url.Values) (map[string]interface{}, error) {
	body, err := self.oauth1Request(method, uri, form)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func (self *Oauth1) Oauth1_apis(method string, uri string, form url.Values) ([]map[string]interface{}, error) {
	body, err := self.oauth1Request(method, uri, form)
	if err != nil {
		return nil, err
	}

	return To_slice(body)
}
