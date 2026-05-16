package genelet

import (
	"database/sql"
	"net/url"
	"strconv"
	"strings"
)

type Table struct {
	Name  string
	Alias string
	Type  string
	Using string
	On    string
}

type Crud struct {
	DBI
	CurrentTable  string  `json:"current_table"`
	CurrentTables []Table `json:"current_tables"`
}

func NewCrud(db *sql.DB, table string, tables []Table) *Crud {
	crud := new(Crud)
	crud.DB = db
	crud.CurrentTable = table
	if tables != nil {
		crud.CurrentTables = tables
	}
	return crud
}

func TableString(currentTables []Table) string {
	sql, _ := TableStringSafe(currentTables)
	return sql
}

func TableStringSafe(currentTables []Table) (string, error) {
	sql := ""
	for i, table := range currentTables {
		if err := ValidateSQLIdentifier("table", table.Name); err != nil {
			return "", err
		}
		name := table.Name
		if table.Alias != "" {
			if err := ValidateSQLIdentifier("alias", table.Alias); err != nil {
				return "", err
			}
			name += " " + table.Alias
		}
		if i == 0 {
			sql = name
		} else if table.Using != "" {
			switch table.Type {
			case "INNER", "LEFT", "RIGHT", "FULL", "CROSS":
			default:
				return "", Err(1071, "invalid SQL join type "+table.Type)
			}
			if err := ValidateSQLIdentifier("join field", table.Using); err != nil {
				return "", err
			}
			sql += "\n" + table.Type + " JOIN " + name + " USING (" + table.Using + ")"
		} else {
			switch table.Type {
			case "INNER", "LEFT", "RIGHT", "FULL", "CROSS":
			default:
				return "", Err(1071, "invalid SQL join type "+table.Type)
			}
			if err := ValidateSQLJoinCondition(table.On); err != nil {
				return "", err
			}
			sql += "\n" + table.Type + " JOIN " + name + " ON (" + table.On + ")"
		}
	}

	return sql, nil
}

func SelectLabelString(selectPars interface{}) (string, []string) {
	sql, labels, _ := SelectLabelStringSafe(selectPars)
	return sql, labels
}

func SelectLabelStringSafe(selectPars interface{}) (string, []string, error) {
	select_labels := make([]string, 0)
	sql := ""
	switch selectPars.(type) {
	case []string:
		for _, v := range selectPars.([]string) {
			if err := ValidateSQLQualifiedIdentifier("select field", v); err != nil {
				return "", nil, err
			}
			select_labels = append(select_labels, v)
		}
		sql = strings.Join(select_labels, ", ")
	case map[string]string:
		i := 0
		for key, val := range selectPars.(map[string]string) {
			if err := ValidateSQLSelectExpression(key); err != nil {
				return "", nil, err
			}
			if err := ValidateSQLIdentifier("select label", val); err != nil {
				return "", nil, err
			}
			if i == 0 {
				sql = key
			} else {
				sql += ", " + key
			}
			i++
			select_labels = append(select_labels, val)
		}
	default:
		sql = selectPars.(string)
		if err := ValidateSQLSelectExpression(sql); err != nil {
			return "", nil, err
		}
		select_labels = append(select_labels, sql)
	}
	return sql, select_labels, nil
}

func SelectConditionString(extra url.Values, table ...string) (string, []interface{}) {
	sql, values, _ := SelectConditionStringSafe(extra, table...)
	return sql, values
}

func SelectConditionStringSafe(extra url.Values, table ...string) (string, []interface{}, error) {
	sql := ""
	values := make([]interface{}, 0)
	i := 0
	for field, value := range extra {
		if strings.HasSuffix(field, "_gsql") {
			return "", nil, Err(1071, "raw _gsql conditions are not allowed")
		}
		if i > 0 {
			sql += " AND "
		}
		sql += "("

		if table != nil && table[0] != "" {
			if err := ValidateSQLIdentifier("table", table[0]); err != nil {
				return "", nil, err
			}
			if !strings.Contains(field, ".") {
				field = table[0] + "." + field
			}
		}
		if err := ValidateSQLQualifiedIdentifier("condition field", field); err != nil {
			return "", nil, err
		}
		n := len(value)
		if n > 1 {
			sql += field + " IN (" + strings.Join(strings.Split(strings.Repeat("?", n), ""), ",") + ")"
			for _, v := range value {
				values = append(values, v)
			}
		} else if n == 1 {
			sql += field + " =?"
			values = append(values, value[0])
		}
		sql += ")"
		i++
	}

	return sql, values, nil
}

func SingleConditionString(keyname interface{}, ids []interface{}, extra ...url.Values) (string, []interface{}) {
	sql, values, _ := SingleConditionStringSafe(keyname, ids, extra...)
	return sql, values
}

