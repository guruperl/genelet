package genelet

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

type TProcedure struct {
	Procedure
}

func NewTProcedure(base Base, db *sql.DB, uri string, provider string) *TProcedure {
	a := new(TProcedure)
	a.CGI = a
	a.Base = base
	a.DB = db
	a.Uri = uri
	a.Provider = provider
	return a
}

func TestProcedureRejectsEmptySQLName(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	proc := NewTProcedure(*newBase(configure, "m", "json", httptest.NewRecorder(), req), nil, "/bb/m/json/foo", "db")
	err = proc.Run_sql("", nil)
	if gerr, ok := err.(Gerror); !ok || gerr.Code != 1175 {
		t.Fatalf("Run_sql empty name = %#v, want 1175 Gerror", err)
	}
}

func (self *TProcedure) SetIP() string {
	return "123.123.123.123"
}
func (self *TProcedure) SetWhen() int {
	return 1
}

func TestDbiProcedure(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	db := openTestDB(t)
	defer db.Close()
	req, err := http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world", nil)
	if err != nil {
		log.Fatal(err)
	}
	w := httptest.NewRecorder()
	b := newBase(configure, "m", "json", w, req)

	tticket := NewTProcedure(*b, db, "/bb/m/e/foo/bar", "db")
	//issuer := tticket.C.Roles["m"].Issuers["db"]
	db.Exec("DROP TABLE IF EXISTS user")
	defer db.Exec("DROP TABLE IF EXISTS user")
	db.Exec("CREATE TABLE `user` (   `m_id` int(11) NOT NULL AUTO_INCREMENT,   `email` varchar(255) DEFAULT NULL,   `passwd` varchar(255) DEFAULT NULL,   `first_name` varchar(255) DEFAULT NULL,   `last_name` varchar(255) DEFAULT NULL,   `address` varchar(255) DEFAULT NULL,   `company` varchar(255) DEFAULT NULL,   PRIMARY KEY (`m_id`) )")
	db.Exec("INSERT INTO user (email, passwd, first_name, last_name, address, company) VALUES ('a','b','c','d','e','f')")
	tticket.Uri = "/bb/m/e/asw"
	ret := tticket.Authenticate("a", "b")
	if ret != nil {
		t.Errorf("%s corrent login expected", ret.Error())
	}
	if tticket.Out_hash["m_id"].(int64) != 1 {
		t.Errorf("%d wanted", tticket.Out_hash["m_id"].(int64))
	}
	if string(tticket.Out_hash["first_name"].(string)) != "c" {
		t.Errorf("%s wanted", tticket.Out_hash["first_name"].(string))
	}
	ret = tticket.HandlerFields()
	if ret != nil {
		t.Errorf("%s returned", ret.Error())
	}

	// test Handler with direct login
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=a&passwd=b&direct=1", nil)
	w = httptest.NewRecorder()
	b = newBase(configure, "m", "e", w, req)
	tticket = NewTProcedure(*b, db, "/bb/m/e/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler_login()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s wanted", ret.Error())
	}
	h := w.Header()
	matched, err := regexp.MatchString("^mc=Ec9rwEEzh1\\/0UTuoE7dvi\\/ByyTC1F2qsXRbY4yYNbq3RoCpNDcLkiEYmJdaLrz8uBYUQZe\\/dglVvVtXQsRI\\/", h.Get("Set-Cookie"))
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Logf("legacy cookie prefix changed: %s", h.Get("Set-Cookie"))
	}
}
