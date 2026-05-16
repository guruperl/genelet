package genelet

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestControllerCORS(t *testing.T) {
	controller := &Controller{
		C: &Config{
			ServerURL:   "http://admin.example.test:8080",
			CORSOrigins: []string{"https://ops.example.test"},
		},
		Logger: zap.NewNop(),
	}

	tests := []struct {
		name         string
		origin       string
		wantStatus   int
		wantAllow    string
		wantVary     string
		wantCreds    string
		wantNoAccess bool
	}{
		{
			name:       "server url origin",
			origin:     "http://admin.example.test:8080",
			wantStatus: http.StatusOK,
			wantAllow:  "http://admin.example.test:8080",
			wantVary:   "Origin",
			wantCreds:  "true",
		},
		{
			name:       "configured origin",
			origin:     "https://ops.example.test",
			wantStatus: http.StatusOK,
			wantAllow:  "https://ops.example.test",
			wantVary:   "Origin",
			wantCreds:  "true",
		},
		{
			name:         "rejected origin",
			origin:       "https://evil.example.test",
			wantStatus:   http.StatusForbidden,
			wantVary:     "Origin",
			wantNoAccess: true,
		},
		{
			name:       "no origin",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/goto/adv/json/login", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			}
			w := httptest.NewRecorder()

			controller.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Header().Get("Vary"); got != tt.wantVary {
				t.Fatalf("Vary = %q, want %q", got, tt.wantVary)
			}
			if tt.wantNoAccess {
				if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
					t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
				}
				if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
					t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", got)
				}
				return
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllow {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantAllow)
			}
			if got := w.Header().Get("Access-Control-Allow-Credentials"); got != tt.wantCreds {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", got, tt.wantCreds)
			}
		})
	}
}

type securityModel struct{}

func (m *securityModel) SetDefaults(url.Values, *[]map[string]interface{}, *map[string]interface{}, map[string]interface{}) {
}

func (m *securityModel) SetDB(any interface{}) {}

type securityFilter struct {
	action string
}

func (f *securityFilter) SetAll(_ Base, action string, _ string, _ *map[string]interface{}) {
	f.action = action
}

func (f *securityFilter) GetAll() (map[string][]string, []string) {
	return map[string][]string{
		"groups":  {"adv", "admin"},
		"options": {"no_db", "no_method"},
	}, nil
}

func (f *securityFilter) Preset() error { return nil }

func (f *securityFilter) Before(model *securityModel, extra url.Values, nextextra url.Values) error {
	return nil
}

func (f *securityFilter) After(model *securityModel) error { return nil }

func securityController() *Controller {
	return NewController(&Config{
		ActionName: "action",
		DefaultActions: map[string]string{
			http.MethodGet:    "topics",
			http.MethodPost:   "insert",
			http.MethodDelete: "delete",
		},
		Script:      "/goto",
		GoProbeName: "go_probe",
		CSRFName:    "go_csrf",
		Secret:      "app-secret",
		Blks:        map[string]map[string]string{},
		Chartags:    map[string]Chartag{"json": {Case: 1, ContentType: "application/json"}},
		Roles: map[string]Role{
			"adv": {
				Id_name:    "adv_id",
				Attributes: []string{"adv_id", "agency_id", "team_id"},
				Surface:    "mc",
				Secret:     "role-secret",
				Coding:     "role-coding",
				Duration:   3600,
				MaxAge:     3600,
			},
			"admin": {
				Is_admin:   true,
				Id_name:    "admin_id",
				Attributes: []string{"admin_id"},
				Surface:    "ac",
				Secret:     "admin-secret",
				Coding:     "admin-coding",
				Duration:   3600,
			},
		},
	}, nil, zap.NewNop())
}