func SingleConditionStringSafe(keyname interface{}, ids []interface{}, extra ...url.Values) (string, []interface{}, error) {
	sql := ""
	extraValues := make([]interface{}, 0)

	switch keyname.(type) {
	case []string:
		for i, item := range keyname.([]string) {
			if err := ValidateSQLQualifiedIdentifier("key field", item); err != nil {
				return "", nil, err
			}
			val := ids[i]
			if i == 0 {
				sql = "("
			} else {
				sql += " AND "
			}
			switch val.(type) {
			case []interface{}:
				n := len(val.([]interface{}))
				sql += item + " IN (" + strings.Join(strings.Split(strings.Repeat("?", n), ""), ",") + ")"
				for _, v := range val.([]interface{}) {
					extraValues = append(extraValues, v)
				}
			default:
				sql += item + " =?"
				extraValues = append(extraValues, val)
			}
		}
		sql += ")"
	case string:
		if err := ValidateSQLQualifiedIdentifier("key field", keyname.(string)); err != nil {
			return "", nil, err
		}
		n := len(ids)
		if n > 1 {
			sql = "(" + keyname.(string) + " IN (" + strings.Join(strings.Split(strings.Repeat("?", n), ""), ",") + "))"
		} else {
			sql = "(" + keyname.(string) + "=?)"
		}
		for _, v := range ids {
			extraValues = append(extraValues, v)
		}
	}

	if extra != nil && len(extra) > 0 {
		s, arr, err := SelectConditionStringSafe(extra[0])
		if err != nil {
			return "", nil, err
		}
		if s != "" {
			sql += " AND " + s
			for _, v := range arr {
				extraValues = append(extraValues, v)
			}
		}
	}

	return sql, extraValues, nil
}

func (self *Crud) InsertHash(fieldValues url.Values) error {
	return self.insertHash_("INSERT", fieldValues)
}

func (self *Crud) ReplaceHash(fieldValues url.Values) error {
	return self.insertHash_("REPLACE", fieldValues)
}

func (self *Crud) insertHash_(how string, fieldValues url.Values) error {
	if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
		return err
	}
	fields := make([]string, 0)
	values := make([]interface{}, 0)
	for k, v := range fieldValues {
		if err := ValidateSQLIdentifier("field", k); err != nil {
			return err
		}
		fields = append(fields, k)
		values = append(values, v[0])
	}
	sql := how + " INTO " + self.CurrentTable + " (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(strings.Split(strings.Repeat("?", len(fields)), ""), ",") + ")"
	return self.DoSQL(sql, values...)
}

func (self *Crud) UpdateHash(fieldValues url.Values, keyname interface{}, ids []interface{}, extra ...url.Values) error {
	empties := make([]string, 0)
	return self.UpdateHashNulls(fieldValues, keyname, ids, empties, extra...)
}

func (self *Crud) UpdateHashNulls(fieldValues url.Values, keyname interface{}, ids []interface{}, empties []string, extra ...url.Values) error {
	if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
		return err
	}
	field0 := make([]string, 0)
	values := make([]interface{}, 0)
	for k, v := range fieldValues {
		if err := ValidateSQLIdentifier("field", k); err != nil {
			return err
		}
		field0 = append(field0, k+"=?")
		values = append(values, v[0])
	}

	sql := "UPDATE " + self.CurrentTable + " SET " + strings.Join(field0, ", ")
	for _, v := range empties {
		if fieldValues.Get(v) != "" {
			continue
		}
		switch keyname.(type) {
		case []string:
			if Grep(keyname.([]string), v) {
				continue
			}
		case string:
			if v == keyname.(string) {
				continue
			}
		}
		if err := ValidateSQLIdentifier("field", v); err != nil {
			return err
		}
		sql += ", " + v + "=NULL"
	}

	where, extraValues, err := SingleConditionStringSafe(keyname, ids, extra...)
	if err != nil {
		return err
	}
	if where != "" {
		sql += "\nWHERE " + where
	}
	for _, v := range extraValues {
		values = append(values, v)
	}

	return self.DoSQL(sql, values...)
}

func (self *Crud) InsupdTable(fieldValues url.Values, keyname string, uniques []string, s_hash *string) error {
	if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
		return err
	}
	if err := ValidateSQLIdentifier("field", keyname); err != nil {
		return err
	}
	s := "SELECT " + keyname + " FROM " + self.CurrentTable + "\nWHERE "
	v := make([]interface{}, 0)
	for i, val := range uniques {
		if err := ValidateSQLIdentifier("field", val); err != nil {
			return err
		}
		if i > 0 {
			s += " AND "
		}
		s += val + "=?"
		v = append(v, fieldValues.Get(val))
	}

	lists := make([]map[string]interface{}, 0)
	err := self.SelectSQL(&lists, s, v...)
	if err != nil {
		return err
	}
	if len(lists) > 1 {
		return Err(1070)
	}

	if len(lists) == 1 {
		id := lists[0][keyname].(int64)
		err = self.UpdateHash(fieldValues, keyname, []interface{}{id}, nil)
		if err != nil {
			return err
		}
		*s_hash = "update"
		fieldValues.Set(keyname, strconv.FormatInt(id, 10))
	} else {
		err = self.InsertHash(fieldValues)
		if err != nil {
			return err
		}
		*s_hash = "insert"
		fieldValues.Set(keyname, strconv.FormatInt(self.LastID, 10))
	}

	return nil
}

