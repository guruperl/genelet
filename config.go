// Package genelet is a genelet package for genelet framework.
package genelet

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type PatternCase int

const (
	REROUTE PatternCase = iota
	CACHE
	STATIC
)

type Pattern struct {
	Reg      string
	Regs     *regexp.Regexp
	Keys     []string
	Initials string
	Expire   int
	Case     PatternCase
}

type Issuer struct {
	Default      bool
	Screen       int8
	Sql          string
	Sql_as       string
	PasswordHash string            `json:"Password_hash"`
	ProviderPars map[string]string `json:"Provider_pars"`
	Credential   []string
	InPars       []string
	OutPars      []string
	ConditionURI [][]string
}

type Role struct {
	Id_name    string
	Id_cipher  bool
	Type_id    int
	Is_admin   bool
	Attributes []string

	Coding    string
	Secret    string
	Surface   string
	Length    int8
	Duration  int
	Userlist  []string
	Grouplist []string
	Logout    string
	Domain    string
	Path      string
	MaxAge    int

	Issuers map[string]Issuer
}

type Config struct {
	UploadDir             string
	Template              string
	Pubrole               string
	Secret                string
	ServerURL             string
	CORSOrigins           []string
	UploadURL             string
	ServerPort            string
	DocumentRoot          string
	ProjectRoot           string
	Script                string
	ComponentName         string
	ActionName            string
	DefaultActions        map[string]string
	RoleName              string
	Oauth2s               []string
	Oauth1s               []string
	LoginName             string
	LogoutName            string
	TagName               string
	ProviderName          string
	CallbackName          string
	GoStampName           string
	GoMD5Name             string
	GoURIName             string
	GoProbeName           string
	GoErrName             string
	UploadMaxBytes        int64
	CSRFName              string
	RequestTimeoutSeconds int

	ConnectArray []string
	Blks         map[string]map[string]string
	Chartags     map[string]Chartag
	Roles        map[string]Role
	Errors       map[string]string
	Custom       map[string]string
	Patterns     []Pattern
}

func NewConfig(filename string) (*Config, error) {
	parsed := new(Config)
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, parsed)
	if err != nil {
		return nil, err
	}

	if parsed.ConnectArray == nil {
		if os.Getenv("DBUSER") != "" && os.Getenv("DBPASS") != "" && os.Getenv("DBNAME") != "" {
			host := "localhost:3306"
			if x := os.Getenv("DBHOST"); x != "" {
				host = x
				if !strings.Contains(host, ":") {
					host += ":3306"
				}
			}
			parsed.ConnectArray = []string{"mysql", os.Getenv("DBUSER") + ":" + os.Getenv("DBPASS") + "@tcp(" + host + ")/" + os.Getenv("DBNAME")}
		} else {
			return nil, fmt.Errorf("ConnectArray is not set")
		}
	}

	if parsed.ServerURL == "" {
		parsed.ServerURL = "http://localhost"
	}
	if parsed.ServerPort == "" {
		parsed.ServerPort = "80"
	}
	if parsed.UploadDir == "" {
		parsed.UploadDir = "/tmp"
	}
	if parsed.UploadURL == "" {
		parsed.UploadURL = parsed.ServerURL + "/uploads"
	}
	if parsed.ComponentName == "" {
		parsed.ComponentName = "component"
	}
	if parsed.ActionName == "" {
		parsed.ActionName = "action"
	}
	if parsed.GoStampName == "" {
		parsed.GoStampName = "go_stamp"
	}
	if parsed.GoMD5Name == "" {
		parsed.GoMD5Name = "go_md5"
	}
	if parsed.GoURIName == "" {
		parsed.GoURIName = "go_uri"
	}
	if parsed.RoleName == "" {
		parsed.RoleName = "role"
	}
	if parsed.Oauth2s == nil {
		parsed.Oauth2s = []string{"google", "facebook", "microsoft", "qq", "sina"}
	}
	if parsed.Oauth1s == nil {
		parsed.Oauth1s = []string{"twitter", "linkedin"}
	}
	if parsed.LoginName == "" {
		parsed.LoginName = "login"
	}
	if parsed.LogoutName == "" {
		parsed.LogoutName = "logout"
	}
	if parsed.TagName == "" {
		parsed.TagName = "tag"
	}
	if parsed.ProviderName == "" {
		parsed.ProviderName = "provider"
	}
	if parsed.CallbackName == "" {
		parsed.CallbackName = "callback"
	}
	if parsed.GoProbeName == "" {
		parsed.GoProbeName = "go_probe"
	}
	if parsed.GoErrName == "" {
		parsed.GoErrName = "go_err"
	}
	if parsed.Errors == nil {
		parsed.Errors = make(map[string]string)
	}

	if parsed.DefaultActions == nil {
		parsed.DefaultActions = map[string]string{"GET": "dashboard", "GET_item": "edit", "PUT": "update", "POST": "insert", "DELETE": "delete"}
	}

	//for _, pattern := range parsed.Patterns {
	//pattern.Regs = regexp.MustCompile(pattern.Reg)
	//}

	return parsed, nil
}
