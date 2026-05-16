package genelet

import (
	"context"
	"database/sql"
	"math"
	"math/rand"
	"net/url"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type Model struct {
	Crud
	Context context.Context

	ARGS  url.Values
	LISTS *[]map[string]interface{}
	OTHER *map[string]interface{}

	SORTBY      string
	SORTREVERSE string
	PAGENO      string
	ROWCOUNT    string
	TOTALNO     string
	MAXPAGENO   string
	FIELD       string
	EMPTIES     string

	Nextpages map[string][]map[string]interface{}
	Storage   map[string]interface{}

	CurrentKey    string            `json:"current_key"`
	CurrentKeys   []string          `json:"current_keys"`
	CurrentIDAuto string            `json:"current_id_auto"`
	KeyIN         map[string]string `json:"key_in"`

	InsertPars     []string          `json:"insert_pars"`
	EditPars       []string          `json:"edit_pars"`
	UpdatePars     []string          `json:"update_pars"`
	InsupdPars     []string          `json:"insupd_pars"`
	TopicsPars     []string          `json:"topics_pars"`
	TopicsHashpars map[string]string `json:"topics_hashpars"`

	TotalForce int `json:"total_force"`
}

func (self *Model) Initialize(comp *Component, logger ...*zap.Logger) {
	if len(logger) > 0 {
		self.Crud.DBI.Logger = logger[0]
	}

	self.SORTBY = comp.Sortby
	self.SORTREVERSE = comp.Sortreverse
	self.PAGENO = comp.Pageno
	self.ROWCOUNT = comp.Rowcount
	self.TOTALNO = comp.Totalno
	self.MAXPAGENO = comp.Maxpageno
	self.FIELD = comp.Fields
	self.EMPTIES = comp.Empties

	self.Nextpages = comp.Nextpages

	self.CurrentTable = comp.CurrentTable
	self.CurrentTables = comp.CurrentTables
	self.CurrentKey = comp.CurrentKey
	self.CurrentKeys = comp.CurrentKeys
	self.CurrentIDAuto = comp.CurrentIDAuto
	self.KeyIN = comp.KeyIN

	self.InsertPars = comp.InsertPars
	self.EditPars = comp.EditPars
	self.UpdatePars = comp.UpdatePars
	self.InsupdPars = comp.InsupdPars
	self.TopicsPars = comp.TopicsPars
	self.TopicsHashpars = comp.TopicsHash

	self.TotalForce = comp.TotalForce
}

func (self *Model) SetDB(db *sql.DB) {
	self.DB = db
}

func (self *Model) SetDefaults(args url.Values, lists *[]map[string]interface{}, other *map[string]interface{}, storage map[string]interface{}) {
	self.ARGS = args
	self.LISTS = lists
	self.OTHER = other
	self.Storage = storage
}

func (self *Model) filteredFields(pars []string) []string {
	ARGS := self.ARGS
	field := ARGS[self.FIELD]
	if field == nil {
		return pars
	}

	out := make([]string, 0)
	for _, val := range field {
		for _, v := range pars {
			if val == v {
				out = append(out, v)
				break
			}
		}
	}
	return out
}

func (self *Model) get_fv(pars []string) url.Values {
	ARGS := self.ARGS
	fieldValues := make(url.Values)
	for _, f := range self.filteredFields(pars) {
		if ARGS.Get(f) != "" {
			fieldValues[f] = ARGS[f]
		}
	}
	return fieldValues
}

func (self *Model) get_id_val(extra url.Values) (interface{}, []interface{}) {
	ARGS := self.ARGS
	if len(self.CurrentKeys) > 0 {
		val := make([]interface{}, 0)
		for _, v := range self.CurrentKeys {
			if ARGS.Get(v) != "" {
				val = append(val, ARGS.Get(v))
			} else if extra[v] != nil {
				val = append(val, extra[v])
			} else {
				return self.CurrentKeys, nil
			}
		}
		return self.CurrentKeys, val
	}

	if ARGS.Get(self.CurrentKey) != "" {
		return self.CurrentKey, []interface{}{ARGS.Get(self.CurrentKey)}
	} else if extra.Get(self.CurrentKey) != "" {
		return self.CurrentKey, []interface{}{extra.Get(self.CurrentKey)}
	} else if ARGS.Get("_gid_url") != "" {
		return self.CurrentKey, []interface{}{ARGS.Get("_gid_url")}
	}
	return self.CurrentKey, nil
}

func (self *Model) Topics(extra ...url.Values) error {
	ARGS := self.ARGS
	totalno := self.TOTALNO
	if self.TotalForce != 0 && ARGS.Get(self.ROWCOUNT) != "" && (ARGS.Get(self.PAGENO) == "" || ARGS.Get(self.PAGENO) == "1") {
		var nt int
		if self.TotalForce < -1 {
			nt = int(math.Abs(float64(self.TotalForce)))
		} else if self.TotalForce == -1 || ARGS.Get(totalno) == "" {
			hash := make(map[string]interface{})
			err := self.TotalHash(hash, totalno, extra...)
			if err != nil {
				return err
			}
			nt = int(hash[totalno].(int64))
		} else {
			nt, _ = strconv.Atoi(ARGS.Get(totalno))
		}
		ARGS.Set(totalno, strconv.Itoa(nt))
		nr, _ := strconv.Atoi(ARGS.Get(self.ROWCOUNT))
		max_pageno := int((nt-1)/nr) + 1
		ARGS.Set(self.MAXPAGENO, strconv.Itoa(max_pageno))
	}

	var fields interface{}
	if self.TopicsHashpars == nil {
		fields = self.filteredFields(self.TopicsPars)
	} else {
		fields = self.TopicsHashpars
	}
	order, err := self.GetOrderStringChecked()
	if err != nil {
		return err
	}
	err = self.TopicsHashOrder(self.LISTS, fields, order, extra...)
	if err != nil {
		return err
	}

	return self.ProcessAfter("topics", extra...)
}

func (self *Model) Edit(extra ...url.Values) error {
	var id interface{}
	var val []interface{}
	if len(extra) == 0 {
		id, val = self.get_id_val(nil)
	} else {
		id, val = self.get_id_val(extra[0])
	}
	if val == nil {
		return Err(1040)
	}

	fields := self.filteredFields(self.EditPars)
	if fields == nil {
		return Err(1077)
	}

	var err error
	if len(extra) > 0 && extra[0] != nil {
		err = self.EditHash(self.LISTS, fields, id, val, extra[0])
	} else {
		err = self.EditHash(self.LISTS, fields, id, val)
	}
	if err != nil {
		return err
	}

	return self.ProcessAfter("edit", extra...)
}

// Insert use 'extra' to override fieldValues for selected fields
func (self *Model) Insert(extra ...url.Values) error {
	fieldValues := self.get_fv(self.InsertPars)

	if len(extra) > 0 {
		for key, value := range extra[0] {
			if Grep(self.InsertPars, key) {
				fieldValues[key] = value
			}
		}
	}
	if fieldValues == nil {
		return Err(1078)
	}

	err := self.InsertHash(fieldValues)
	if err != nil {
		return err
	}

	if self.CurrentIDAuto != "" {
		autoID := strconv.FormatInt(self.LastID, 10)
		fieldValues.Set(self.CurrentIDAuto, autoID)
		self.ARGS.Set(self.CurrentIDAuto, autoID)
	}
	*self.LISTS = from_fv(fieldValues)

	return self.ProcessAfter("insert", extra...)
}

func (self *Model) Insupd(extra ...url.Values) error {
	uniques := self.InsupdPars
	if uniques == nil {
		return Err(1078)
	}

	fieldValues := self.get_fv(self.InsertPars)
	if len(extra) > 0 {
		for key, value := range extra[0] {
			if Grep(self.InsertPars, key) {
				fieldValues[key] = value
			}
		}
	}
	if fieldValues == nil {
		return Err(1076)
	}

	for _, v := range uniques {
		if fieldValues[v] == nil {
			return Err(1075)
		}
	}

	upd_fieldValues := self.get_fv(self.UpdatePars)

	s_hash := ""
	err := self.InsupdHash(fieldValues, upd_fieldValues, self.CurrentKey, uniques, &s_hash)
	if err != nil {
		return err
	}

	if self.CurrentIDAuto != "" && s_hash == "insert" {
		fieldValues[self.CurrentIDAuto] = []string{strconv.FormatInt(self.LastID, 10)}
	}
	hash := make(map[string]interface{})
	for k, v := range fieldValues {
		hash[k] = v[0]
	}
	*self.LISTS = append(*self.LISTS, hash)

	return self.ProcessAfter("insupd", extra...)
}

func (self *Model) Update(extra ...url.Values) error {
	ARGS := self.ARGS
	var id interface{}
	var val []interface{}
	if len(extra) > 0 {
		id, val = self.get_id_val(extra[0])
	} else {
		id, val = self.get_id_val(nil)
	}
	if val == nil {
		return Err(1040)
	}

	fieldValues := self.get_fv(self.UpdatePars)
	if fieldValues == nil {
		return Err(1074)
	}

	if len(fieldValues) <= 1 && fieldValues[id.(string)] != nil {
		*self.LISTS = from_fv(fieldValues)
		return self.ProcessAfter("update", extra...)
	}

	var err error
	if ARGS.Get(self.EMPTIES) != "" {
		err = self.UpdateHashNulls(fieldValues, id, val, ARGS[self.EMPTIES], extra...)
	} else {
		err = self.UpdateHash(fieldValues, id, val, extra...)
	}
	if err != nil {
		return err
	}

	switch id.(type) {
	case []string:
		for i, v := range id.([]string) {
			fieldValues.Set(v, val[i].(string))
		}
	case string:
		fieldValues.Set(id.(string), val[0].(string))
	}
	*self.LISTS = from_fv(fieldValues)

	return self.ProcessAfter("Update", extra...)
}

func from_fv(fieldValues url.Values) []map[string]interface{} {
	hash := make(map[string]interface{})
	for k, v := range fieldValues {
		hash[k] = v[0]
	}
	return []map[string]interface{}{hash}
}

func (self *Model) Delete(extra ...url.Values) error {
	var id interface{}
	var val []interface{}
	if extra == nil {
		id, val = self.get_id_val(nil)
	} else {
		id, val = self.get_id_val(extra[0])
	}
	if val == nil {
		return Err(1040)
	}

	if self.KeyIN != nil {
		for table, keyname := range self.KeyIN {
			for _, v := range val {
				err := self.Existing(table, keyname, v)
				if err != nil {
					return err
				}
			}
		}
	}

	err := self.DeleteHash(id, val, extra...)
	if err != nil {
		return err
	}

	hash := make(map[string]interface{})
	switch t := id.(type) {
	case []string:
		for i, v := range t {
			hash[v] = val[i]
		}
	case string:
		hash[id.(string)] = val[0]
	}
	*self.LISTS = make([]map[string]interface{}, 0)
	*self.LISTS = append(*self.LISTS, hash)

	return self.ProcessAfter("delete", extra...)
}

func (self *Model) Existing(table string, field string, val interface{}) error {
	if err := ValidateSQLIdentifier("table", table); err != nil {
		return err
	}
	if err := ValidateSQLIdentifier("field", field); err != nil {
		return err
	}
	hash := make(map[string]interface{})
	err := self.GetSQL(hash, "SELECT "+field+" FROM "+table+" WHERE "+field+"=? LIMIT 1", val)
	if err != nil {
		return err
	}
	if hash[field] != nil {
		return Err(1075)
	}

	return nil
}

func (self *Model) Randomid(table string, field string, m ...interface{}) error {
	var min, max, trials int
	if m == nil {
		min = 0
		max = 4294967295
		trials = 10
	} else {
		min = m[0].(int)
		max = m[1].(int)
		if m[2] == nil {
			trials = 10
		} else {
			trials = m[2].(int)
		}
	}

	for i := 0; i < trials; i++ {
		val := min + int(rand.Float32()*float32(max-min))
		err := self.Existing(table, field, val)
		if err != nil {
			continue
		}
		self.ARGS.Set(field, strconv.Itoa(val))
		return nil
	}

	return Err(1076)
}

func (self *Model) GetOrderString() string {
	order, err := self.GetOrderStringChecked()
	if err != nil {
		return ""
	}
	return order
}

func (self *Model) GetOrderStringChecked() (string, error) {
	ARGS := self.ARGS
	var column string
	if ARGS.Get(self.SORTBY) == "" {
		if len(self.CurrentKeys) > 0 {
			column = strings.Join(self.CurrentKeys, ",")
		} else {
			column = self.CurrentKey
		}
	} else {
		column = ARGS.Get(self.SORTBY)
	}

	if self.CurrentTables != nil && !strings.Contains(column, `.`) {
		if self.CurrentTables[0].Alias != "" {
			column = self.CurrentTables[0].Alias + "." + column
		} else {
			column = self.CurrentTables[0].Name + "." + column
		}
	}
	if err := ValidateSQLOrderBy(column); err != nil {
		return "", err
	}
	order := "ORDER BY " + column
	if ARGS.Get(self.SORTREVERSE) != "" {
		order += " DESC"
	}

	if ARGS.Get(self.ROWCOUNT) != "" {
		rowcount, err := strconv.Atoi(ARGS.Get(self.ROWCOUNT))
		if err != nil {
			return "", err
		}
		pageno := 1
		if ARGS.Get(self.PAGENO) == "" {
			ARGS.Set(self.PAGENO, "1")
		} else {
			pageno, err = strconv.Atoi(ARGS.Get(self.PAGENO))
			if err != nil {
				return "", err
			}
		}
		if rowcount < 1 || pageno < 1 {
			return "", Err(1071, "rowcount and pageno must be positive")
		}
		order += " LIMIT " + strconv.Itoa(rowcount) + " OFFSET " + strconv.Itoa((pageno-1)*rowcount)
	}

	return order, nil
}

func (self *Model) another_object(page map[string]interface{}, args url.Values, lists *[]map[string]interface{}, other *map[string]interface{}) (interface{}, string, string, error) {
	model := page["model"].(string)
	if self.Storage == nil {
		return nil, "", "", Err(2013, "No storage")
	}
	p := self.Storage[model]
	if p == nil {
		return nil, "", "", Err(2013, "No stored "+model)
	}

	if err := InvokeVoid(p, "SetDefaults", args, lists, other, self.Storage); err != nil {
		return nil, "", "", err
	}
	if err := InvokeVoid(p, "SetDB", self.DB); err != nil {
		return nil, "", "", err
	}

	action := page["action"].(string)
	method, err := actionMethod(action)
	if err != nil {
		return nil, "", "", err
	}
	return p, method, model + "_" + action, nil
}

func (self *Model) CallOnce(page map[string]interface{}, extra ...url.Values) error {
	args := self.ARGS
	ARGS, err := url.ParseQuery(args.Encode())
	if err != nil {
		return err
	}
	if ARGS.Get("sortby") != "" {
		ARGS.Del("sortby")
	}
	if ARGS.Get("sortreverse") != "" {
		ARGS.Del("sortreverse")
	}
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})

	var next_extra url.Values
	if extra != nil {
		next_extra = extra[0]
	}
	if page["manual"] != nil {
		if next_extra == nil {
			next_extra = url.Values{}
		}
		for k, v := range page["manual"].(map[string]interface{}) {
			next_extra.Set(k, Interface2String(v))
		}
	}

	p, action, marker, err := self.another_object(page, ARGS, &lists, &other)
	if err != nil {
		return err
	}

	if page["alias"] != nil {
		marker = page["alias"].(string)
	}

	OTHER := *self.OTHER
	if OTHER[marker] != nil {
		return nil
	}

	if extra != nil {
		err = InvokeError(p, action, next_extra)
	} else {
		err = InvokeError(p, action)
	}
	if err != nil {
		return err
	}

	if len(lists) > 0 {
		OTHER[marker] = lists
	}
	if len(other) > 0 {
		for k, v := range other {
			OTHER[k] = v
		}
	}
	return nil
}

