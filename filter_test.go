package genelet

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"
)

func TestFilter(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	args := make(url.Values)
	args.Set("aaa", "111")
	args.Set("bbb", "222")
	args.Set("Errorstr", Err(1001).Error())
	args.Set("Script", configure.Script)
	args.Set(configure.RoleName, "m")
	args.Set(configure.GoURIName, "aaa")
	args.Set("LoginName", "email")
	args.Set("Password_name", "passwd")

	topics := map[string][]string{"aliases": {"list", "subj"}, "groups": {"m", "p"}, "validate": {"aaa", "bbb"}}
	ads := map[string][]string{"groups": {"m"}, "validate": {"ccc"}}
	dashboard := map[string][]string{"groups": {"p"}, "validate": {"ccc"}}
	actions := map[string]map[string][]string{"dashboard": dashboard, "topics": topics, "ads": ads}
	r, err := http.NewRequest("GET", "http://xxx.yyy", bytes.NewBuffer([]byte("sss")))
	if err != nil {
		panic(err)
	}
	f := new(Filter)
	f.C = configure
	f.R = r
	f.RoleValue = "m"
	f.ChartagValue = "json"
	f.R.Form = args
	f.Actions = actions
	f.Action = configure.DefaultActions["GET"]

	a := f.Action
	// a, as := f.Get_action()
	as, _ := f.GetAll()
	if a != "dashboard" {
		t.Errorf("%v got", a)
	}
	if as["groups"][0] != "p" {
		t.Errorf("%v got", as)
	}

	args.Set("_gaction", "topics")
	_, ok := as["validate"]
	if ok != true {
		t.Errorf("%s got", "validate not")
	}
	args.Set("_gaction", "ads")
	validate, ok := as["validate"]
	if ok && validate[0] != "ccc" {
		t.Errorf("%s got", validate[0])
	}
}
