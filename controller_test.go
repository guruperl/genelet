package genelet

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