func TestControllerScrubsUntrustedForwardedHeaders(t *testing.T) {
	controller := securityController()
	req := httptest.NewRequest(http.MethodGet, "/goto/adv/json/thing", nil)
	req.Header.Set("X-Forwarded-User", "999")
	req.Header.Set("X-Forwarded-Group", "evil|evil")
	req.Header.Set("X-Forwarded-Time", "999")
	req.Header.Set("X-Forwarded-Duration", "999")
	signer := NewAccess(Base{C: controller.C, W: httptest.NewRecorder(), R: req, RoleValue: "adv", ChartagValue: "json"})
	req.AddCookie(&http.Cookie{Name: "mc", Value: signer.Signature("1", "10", "20")})
	controller.ModelFactories["thing"] = func() interface{} { return &securityModel{} }
	controller.FilterFactories["thing"] = func() interface{} { return &securityFilter{} }

	w := httptest.NewRecorder()
	controller.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "999") || strings.Contains(body, "evil") {
		t.Fatalf("response used spoofed forwarded headers: %s", body)
	}
	if !strings.Contains(body, `"adv_id":["1"]`) || !strings.Contains(body, `"agency_id":["10"]`) || !strings.Contains(body, `"team_id":["20"]`) {
		t.Fatalf("response missing verified cookie identity: %s", body)
	}
}