func (self *Model) CallNextpage(page map[string]interface{}, extra ...url.Values) error {
	LISTS := *self.LISTS
	if len(LISTS) < 1 {
		return nil
	}

	next_extra := make(url.Values)
	var err error
	if extra != nil {
		next_extra, err = url.ParseQuery(extra[0].Encode())
		if err != nil {
			return nil
		}
	}
	if page["manual"] != nil {
		for k, v := range page["manual"].(map[string]interface{}) {
			next_extra.Set(k, Interface2String(v))
		}
	}

	args := self.ARGS
	ARGS, err := url.ParseQuery(args.Encode())
	if err != nil {
		return err
	}
	if ARGS.Get("sortby") != "" {
		ARGS.Del("sortby")
	}
	if ARGS.Get("sortreverse") != "" {
		ARGS.Del("sortreverse")
	}
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	p, action, marker, err := self.another_object(page, ARGS, &lists, &other)
	if err != nil {
		return err
	}

	if page["alias"] != nil {
		marker = page["alias"].(string)
	}

	OTHER := *self.OTHER
	for _, item := range LISTS {
		found := false
		for k, v := range page["relate_item"].(map[string]interface{}) {
			if item[k] != nil {
				found = true
				next_extra.Set(v.(string), Interface2String(item[k]))
			} else {
				next_extra.Del(v.(string))
			}
		}
		if !found {
			continue
		}
		nextextra := make(url.Values)
		if err := InvokeError(p, action, next_extra, nextextra); err != nil {
			return err
		}

		if len(lists) > 0 {
			item[marker] = lists
		}
		if len(other) > 0 {
			for k, v := range other {
				OTHER[k] = v
			}
		}
		lists = make([]map[string]interface{}, 0)
		other = make(map[string]interface{})
	}

	return nil
}

