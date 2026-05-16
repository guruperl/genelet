package genelet

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	int_cipher "github.com/delongw/go-int-cipher"
	"go.uber.org/zap"
)

type Controller struct {
	C                *Config
	DB               *sql.DB
	DBDriver         string
	Models           map[string]interface{}
	Filters          map[string]interface{}
	Storage          map[string]interface{}
	ModelFactories   map[string]func() interface{}
	FilterFactories  map[string]func() interface{}
	StorageFactories map[string]func() interface{}
	Logger           *zap.Logger
}

func NewController(config *Config, db *sql.DB, logger ...*zap.Logger) *Controller {
	c := &Controller{
		C:                config,
		DB:               db,
		DBDriver:         config.DriverName(),
		Storage:          map[string]interface{}{},
		ModelFactories:   map[string]func() interface{}{},
		FilterFactories:  map[string]func() interface{}{},
		StorageFactories: map[string]func() interface{}{},
	}
	if len(logger) > 0 {
		c.Logger = logger[0]
	}
	c.ensureDefaults()
	return c
}

func (self *Controller) ensureDefaults() {
	if self.Logger == nil {
		self.Logger = zap.NewNop()
	}
}

func (self *Controller) dbDriver() string {
	if self.DBDriver != "" {
		return self.DBDriver
	}
	if self.C != nil {
		return self.C.DriverName()
	}
	return ""
}

func (self *Controller) staticPage(w http.ResponseWriter, r *http.Request) {
	if self.C == nil || self.C.DocumentRoot == "" {
		http.NotFound(w, r)
		return
	}
	if strings.Contains(r.URL.Path, "..") {
		http.NotFound(w, r)
		return
	}
	root, err := filepath.Abs(self.C.DocumentRoot)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cleanPath := filepath.Clean("/" + r.URL.Path)
	target, err := filepath.Abs(filepath.Join(root, strings.TrimPrefix(cleanPath, "/")))
	if err != nil || (target != root && !strings.HasPrefix(target, root+string(filepath.Separator))) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, target)
}

func (self *Controller) corsAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	c := self.C
	if c == nil {
		return false
	}
	if server, err := url.Parse(c.ServerURL); err == nil && server.Scheme != "" && server.Host != "" {
		if origin == server.Scheme+"://"+server.Host {
			return true
		}
	} else if origin == c.ServerURL {
		return true
	}
	for _, allowed := range c.CORSOrigins {
		if allowed != "" && origin == allowed {
			return true
		}
	}
	return false
}

