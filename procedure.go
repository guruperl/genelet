package genelet

// InPars and OutPars must match exactly what needed in procedure and
// the names in OutPars should follow those in Attributes otherwise it
// will not be reagconized
import (
	"database/sql"
	"net/url"
	"strings"
)

type Procedure struct {
	DB *sql.DB
	Ticket
}

func NewProcedure(base Base, db *sql.DB, uri string, provider string) *Procedure {
	a := new(Procedure)
	a.CGI = a
	a.Base = base
	a.DB = db
	a.Uri = uri
	a.Provider = provider
	return a
}

func (self *Procedure) Run_sql(call_name string, in_vals []interface{}) error {
	role := self.C.Roles[self.RoleValue]
	issuer := role.Issuers[self.Provider]
	if (issuer.Screen & 1) != 0 {
		in_vals = append(in_vals, Ip2int(self.GetIP()))
	}
	if (issuer.Screen & 2) != 0 {
		in_vals = append(in_vals, self.Uri)
	}
	//	if (issuer.Screen & 4) !=0 {in_vals= append(in_vals, self.Get_ua())}
	//	if (issuer.Screen & 8) !=0 {in_vals= append(in_vals, self.Get_referer())}
	outPars := issuer.OutPars
	if outPars == nil {
		outPars = role.Attributes
	}

	self.Out_hash = make(map[string]interface{})
	var err error
	dbi := &DBI{DB: self.DB}
	if strings.ToLower(call_name[0:7]) == "select " {
		err = dbi.GetSQLLabel(self.Out_hash, call_name, outPars, in_vals...)
	} else {
		err = dbi.DoProc(self.Out_hash, outPars, call_name, in_vals...)
	}
	if err != nil {
		return Err(1036, err.Error())
	}

	return nil
}

func (self *Procedure) Authenticate(login, passwd string) error {
	role := self.C.Roles[self.RoleValue]
	issuer := role.Issuers[self.Provider]
	if issuer.PasswordHash != "" {
		if login == "" || passwd == "" {
			return Err(1037)
		}
		if err := self.Run_sql(issuer.Sql, []interface{}{login}); err != nil {
			return err
		}
		stored, ok := self.Out_hash[issuer.PasswordHash]
		if !ok || stored == nil {
			return Err(1031)
		}
		if err := CheckPasswordHash(passwd, Interface2String(stored)); err != nil {
			return Err(1031)
		}
		delete(self.Out_hash, issuer.PasswordHash)
		return nil
	}
	return self.Run_sql(issuer.Sql, []interface{}{login, passwd})
}

func (self *Procedure) Authenticate_as(login string) error {
	role := self.C.Roles[self.RoleValue]
	issuer := role.Issuers[self.Provider]
	return self.Run_sql(issuer.Sql_as, []interface{}{login})
}

func (self *Procedure) Callback_address() string {
	http := "http"
	if self.R.TLS != nil {
		http += "s"
	}
	return http + "://" + self.R.Host + self.C.Script + "/" + self.RoleValue + "/" + self.ChartagValue + "/" + self.Provider + "?" + self.C.GoURIName + "=" + url.QueryEscape(self.Uri)
}

func (self *Procedure) Fill_provider(back map[string]interface{}) error {
	role := self.C.Roles[self.RoleValue]
	issuer := role.Issuers[self.Provider]
	in_vals := make([]interface{}, 0)
	for _, par := range issuer.InPars {
		if val, ok := back[par]; ok {
			in_vals = append(in_vals, val)
		} else {
			in_vals = append(in_vals, "")
		}
	}

	if err := self.Run_sql(issuer.Sql, in_vals); err != nil {
		return err
	}

	for _, key := range role.Attributes {
		if _, ok := self.Out_hash[key]; !ok {
			if out, ok := back[key]; ok {
				self.Out_hash[key] = out
			}
		}
	}

	return nil
}
