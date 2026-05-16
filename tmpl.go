package genelet

import (
	"bytes"
	"html/template"
	"net/url"
)

type Tmpl struct {
	Lists   []map[string]interface{}
	ARGS    url.Values
	Other   map[string]interface{}
	Extra   map[string]interface{}
	Success bool
}

func (self *Tmpl) Get_page(T *template.Template) (string, error) {
	var buffer bytes.Buffer
	err := T.Execute(&buffer, *self)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