func (self *Controller) loginPage(base *Base) {
	self.ensureDefaults()
	c := self.C
	uri := base.R.Form.Get(c.GoURIName)
	if uri == "" {
		if found, err := base.R.Cookie(c.GoProbeName); err == nil {
			uri = found.Value
		}
	}
	safeURI, err := c.ValidateLocalRedirect(uri)
	if err != nil {
		base.SendStatusPage(http.StatusBadRequest, "Bad Request")
		return
	}
	uri = safeURI
	base.R.Form.Set(c.GoURIName, uri)
	glog := self.Logger.Sugar()

	provider := base.R.Form.Get(c.ProviderName)
	glog.Infof("provider %s", provider)
	if provider == "" {
		provider = base.GetProvider()
		if provider == "" {
			http.NotFound(base.W, base.R)
			return
		}
	}

	db := self.DB

	err = nil
	if Grep(c.Oauth2s, provider) {
		ticket := NewOauth2(*base, db, uri, provider)
		glog.Infof("%s %s", "oauth2 uses: ", provider)
		err = ticket.Handler_login()
		uri = ticket.Uri // use the same vriable for the targeting uri
	} else if Grep(c.Oauth1s, provider) {
		ticket := NewOauth1(*base, db, uri, provider)
		glog.Infof("%s %s", "oauth1 uses: ", provider)
		err = ticket.Handler_login()
		uri = ticket.Uri // use the same vriable for the targeting uri
	} else {
		glog.Infof("%s %s", "login uses: ", provider)
		ticket := NewProcedure(*base, db, uri, provider)
		err = ticket.Handler()
		uri = ticket.Uri // use the same vriable for the targeting uri
	}
	if err == nil {
		return
	}
	if base.ChartagValue == "json" {
		base.SendPage(c.Chartags[base.ChartagValue].CallChallenge())
		return
	}

	glog.Infof("%s %#v", "ticket error: ", err)
	gerr, ok := asGerror(err)
	if !ok {
		http.Error(base.W, err.Error(), http.StatusInternalServerError)
		return
	}
	if gerr.Code < 1000 {
		base.SendStatusPage(gerr.Code, gerr.Errstr)
		return
	}
	issuer := (c.Roles[base.RoleValue]).Issuers[provider]
	fn := c.LoginName + "." + base.ChartagValue
	T := template.New(fn).Option("missingkey=zero")
	T, err = T.ParseFiles(c.Template + "/" + base.RoleValue + "/" + c.LoginName + "." + base.ChartagValue)
	if err == nil {
		errstr := c.Errors[strconv.Itoa(gerr.Code)]
		if errstr == "" {
			errstr = gerr.Error()
		}
		var buffer bytes.Buffer
		err = T.Execute(&buffer, map[string]interface{}{
			"LoginName": c.LoginName, "GoURIName": c.GoURIName,
			"Errorstr": errstr, "GoURI": uri,
			"Login": issuer.Credential[0], "Password": issuer.Credential[1]})
		base.SendNocache(buffer.String())
	}
	if err != nil {
		http.Error(base.W, err.Error(), 500)
	}
}

func (self *Controller) checkForm(r *http.Request, dir string) error {
	self.ensureDefaults()
	glog := self.Logger.Sugar()
	reader, err := r.MultipartReader()
	if reader != nil && err != nil {
		return err
	}

	if reader == nil {
		glog.Infof("No multipart")

		err = r.ParseForm()
		if err != nil {
			return err
		}
		form := r.Form

		header := r.Header
		// not json, return
		if header.Get("Content-Type") == "" || !strings.Contains(header.Get("Content-Type"), "application/json") || r.Body == nil {
			return nil
		}

		data, err := io.ReadAll(r.Body)
		t := make(map[string]interface{})
		if err == nil && data != nil {
			err = json.Unmarshal(data, &t)
		}
		if err != nil {
			return err
		}

		for key, value := range t {
			if value == nil {
				continue
			}
			switch s := value.(type) {
			case []string:
				for _, v := range s {
					if v != "" {
						form.Add(key, v)
					}
				}
			case []uint8:
				form.Add(key, string(s))
			case []interface{}:
				for _, v := range s {
					form.Add(key, Interface2String(v))
				}
			default:
				form.Add(key, Interface2String(value))
			}
		}
	} else {
		glog.Infof("multipart/uploading found")
		r.Form = make(url.Values)
		form := r.Form
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			fieldName := part.FormName()
			if fieldName == "" {
				part.Close()
				return Err(1010, "empty multipart field name")
			}
			fileName := part.FileName()
			if fileName == "" {
				var b bytes.Buffer
				limit := self.C.uploadLimit()
				if _, err := io.Copy(&b, io.LimitReader(part, limit+1)); err != nil {
					part.Close()
					return err
				}
				if int64(b.Len()) > limit {
					part.Close()
					return Err(1010, "multipart field too large")
				}
				form.Add(fieldName, b.String())
			} else {
				cleanName := filepath.Base(fileName)
				if cleanName != fileName || cleanName == "." || cleanName == string(filepath.Separator) {
					return Err(1010, "invalid upload filename")
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				fullname := filepath.Join(dir, cleanName)
				dst, err := os.Create(fullname)
				if err != nil {
					return err
				}
				defer dst.Close()
				limit := self.C.uploadLimit()
				written, err := io.Copy(dst, io.LimitReader(part, limit+1))
				if err != nil {
					return err
				}
				if written > limit {
					_ = os.Remove(fullname)
					return Err(1010, "upload file too large")
				}
				form.Add(fieldName, cleanName)
			}
			part.Close()
		}
	}

	return nil
}

