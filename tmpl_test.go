package genelet

import (
	"html/template"
	"regexp"
	"testing"
)

func TestTmpl(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := new(Tmpl)
	tmpl.Other = map[string]interface{}{"Errorstr": Err(1001).Error(), "Script": "bb", configure.RoleName: "m", configure.GoURIName: "aaa", "Login_name": "email", "Password_name": "passwd"}
	T0, err := template.ParseFiles("tmpl.html")
	if err != nil {
		t.Errorf("%s error", err.Error())
	}
	str, err := tmpl.Get_page(T0)
	if err != nil {
		t.Errorf("%s error", err.Error())
	}
	matched, err := regexp.MatchString("Google authorization required", str)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("%s wanted", str)
	}
	matched, err = regexp.MatchString("aaa.*bb.*email.*passwd", str)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("%s wanted", str)
	}
}