func actionMethod(action string) (string, error) {
	if action == "" {
		return "", Err(1051, "empty action")
	}
	b := []byte(action)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b), nil
}

func (self *Model) ProcessAfter(action string, extra ...url.Values) error {
	if self.Nextpages == nil || self.Nextpages[action] == nil {
		return nil
	}

	for i, page := range self.Nextpages[action] {
		var err error
		if page["relate_item"] == nil {
			if extra != nil && len(extra) >= (i+2) && extra[i+1] != nil {
				err = self.CallOnce(page, extra[i+1])
			} else {
				err = self.CallOnce(page, make(url.Values))
			}
		} else if len(*self.LISTS) == 0 {
			return nil
		} else {
			if extra != nil && len(extra) >= (i+2) && extra[i+1] != nil {
				err = self.CallNextpage(page, extra[i+1])
			} else {
				err = self.CallNextpage(page, make(url.Values))
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *Model) ProperValue(v string, extra url.Values) string {
	ARGS := self.ARGS
	var out string
	if extra == nil {
		out = ARGS.Get(v)
	} else {
		out = extra.Get(v)
		if out == "" {
			out = ARGS.Get(v)
		}
	}
	return out
}

func (self *Model) ProperValues(vs []string, extra url.Values) []string {
	ARGS := self.ARGS
	outs := make([]string, len(vs))
	for i, v := range vs {
		if extra == nil {
			outs[i] = ARGS.Get(v)
		} else {
			outs[i] = extra.Get(v)
			if outs[i] == "" {
				outs[i] = ARGS.Get(v)
			}
		}
	}

	return outs
}