func (self *Crud) InsupdHash(fieldValues url.Values, upd_fieldValues url.Values, keyname interface{}, uniques []string, s_hash *string) error {
	if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
		return err
	}
	var ks []string
	switch keyname.(type) {
	case []string:
		ks = keyname.([]string)
	default:
		ks = []string{keyname.(string)}
	}
	if err := ValidateSQLIdentifierList("field", ks); err != nil {
		return err
	}
	s := "SELECT " + strings.Join(ks, ",") + " FROM " + self.CurrentTable + "\nWHERE "
	v := make([]interface{}, 0)
	for i, val := range uniques {
		if err := ValidateSQLIdentifier("field", val); err != nil {
			return err
		}
		if i > 0 {
			s += " AND "
		}
		s += val + "=?"
		v = append(v, fieldValues.Get(val))
	}

	lists := make([]map[string]interface{}, 0)
	err := self.SelectSQL(&lists, s, v...)
	if err != nil {
		return err
	}
	if len(lists) > 1 {
		return Err(1070)
	}

	if len(lists) == 1 {
		ids := make([]interface{}, len(ks))
		for i, k := range ks {
			ids[i] = lists[0][k]
			fieldValues.Set(k, Interface2String(ids[i]))
		}
		err = self.UpdateHash(upd_fieldValues, keyname, ids, nil)
		if err != nil {
			return err
		}
		*s_hash = "update"
	} else {
		err = self.InsertHash(fieldValues)
		if err != nil {
			return err
		}
		*s_hash = "insert"
	}

	return nil
}

func (self *Crud) DeleteHash(keyname interface{}, ids []interface{}, extra ...url.Values) error {
	if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
		return err
	}
	sql := "DELETE FROM " + self.CurrentTable
	where, extraValues, err := SingleConditionStringSafe(keyname, ids, extra...)
	if err != nil {
		return err
	}
	if where != "" {
		sql += "\nWHERE " + where
	}

	return self.DoSQL(sql, extraValues...)
}

func (self *Crud) EditHash(lists *[]map[string]interface{}, selectPars interface{}, keyname interface{}, ids []interface{}, extra ...url.Values) error {
	if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
		return err
	}
	sql, select_labels, err := SelectLabelStringSafe(selectPars)
	if err != nil {
		return err
	}
	sql = "SELECT " + sql + "\nFROM " + self.CurrentTable
	where, extraValues, err := SingleConditionStringSafe(keyname, ids, extra...)
	if err != nil {
		return err
	}
	if where != "" {
		sql += "\nWHERE " + where
	}

	return self.SelectSQLLabel(lists, sql, select_labels, extraValues...)
}

func (self *Crud) TopicsHash(lists *[]map[string]interface{}, selectPars interface{}, extra ...url.Values) error {
	return self.TopicsHashOrder(lists, selectPars, "", extra...)
}

func (self *Crud) TopicsHashOrder(lists *[]map[string]interface{}, selectPars interface{}, order string, extra ...url.Values) error {
	sql, select_labels, err := SelectLabelStringSafe(selectPars)
	if err != nil {
		return err
	}
	table := ""
	if len(self.CurrentTables) > 0 {
		tables, err := TableStringSafe(self.CurrentTables)
		if err != nil {
			return err
		}
		sql = "SELECT " + sql + "\nFROM " + tables
		table = self.CurrentTables[0].Alias
		if table == "" {
			table = self.CurrentTables[0].Name
		}
	} else {
		if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
			return err
		}
		sql = "SELECT " + sql + "\nFROM " + self.CurrentTable
	}

	if len(extra) > 0 {
		where, values, err := SelectConditionStringSafe(extra[0], table)
		if err != nil {
			return err
		}
		if where != "" {
			sql += "\nWHERE " + where
		}
		if order != "" {
			sql += "\n" + order
		}
		return self.SelectSQLLabel(lists, sql, select_labels, values...)
	}

	if order != "" {
		sql += "\n" + order
	}
	return self.SelectSQLLabel(lists, sql, select_labels)
}

func (self *Crud) TotalHash(hash map[string]interface{}, label string, extra ...url.Values) error {
	if err := ValidateSQLIdentifier("label", label); err != nil {
		return err
	}
	table := ""
	sql := "SELECT COUNT(*) FROM "
	if self.CurrentTables != nil {
		tables, err := TableStringSafe(self.CurrentTables)
		if err != nil {
			return err
		}
		sql += tables
		table = self.CurrentTables[0].Alias
		if table == "" {
			table = self.CurrentTables[0].Name
		}
	} else {
		if err := ValidateSQLIdentifier("table", self.CurrentTable); err != nil {
			return err
		}
		sql += self.CurrentTable
	}

	if len(extra) > 0 {
		where, values, err := SelectConditionStringSafe(extra[0], table)
		if err != nil {
			return err
		}
		if where != "" {
			sql += "\nWHERE " + where
		}
		return self.GetSQLLabel(hash, sql, []string{label}, values...)
	}

	return self.GetSQLLabel(hash, sql, []string{label})
}