func (self *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	self.ensureDefaults()
	if self.C == nil {
		http.Error(w, "Genelet config is not set", http.StatusInternalServerError)
		return
	}
	scrubGeneletForwardedHeaders(r.Header)
	glog := self.Logger.Sugar()
	c := self.C
	if timeout := c.requestTimeout(); timeout > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		r = r.WithContext(ctx)
	}
	length := len(c.Script)

	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Add("Vary", "Origin")
		if !self.corsAllowed(origin) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Max-Age", "1728000")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	}
	if acrh := r.Header.Get("Access-Control-Request-Headers"); acrh != "" {
		w.Header().Set("Access-Control-Allow-Headers", acrh)
	}
	if r.Method == "OPTIONS" || r.Method == "HEAD" {
		w.WriteHeader(200)
		return
	}

	if !strings.HasPrefix(r.URL.Path, c.Script+"/") {
		glog.Infof("%s %s, [static]", r.Method, r.URL.Path)
		self.staticPage(w, r)
		return
	}

	glog.Infof("%s %s, %v", r.Method, r.URL.Path, r.URL.Query())
	var methodFound bool
	for k := range c.DefaultActions {
		if k == r.Method {
			methodFound = true
			break
		}
	}
	if !methodFound {
		http.Error(w, "The http method is not supported", http.StatusMethodNotAllowed)
		return
	}

	pathInfo := strings.Split(r.URL.Path[length+1:], "/")
	if len(pathInfo) == 4 {
		r.Header.Set("X-Forwarded-ID", pathInfo[3])
	} else if len(pathInfo) != 3 {
		glog.Infof("not genelet url")
		http.Error(w, "Bad Request", 400)
		return
	}

	chartag, ok := c.Chartags[pathInfo[1]]
	if !ok {
		glog.Infof("check chartag")
		http.Error(w, "Bad Request", 400)
		return
	}

	base := &Base{C: c, W: w, R: r, RoleValue: pathInfo[0], ChartagValue: pathInfo[1]}
	gate := NewGate(*base)
	obj := pathInfo[2]

	glog.Infof("parse form")
	err := self.checkForm(r, c.UploadDir)
	if err != nil {
		http.Error(w, "Bad Request: "+err.Error(), 400)
		return
	}

	_, ok = c.Roles[base.RoleValue]
	if !ok && (gate.RoleValue != c.Pubrole) {
		http.NotFound(w, r)
		return
	}

	if obj == c.LoginName || Grep(c.Oauth2s, obj) || Grep(c.Oauth1s, obj) {
		glog.Infof("loging for %s", obj)
		if obj != c.LoginName {
			r.Form.Set(c.ProviderName, obj)
		}
		self.loginPage(base)
		glog.Infof("end login")
		return
	} else if obj == c.LogoutName {
		glog.Infof("start logout")
		err = gate.HandleLogout()
		if err != nil {
			gerr, ok := asGerror(err)
			if !ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			gate.SendStatusPage(gerr.Code, gerr.Errstr)
			glog.Infof("end logout")
		}
		return
	}

	if gate.RoleValue != c.Pubrole {
		err = gate.Forbid()
		if err != nil {
			glog.Infof("forbidden %v", err)
			gerr, ok := asGerror(err)
			if !ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			gate.SendStatusPage(gerr.Code, gerr.Errstr)
			return
		}
	}

	err = self.Handle(obj, *base, r.Method)
	if err != nil {
		switch g := err.(type) {
		case Gerror:
			if g.Code == 303 {
				base.SendStatusPage(303, err.Error())
				return
			} else if g.Code < 1000 {
				err = Gerror{g.Code, http.StatusText(g.Code)}
			} else {
				err = Gerror{g.Code, err.Error()}
			}
		default:
			err = Gerror{1000, err.Error()}
		}
		glog.Infof("error: %v", err)

		tmplfile := c.Template + "/" + base.RoleValue + "/error." + base.ChartagValue
		T0, er := template.ParseFiles(tmplfile)
		if er != nil {
			base.SendPage(addJSON(chartag.Case, er.Error()))
			return
		}
		var buffer bytes.Buffer
		er = T0.Execute(&buffer, err)
		if er != nil {
			base.SendPage(addJSON(chartag.Case, er.Error()))
		} else if err.(Gerror).Code < 1000 {
			base.SendStatusPage(err.(Gerror).Code, buffer.String())
		} else {
			base.SendPage(buffer.String())
		}
	}
	glog.Infof("handler ended.")
}

