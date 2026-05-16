package genelet

import (
	"testing"
)

const (
	filename = "test.conf"
)

func TestConfig(t *testing.T) {
	c, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	if c.DocumentRoot != "aa" {
		t.Errorf("%s wanted", "aa")
	}
	c.DocumentRoot = "root"
	if c.DocumentRoot != "root" {
		t.Errorf("%s wanted", "root")
	}
	if c.Script != "bb" {
		t.Errorf("%s wanted", "bb")
	}
	if len(c.CORSOrigins) != 1 || c.CORSOrigins[0] != "https://admin.example.test" {
		t.Errorf("%v wanted", c.CORSOrigins)
	}
	if c.Pubrole != "cc" {
		t.Errorf("%s wanted", "cc")
	}
	if c.Secret != "" {
		t.Errorf("%s is empty", "secret")
	}
	c.Secret = "dd"
	if c.Secret != "dd" {
		t.Errorf("%s wanted", "dd")
	}
	if c.Template != "ee" {
		t.Errorf("%s wanted", "ee")
	}
	if c.ActionName != "action" {
		t.Errorf("%s wanted", "action")
	}
	if c.GoURIName != "go_uri" {
		t.Errorf("%s wanted", "go_uri")
	}
	if c.ConnectArray[0] != "mysql" {
		t.Errorf("%s wanted", c.ConnectArray[0])
	}
	if c.ConnectArray[1] != "genelet_test:genelet_pass@tcp(127.0.0.1:3306)/genelet_test" {
		t.Errorf("%s wanted", c.ConnectArray[1])
	}
	if c.ConnectArray[2] != "ccc" {
		t.Errorf("%s wanted", "ccc")
	}
	char := c.Chartags["json"]
	if char.ContentType != "application/json; charset=\"UTF-8\"" {
		t.Errorf("%s wanted", "application/json; charset=\"UTF-8\"")
	}
	if char.Challenge != "challenge" {
		t.Errorf("%s wanted", "challenge")
	}
}
