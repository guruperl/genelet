package genelet

import (
	"database/sql"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

type DBI struct {
	DB       *sql.DB
	Driver   string
	LastID   int64
	Affected int64

	Logger *zap.Logger
}

func (self *DBI) SetDriver(driver string) {
	self.Driver = NormalizeDriver(driver)
}

func (self *DBI) sql(query string) string {
	return RebindSQL(self.Driver, query)
}

func (self *DBI) ExecSQL(sql string) error {
	_, err := self.DB.Exec(self.sql(sql))

	return err
}

func (self *DBI) DoSQL(sql string, args ...interface{}) error {
	if self.Logger != nil {
		glog := self.Logger.Sugar()
		glog.Infof("%s\n", self.sql(sql))
		glog.Infof("%#v\n", args)
	}
	sth, err := self.DB.Prepare(self.sql(sql))
	if err != nil {
		return err
	}
	defer sth.Close()
	result, err := sth.Exec(args...)
	if err != nil {
		return err
	}
	lastID, err := result.LastInsertId()
	if err == nil {
		self.LastID = lastID
	}
	affected, err := result.RowsAffected()
	if err == nil {
		self.Affected = affected
	}

	return nil
}

func (self *DBI) DoSQLReturning(sql string, returnLabel string, args ...interface{}) error {
	if returnLabel == "" || NormalizeDriver(self.Driver) != "postgres" {
		return self.DoSQL(sql, args...)
	}
	if err := ValidateSQLIdentifier("returning field", returnLabel); err != nil {
		return err
	}
	query := sql + " RETURNING " + returnLabel
	if self.Logger != nil {
		glog := self.Logger.Sugar()
		glog.Infof("%s\n", self.sql(query))
		glog.Infof("%#v\n", args)
	}
	sth, err := self.DB.Prepare(self.sql(query))
	if err != nil {
		return err
	}
	defer sth.Close()
	if err := sth.QueryRow(args...).Scan(&self.LastID); err != nil {
		return err
	}
	self.Affected = 1
	return nil
}

func (self *DBI) DoSQLs(sql string, args [][]interface{}) error {
	if self.Logger != nil {
		glog := self.Logger.Sugar()
		glog.Infof("%s\n", self.sql(sql))
		glog.Infof("%#v\n", args)
	}
	sth, err := self.DB.Prepare(self.sql(sql))
	if err != nil {
		return err
	}
	defer sth.Close()
	for _, v := range args {
		result, err := sth.Exec(v...)
		if err != nil {
			return err
		}
		lastID, err := result.LastInsertId()
		if err == nil {
			self.LastID = lastID
		}
		affected, err := result.RowsAffected()
		if err == nil {
			self.Affected = affected
		}
	}

	return nil
}

func (self *DBI) GetSQL(res map[string]interface{}, sql string, args ...interface{}) error {
	return self.GetSQLLabel(res, sql, nil, args...)
}
func (self *DBI) GetArgs(ARGS url.Values, sql string, args ...interface{}) error {
	res := make(map[string]interface{})
	if err := self.GetSQL(res, sql, args...); err != nil {
		return err
	}
	for k, v := range res {
		if v == nil {
			continue
		}
		ARGS.Set(k, Interface2String(v))
	}
	return nil
}

func (self *DBI) SelectSQL(lists *[]map[string]interface{}, sql string, args ...interface{}) error {
	return self.SelectSQLLabel(lists, sql, nil, args...)
}

func (self *DBI) GetSQLLabel(res map[string]interface{}, sql string, selectLabels []string, args ...interface{}) error {
	lists := make([]map[string]interface{}, 0)
	err := self.SelectSQLLabel(&lists, sql, selectLabels, args...)
	if err != nil {
		return err
	}
	if len(lists) == 1 {
		for k, v := range lists[0] {
			res[k] = v
		}
	}
	return nil
}

func (self *DBI) SelectSQLLabel(lists *[]map[string]interface{}, sql string, selectLabels []string, args ...interface{}) error {
	if self.Logger != nil {
		glog := self.Logger.Sugar()
		glog.Infof("%s\n", self.sql(sql))
		glog.Infof("%v\n", args)
	}
	sth, err := self.DB.Prepare(self.sql(sql))
	if err != nil {
		return err
	}
	defer sth.Close()
	rows, err := sth.Query(args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if selectLabels == nil {
		selectLabels, err = rows.Columns()
		if err != nil {
			return err
		}
	}
	names := make([]interface{}, len(selectLabels))
	x := make([]interface{}, len(selectLabels))
	for j := range selectLabels {
		x[j] = &names[j]
	}
	for rows.Next() {
		err = rows.Scan(x...)
		if err != nil {
			return err
		}
		res := make(map[string]interface{})
		for j, v := range selectLabels {
			name := names[j]
			if name != nil {
				switch t := name.(type) {
				case []uint8:
					res[v] = string(t)
					//				case int64:
					//					res[v] = strconv.FormatInt(name.(int64), 10)
				default:
					res[v] = name
				}
				//			} else {
				//				res[v] = nil
			}
		}
		*lists = append(*lists, res)
	}
	return rows.Err()
}

func (self *DBI) DoProc(hash map[string]interface{}, names []string, procName string, args ...interface{}) error {
	if err := ValidateSQLQualifiedIdentifier("procedure", procName); err != nil {
		return err
	}
	n := len(args)
	strG := strings.Join(strings.Split(strings.Repeat("?", n), ""), ",")
	switch NormalizeDriver(self.Driver) {
	case "postgres":
		lists := make([]map[string]interface{}, 0)
		if err := self.SelectSQLLabel(&lists, "SELECT * FROM "+procName+"("+strG+")", names, args...); err != nil {
			return err
		}
		if hash != nil && len(lists) == 1 {
			for k, v := range lists[0] {
				hash[k] = v
			}
		}
		return nil
	case "sqlite3":
		return Err(1175, "stored procedures are not supported by sqlite")
	}
	str := "CALL " + procName + "(" + strG
	strN := ""
	if names != nil {
		strN = "@" + strings.Join(names, ",@")
		str += ", " + strN
	}
	str += ")"

	err := self.DoSQL(str, args...)
	if err != nil {
		return err
	}
	if names == nil || hash == nil {
		return nil
	}
	return self.GetSQLLabel(hash, "SELECT "+strN, names)
}

func (self *DBI) SelectProc(lists *[]map[string]interface{}, procName string, args ...interface{}) error {
	return self.SelectDoProcLabel(lists, nil, nil, procName, nil, args...)
}

func (self *DBI) SelectProcLabel(lists *[]map[string]interface{}, procName string, selectLabels []string, args ...interface{}) error {
	return self.SelectDoProcLabel(lists, nil, nil, procName, selectLabels, args...)
}

func (self *DBI) SelectDoProc(lists *[]map[string]interface{}, hash map[string]interface{}, names []string, procName string, args ...interface{}) error {
	return self.SelectDoProcLabel(lists, hash, names, procName, nil, args...)
}

func (self *DBI) SelectDoProcLabel(lists *[]map[string]interface{}, hash map[string]interface{}, names []string, procName string, selectLabels []string, args ...interface{}) error {
	if err := ValidateSQLQualifiedIdentifier("procedure", procName); err != nil {
		return err
	}
	n := len(args)
	strG := strings.Join(strings.Split(strings.Repeat("?", n), ""), ",")
	switch NormalizeDriver(self.Driver) {
	case "postgres":
		if selectLabels == nil && names != nil {
			selectLabels = names
		}
		err := self.SelectSQLLabel(lists, "SELECT * FROM "+procName+"("+strG+")", selectLabels, args...)
		if err != nil {
			return err
		}
		if hash != nil && len(*lists) == 1 {
			for k, v := range (*lists)[0] {
				hash[k] = v
			}
		}
		return nil
	case "sqlite3":
		return Err(1175, "stored procedures are not supported by sqlite")
	}
	str := "CALL " + procName + "(" + strG
	strN := ""
	if names != nil {
		strN = "@" + strings.Join(names, ",@")
		str += ", " + strN
	}
	str += ")"

	err := self.SelectSQLLabel(lists, str, selectLabels, args...)
	if err != nil {
		return err
	}
	if hash == nil {
		return nil
	}
	if names == nil {
		return nil
	}
	return self.GetSQLLabel(hash, "SELECT "+strN, names)
}