func addJSON(c int8, msg string) string {
	if c > 0 {
		return `{"data": "` + msg + `"}`
	}
	return msg
}

func (self *Controller) Handle(obj string, base Base, method string) error {
	self.ensureDefaults()
	glog := self.Logger.Sugar()
	model, ok := self.newModel(obj)
	if !ok {
		return Err(404)
	}
	filter, ok := self.newFilter(obj)
	if !ok {
		return Err(404)
	}
	storage := self.newStorage()
	who := base.RoleValue
	tag := base.ChartagValue

	c := self.C
	r := base.R
	ARGS := r.Form

	ARGS.Set("_gobj", obj)
	ARGS.Set("_gtime", strconv.FormatInt(time.Now().Unix(), 10))

	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	if err := InvokeVoid(model, "SetDefaults", ARGS, &lists, &other, storage); err != nil {
		return err
	}

	action := ARGS.Get(c.ActionName)
	if action == "" {
		action = c.DefaultActions[method]
		if method == "GET" && r.Header.Get("X-Forwarded-ID") != "" {
			action = c.DefaultActions["GET_item"]
		}
	}
	if actionRequiresPost(action) && method != http.MethodPost && method != http.MethodDelete {
		return Err(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
	}
	if r.Header.Get("X-Forwarded-ID") != "" {
		ARGS.Set("_gid_url", r.Header.Get("X-Forwarded-ID"))
	}
	glog.Infof("action: %s", action)
	if err := InvokeVoid(filter, "SetAll", base, action, obj, &other); err != nil {
		return err
	}
	ret, err := TryInvoke(filter, "GetAll")
	if err != nil {
		return err
	}
	if len(ret) != 2 || ret[0].Interface() == nil {
		return Err(404)
	}
	actionHash, ok := ret[0].Interface().(map[string][]string)
	if !ok {
		return Err(1051, "GetAll returned invalid action map")
	}
	fk := make([]string, 0)
	if ret[1].Interface() != nil {
		var ok bool
		fk, ok = ret[1].Interface().([]string)
		if !ok {
			return Err(1051, "GetAll returned invalid foreign-key list")
		}
	}
	parts := strings.Split(r.RequestURI, "/")
	parts[3] = "json"
	ARGS.Set("_guri_json", strings.Join(parts, "/"))
	ARGS.Set("_guri", r.RequestURI)
	ARGS.Set("_grole", who)
	ARGS.Set("_action", action)

	role, ok := c.Roles[base.RoleValue]
	var isAdmin bool
	if ok {
		ARGS.Set("_gid_name", role.Id_name)
		ARGS.Set("_gtype_id", strconv.Itoa(role.Type_id))
		if role.Is_admin {
			ARGS.Set("_gadmin", "1")
			isAdmin = true
		}
		h := r.Header
		ARGS.Set("_gwhen", h.Get("X-Forwarded-Time"))
		ARGS.Set("_gduration", h.Get("X-Forwarded-Duration"))
		if h.Get("X-Forwarded-User") == "" || h.Get("X-Forwarded-User") == "NULL" {
			return Err(401)
		}
		cipherSet := func(k, v string) {
			if k == role.Id_name && role.Id_cipher {
				id64, _ := strconv.ParseInt(v, 10, 64)
				ARGS.Set(k, strconv.FormatInt(int64(int_cipher.Decrypt(uint(id64), c.Secret)), 10))
				if k == role.Attributes[0] {
					ARGS.Set("_gid_cipher", v)
				}
			} else {
				ARGS.Set(k, v)
			}
		}
		cipherSet(role.Attributes[0], h.Get("X-Forwarded-User"))
		if len(role.Attributes) > 1 {
			groups := strings.Split(h.Get("X-Forwarded-Group"), "|")
			if len(groups) != len(role.Attributes)-1 {
				return Err(401, "forwarded group count mismatch")
			}
			for i := 1; i < len(role.Attributes); i++ {
				cipherSet(role.Attributes[i], groups[i-1])
			}
		}
	}

	glog.Infof("access control")
	if !isAdmin && !Grep(actionHash["groups"], who) {
		return Err(401)
	}

	extra := make(url.Values)
	if !isAdmin && ok {
		glog.Infof("check fk")
		err := self.assignFK(who, fk, ARGS, extra)
		if err != nil {
			return err
		}
	}

	if isMutatingMethod(method) {
		if err := base.ValidateCSRF(); err != nil {
			return err
		}
	}
	glog.Infof("preset")
	err = InvokeError(filter, "Preset")
	if err != nil {
		return err
	}

	glog.Infof("validation")
	validate, ok := actionHash["validate"]
	if ok {
		for _, field := range validate {
			if ARGS.Get(field) == "" {
				return Err(1092, field)
			}
		}
	}

	options, ok := actionHash["options"]
	if !ok || !Grep(options, "no_db") {
		if err := InvokeVoid(model, "SetDB", self.DB); err != nil {
			return err
		}
		if err := InvokeOptionalVoid(model, "SetDriver", self.dbDriver()); err != nil {
			return err
		}
	}

	nextextra := make(url.Values)
	glog.Infof("before")
	err = InvokeError(filter, "Before", model, extra, nextextra)
	if err != nil {
		return err
	}

	if !Grep(options, "no_method") {
		x, err := actionMethod(action)
		if err != nil {
			return err
		}
		glog.Infof("call model")
		err = InvokeError(model, x, extra, nextextra)
		if err != nil {
			return err
		}
		glog.Infof("call model OK")
	}

	if !isAdmin && len(lists) > 0 {
		glog.Infof("fk tobe")
		self.assignFKTobe(who, fk, ARGS, lists)
	}

	glog.Infof("after")
	err = InvokeError(filter, "After", model)
	if err != nil {
		return err
	}

	glog.Infof("call blocks")
	err = c.Sendmail(lists, ARGS, other)
	if err != nil {
		return err
	}

	tmpl := &Tmpl{ARGS: ARGS, Lists: lists, Other: other, Success: true}
	chartag := c.Chartags[tag]
	if chartag.Case > 0 {
		glog.Infof("generate json")
		if ARGS.Get(role.Id_name) != "" && role.Id_cipher {
			ARGS.Set(role.Id_name, ARGS.Get("_gid_cipher"))
		}
		for k := range ARGS {
			if len(k) > 2 && k[0:2] == "_g" {
				ARGS.Del(k)
			}
		}
		b, eb := json.Marshal(tmpl)
		if eb != nil {
			return eb
		}
		base.SendPage(string(b))
		return nil
	}

	glog.Infof("call template")
	other["Component"] = obj
	other["Tag"] = tag
	other["Role"] = who
	other["Action"] = action

	tmplname := action + "." + tag
	tmplfile := c.Template + "/" + who + "/" + obj + "/" + tmplname
	globfiles := c.Template + "/" + who + "/*." + tag
	T0 := template.New(tmplname).Option("missingkey=zero")
	var er error
	if T0, er = T0.ParseFiles(tmplfile); er == nil {
		if T0, er = T0.ParseGlob(globfiles); er == nil {
			glog.Infof("generate page")
			var output string
			if output, er = tmpl.Get_page(T0); er == nil {
				glog.Infof("sending page")
				base.SendPage(output)
			}
		}
	}
	return er
}

func (self *Controller) newModel(name string) (interface{}, bool) {
	if self.ModelFactories != nil {
		if factory, ok := self.ModelFactories[name]; ok {
			return factory(), true
		}
	}
	if self.Models == nil {
		return nil, false
	}
	model, ok := self.Models[name]
	return model, ok
}

func (self *Controller) newFilter(name string) (interface{}, bool) {
	if self.FilterFactories != nil {
		if factory, ok := self.FilterFactories[name]; ok {
			return factory(), true
		}
	}
	if self.Filters == nil {
		return nil, false
	}
	filter, ok := self.Filters[name]
	return filter, ok
}

func (self *Controller) newStorage() map[string]interface{} {
	storage := make(map[string]interface{}, len(self.Storage)+len(self.StorageFactories))
	for k, v := range self.Storage {
		storage[k] = v
	}
	for k, factory := range self.StorageFactories {
		storage[k] = factory()
	}
	return storage
}

func asGerror(err error) (Gerror, bool) {
	if err == nil {
		return Gerror{}, false
	}
	g, ok := err.(Gerror)
	if ok {
		return g, true
	}
	return Gerror{}, false
}

func (self *Controller) assignFK(who string, fk []string, ARGS url.Values, extra url.Values) error {
	if fk == nil || self.C.Secret == "" {
		return nil
	}
	name := fk[0]
	if name == "" {
		return nil
	}

	value := ARGS.Get(name)
	if value == "" {
		return Err(1041)
	}
	extra.Set(name, value)
	roleid := ARGS.Get("_gid_name")
	if name == roleid {
		return nil
	}

	if fk[1] == "" {
		return Err(1054)
	}
	md5 := ARGS.Get(fk[1])
	if md5 == "" {
		return Err(1055)
	}

	stamp := ARGS.Get("_gwhen")
	valueRoleID := ARGS.Get(roleid)
	if md5 != Digest(self.C.Secret, stamp, who, roleid, valueRoleID, name, value) {
		return Err(1052)
	}
	if ARGS.Get("_gduration") != "" {
		gtime, _ := strconv.ParseInt(ARGS.Get("_gtime"), 10, 32)
		slast, _ := strconv.ParseInt(stamp, 10, 32)
		if gtime > slast {
			return Err(1053)
		}
	}

	return nil
}

func (self *Controller) fkTobe(lists []map[string]interface{}, fk []string, stamp, who, roleid, valueRoleID string) {
	if len(fk) <= 2 || fk[2] == "" || fk[2] == roleid || fk[3] == "" {
		return
	}
	name := fk[2]
	for _, item := range lists {
		itemName, ok := item[name]
		if !ok {
			continue
		}
		value := Interface2String(itemName)
		item[fk[3]] = Digest(self.C.Secret, stamp, who, roleid, valueRoleID, name, value)
	}
}

func (self *Controller) assignFKTobe(who string, fk0 []string, ARGS url.Values, lists []map[string]interface{}) error {
	if fk0 == nil || self.C.Secret == "" {
		return nil
	}
	roleid := ARGS.Get("_gid_name")

	stamp := ARGS.Get("_gwhen")
	valueRoleID := ARGS.Get(roleid)

	fk := make([]string, len(fk0))
	copy(fk, fk0)

	self.fkTobe(lists, fk, stamp, who, roleid, valueRoleID)

	for len(fk) > 4 {
		fk = fk[3:]
		which := fk[1]
		if lists[0][which] == nil {
			return Err(1056)
		}
		for _, item := range lists {
			self.fkTobe(item[which].([]map[string]interface{}), fk, stamp, who, roleid, valueRoleID)
		}
	}
	return nil
}