func TestLoginPageRejectsExternalGoURI(t *testing.T) {
	controller := securityController()
	controller.C.GoURIName = "go_uri"
	controller.C.ProviderName = "provider"
	controller.C.LoginName = "login"
	controller.C.Roles["adv"] = Role{
		Attributes: []string{"adv_id"},
		Surface:    "mc",
		Secret:     "role-secret",
		Coding:     "role-coding",
		Issuers: map[string]Issuer{"db": {
			Default:      true,
			Credential:   []string{"login", "password", "direct", "mc"},
			ProviderPars: map[string]string{"Def_login": "u", "Def_password": "p"},
		}},
	}
	req := httptest.NewRequest(http.MethodGet, "/goto/adv/json/login?go_uri=https://evil.example.test/", nil)
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	controller.loginPage(&Base{C: controller.C, W: w, R: req, RoleValue: "adv", ChartagValue: "json"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRequiresCSRFForMutatingMethods(t *testing.T) {
	controller := securityController()
	controller.ModelFactories["thing"] = func() interface{} { return &securityModel{} }
	controller.FilterFactories["thing"] = func() interface{} { return &securityFilter{} }
	getReq := httptest.NewRequest(http.MethodGet, "/goto/admin/json/thing?action=insert", nil)
	getReq.Form = url.Values{"action": {"insert"}}
	getReq.Header.Set("X-Forwarded-User", "1")
	getBase := Base{C: controller.C, W: httptest.NewRecorder(), R: getReq, RoleValue: "admin", ChartagValue: "json"}
	err := controller.Handle("thing", getBase, http.MethodGet)
	if gerr, ok := err.(Gerror); !ok || gerr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET insert = %#v, want 405 Gerror", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/goto/admin/json/thing", nil)
	req.Form = url.Values{}
	req.Header.Set("X-Forwarded-User", "1")
	base := Base{C: controller.C, W: httptest.NewRecorder(), R: req, RoleValue: "admin", ChartagValue: "json"}

	err = controller.Handle("thing", base, http.MethodPost)
	if gerr, ok := err.(Gerror); !ok || gerr.Code != http.StatusForbidden {
		t.Fatalf("Handle missing CSRF = %#v, want 403 Gerror", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/goto/admin/json/thing", nil)
	req.Form = url.Values{}
	req.Header.Set("X-Forwarded-User", "1")
	base = Base{C: controller.C, W: httptest.NewRecorder(), R: req, RoleValue: "admin", ChartagValue: "json"}
	req.Header.Set("X-CSRF-Token", base.CSRFToken())
	if err := controller.Handle("thing", base, http.MethodPost); err != nil {
		t.Fatalf("Handle with CSRF returned error: %v", err)
	}
}

func TestHandleRejectsMissingCSRFBeforeAppHooks(t *testing.T) {
	controller := securityController()
	factoryCalled := false
	filterFactoryCalled := false
	controller.ModelFactories["thing"] = func() interface{} {
		factoryCalled = true
		return &securityModel{}
	}
	controller.FilterFactories["thing"] = func() interface{} {
		filterFactoryCalled = true
		return &securityFilter{}
	}
	req := httptest.NewRequest(http.MethodPost, "/goto/admin/json/thing", nil)
	req.Form = url.Values{}
	req.Header.Set("X-Forwarded-User", "1")
	base := Base{C: controller.C, W: httptest.NewRecorder(), R: req, RoleValue: "admin", ChartagValue: "json"}

	err := controller.Handle("thing", base, http.MethodPost)
	if gerr, ok := err.(Gerror); !ok || gerr.Code != http.StatusForbidden {
		t.Fatalf("Handle missing CSRF = %#v, want 403 Gerror", err)
	}
	if factoryCalled || filterFactoryCalled {
		t.Fatalf("app hooks/factories ran before CSRF rejection: model=%v filter=%v", factoryCalled, filterFactoryCalled)
	}
}

type factoryRaceModel struct {
	args url.Values
}

func (m *factoryRaceModel) SetDefaults(args url.Values, lists *[]map[string]interface{}, other *map[string]interface{}, storage map[string]interface{}) {
	m.args = args
}

func (m *factoryRaceModel) SetDB(any interface{}) {}

func (m *factoryRaceModel) Topics(extra url.Values, nextextra url.Values) error {
	m.args.Set("_seen", m.args.Get("req"))
	return nil
}

type factoryRaceFilter struct {
	Filter
}

func (f *factoryRaceFilter) GetAll() (map[string][]string, []string) {
	return map[string][]string{"groups": []string{"admin"}}, nil
}

func (f *factoryRaceFilter) Before(model *factoryRaceModel, extra url.Values, nextextra url.Values) error {
	return nil
}

func (f *factoryRaceFilter) After(model *factoryRaceModel) error {
	return nil
}

func TestControllerFactoriesUseRequestLocalInstances(t *testing.T) {
	controller := &Controller{
		C: &Config{
			ActionName:     "action",
			DefaultActions: map[string]string{http.MethodGet: "topics"},
			Script:         "/goto",
			Blks:           map[string]map[string]string{},
			Chartags:       map[string]Chartag{"json": {Case: 1, ContentType: "application/json"}},
			Roles: map[string]Role{
				"admin": {Is_admin: true, Id_name: "admin_id", Attributes: []string{"admin_id"}},
			},
		},
		ModelFactories:  map[string]func() interface{}{"thing": func() interface{} { return &factoryRaceModel{} }},
		FilterFactories: map[string]func() interface{}{"thing": func() interface{} { return &factoryRaceFilter{} }},
		Storage:         map[string]interface{}{},
		Logger:          zap.NewNop(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/goto/admin/json/thing", nil)
			req.Form = url.Values{"req": {string(rune('a' + i%26))}}
			req.Header.Set("X-Forwarded-User", "1")
			base := Base{C: controller.C, W: httptest.NewRecorder(), R: req, RoleValue: "admin", ChartagValue: "json"}
			if err := controller.Handle("thing", base, http.MethodGet); err != nil {
				t.Errorf("Handle returned error: %v", err)
			}
			if got := req.Form.Get("_seen"); got != req.Form.Get("req") {
				t.Errorf("_seen = %q, want %q", got, req.Form.Get("req"))
			}
		}()
	}
	wg.Wait()
}

func TestStaticPathRejectionReturnsBeforeServeFile(t *testing.T) {
	root := t.TempDir()
	documentRoot := filepath.Join(root, "www")
	if err := os.Mkdir(documentRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("secret content"), 0600); err != nil {
		t.Fatal(err)
	}

	controller := &Controller{C: &Config{DocumentRoot: documentRoot}}
	req := httptest.NewRequest(http.MethodGet, "/../secret.txt", nil)
	w := httptest.NewRecorder()

	controller.staticPage(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if body := w.Body.String(); body == "secret content" {
		t.Fatal("staticPage served a rejected parent path")
	}
}

type dispatchTestModel struct{}

func (m *dispatchTestModel) SetDefaults(url.Values, *[]map[string]interface{}, *map[string]interface{}, map[string]interface{}) {
}

func (m *dispatchTestModel) SetDB(any interface{}) {}

type dispatchTestFilter struct {
	actionHash map[string]map[string][]string
}

func (f *dispatchTestFilter) SetAll(Base, string, string, *map[string]interface{}) {}

func (f *dispatchTestFilter) GetAll() (map[string][]string, []string) {
	return f.actionHash["topics"], nil
}

func (f *dispatchTestFilter) Preset() error { return nil }

func (f *dispatchTestFilter) Before(model int, extra url.Values, nextextra url.Values) error {
	return nil
}

func (f *dispatchTestFilter) After(model *dispatchTestModel) error { return nil }

func TestControllerDispatchReturnsFrameworkError(t *testing.T) {
	controller := &Controller{
		C: &Config{
			ActionName:     "action",
			DefaultActions: map[string]string{http.MethodGet: "topics"},
			Script:         "/goto",
			Roles: map[string]Role{
				"admin": {Is_admin: true, Id_name: "admin_id", Attributes: []string{"admin_id"}},
			},
		},
		Models: map[string]interface{}{"thing": &dispatchTestModel{}},
		Filters: map[string]interface{}{"thing": &dispatchTestFilter{actionHash: map[string]map[string][]string{
			"topics": {"groups": []string{"admin"}},
		}}},
		Storage: map[string]interface{}{},
		Logger:  zap.NewNop(),
	}
	req := httptest.NewRequest(http.MethodGet, "/goto/admin/json/thing", nil)
	req.Form = url.Values{}
	req.Header.Set("X-Forwarded-User", "1")
	req.Header.Set("X-Forwarded-Group", "")
	req.Header.Set("X-Forwarded-Time", "1")
	req.Header.Set("X-Forwarded-Duration", "3600")
	base := Base{C: controller.C, W: httptest.NewRecorder(), R: req, RoleValue: "admin", ChartagValue: "json"}

	err := controller.Handle("thing", base, http.MethodGet)
	if err == nil {
		t.Fatal("Handle returned nil error for wrong Before signature and missing model action")
	}
	if _, ok := err.(Gerror); !ok {
		t.Fatalf("Handle returned %T, want Gerror", err)
	}
}

type groupFilter struct{}

func (f *groupFilter) SetAll(Base, string, string, *map[string]interface{}) {}

func (f *groupFilter) GetAll() (map[string][]string, []string) {
	return map[string][]string{"groups": []string{"adv"}, "options": []string{"no_db", "no_method"}}, nil
}

func TestForwardedGroupMismatchReturnsError(t *testing.T) {
	controller := &Controller{
		C: &Config{
			ActionName:     "action",
			DefaultActions: map[string]string{http.MethodGet: "topics"},
			Script:         "/goto",
			Roles: map[string]Role{
				"adv": {Id_name: "adv_id", Attributes: []string{"adv_id", "agency_id", "team_id"}},
			},
		},
		Models:  map[string]interface{}{"thing": &dispatchTestModel{}},
		Filters: map[string]interface{}{"thing": &groupFilter{}},
		Storage: map[string]interface{}{},
		Logger:  zap.NewNop(),
	}
	req := httptest.NewRequest(http.MethodGet, "/goto/adv/json/thing", nil)
	req.Form = url.Values{}
	req.Header.Set("X-Forwarded-User", "1")
	req.Header.Set("X-Forwarded-Group", "10")
	base := Base{C: controller.C, W: httptest.NewRecorder(), R: req, RoleValue: "adv", ChartagValue: "json"}

	err := controller.Handle("thing", base, http.MethodGet)
	if err == nil {
		t.Fatal("Handle returned nil error for mismatched forwarded group count")
	}
	gerr, ok := err.(Gerror)
	if !ok || gerr.Code != http.StatusUnauthorized {
		t.Fatalf("Handle error = %#v, want 401 Gerror", err)
	}
}
