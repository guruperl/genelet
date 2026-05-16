package genelet

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOauth2RejectsMissingOrMismatchedState(t *testing.T) {
	config := &Config{
		Script:    "/goto",
		GoURIName: "go_uri",
		Roles: map[string]Role{"m": {
			Issuers: map[string]Issuer{"google": {
				ProviderPars: map[string]string{
					"client_id":        "client",
					"client_secret":    "secret",
					"authorize_url":    "https://auth.example.test",
					"access_token_url": "https://token.example.test",
				},
			}},
		}},
	}
	startReq := httptest.NewRequest(http.MethodGet, "http://example.test/goto/m/json/google", nil)
	startReq.Form = url.Values{}
	startW := httptest.NewRecorder()
	ticket := NewOauth2(Base{C: config, W: startW, R: startReq, RoleValue: "m", ChartagValue: "json"}, nil, "/target", "google")

	err := ticket.Authenticate("", "")
	gerr, ok := err.(Gerror)
	if !ok || gerr.Code != http.StatusSeeOther {
		t.Fatalf("Authenticate start error = %#v, want 303 Gerror", err)
	}
	if !strings.Contains(gerr.Errstr, "state=") {
		t.Fatalf("redirect URL missing state: %s", gerr.Errstr)
	}
	cookies := startW.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("state cookie = %#v, want HttpOnly SameSite=Lax cookie", cookies)
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "http://example.test/goto/m/json/google?state=wrong", nil)
	callbackReq.Form = url.Values{"state": {"wrong"}}
	callbackReq.AddCookie(cookies[0])
	callbackW := httptest.NewRecorder()
	callback := NewOauth2(Base{C: config, W: callbackW, R: callbackReq, RoleValue: "m", ChartagValue: "json"}, nil, "/target", "google")

	err = callback.Authenticate("code", "")
	gerr, ok = err.(Gerror)
	if !ok || gerr.Code != http.StatusBadRequest {
		t.Fatalf("Authenticate callback error = %#v, want 400 Gerror", err)
	}
	expired := callbackW.Result().Cookies()
	if len(expired) != 1 || expired[0].MaxAge >= 0 {
		t.Fatalf("expired state cookie = %#v, want MaxAge < 0", expired)
	}
}
