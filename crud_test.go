package genelet

import (
	"net/url"
	"testing"
)

/*
	func TestCrudStr(t *testing.T) {
		currentTables := []Table{{Name:"user", Alias:"u"}, {Name:"parent", Alias:"p", Type:"INNER", On:"u.parent_id=p.parent_id"}, {Name:"education", Alias:"e", Type: "LEFT", Using: "edu_id"}}
		str := TableString(currentTables)
		if (str != "user u\nINNER JOIN parent p ON (u.parent_id=p.parent_id)\nLEFT JOIN education e USING (edu_id)") {
			t.Errorf("%s wanted", str)
		}

		select_par :=  "firstname"
		sql, labels := SelectLabelString(select_par)
		if (sql != "firstname") {
			t.Errorf("%s wanted", sql)
		}
		if (labels[0] != "firstname") {
			t.Errorf("%s wanted", labels[0])
		}

		selectPars :=  []string{"firstname", "lastname", "id"}
		sql, labels = SelectLabelString(selectPars)
		if (sql != "firstname, lastname, id") {
			t.Errorf("%s wanted", sql)
		}
		if (labels[2] != "id") {
			t.Errorf("%s wanted", labels[2])
		}

		select_hash :=  map[string]string{"firstname":"First", "lastname":"Last", "id":"ID"}
		sql, labels = SelectLabelString(select_hash)
		if (sql != "firstname, lastname, id") {
			t.Errorf("%s wanted", sql)
		}
		if (labels[0] != "First") {
			t.Errorf("%s wanted", labels[0])
		}
		if (labels[1] != "Last") {
			t.Errorf("%s wanted", labels[1])
		}
		if (labels[2] != "ID") {
			t.Errorf("%s wanted", labels[2])
		}

		extra := map[string]interface{}{"firstname":"Peter"}
		sql, c := SelectConditionString(extra)
		if (sql != "(firstname =?)") {
			t.Errorf("%s wanted", sql)
		}
		if (c[0].(string) != "Peter") {
			t.Errorf("%s wanted", c[0].(string))
		}

		sql, c = SelectConditionString(extra, "user")
		if (sql != "(user.firstname =?)") {
			t.Errorf("%s wanted", sql)
		}
		if (c[0].(string) != "Peter") {
			t.Errorf("%s wanted", c[0].(string))
		}


		extra = map[string]interface{}{"firstname":"Peter", "lastname":"Tong", "id":[]interface{}{1,2,3,4}}
		sql, c = SelectConditionString(extra)
		if (sql != "(firstname =?) AND (lastname =?) AND (id IN (?,?,?,?))") {
			t.Errorf("%s wanted", sql)
		}
		if (c[0].(string) != "Peter") {
			t.Errorf("%s wanted", c[0].(string))
		}
		if (c[1].(string) != "Tong") {
			t.Errorf("%s wanted", c[1].(string))
		}
		if (c[2].(int) != 1) {
			t.Errorf("%d wanted", c[2].(int))
		}
		if (c[3].(int) != 2) {
			t.Errorf("%d wanted", c[3].(int))
		}
		if (c[4].(int) != 3) {
			t.Errorf("%d wanted", c[4].(int))
		}
		if (c[5].(int) != 4) {
			t.Errorf("%d wanted", c[5].(int))
		}


		keyname := []string{"user_id","edu_id"}
		ids := []interface{}{[]interface{}{11,22},[]interface{}{33,44,55}}
		s, arr := SingleConditionString(keyname, ids, extra)
		if (s != "(user_id IN (?,?) AND edu_id IN (?,?,?)) AND (firstname =?) AND (lastname =?) AND (id IN (?,?,?,?))") {
			t.Errorf("%s wanted", s)
		}
		if (arr[0].(int) != 11) {
			t.Errorf("%d wanted", arr[0].(int))
		}
		if (arr[1].(int) != 22) {
			t.Errorf("%d wanted", arr[1].(int))
		}
		if (arr[2].(int) != 33) {
			t.Errorf("%d wanted", arr[2].(int))
		}
		if (arr[3].(int) != 44) {
			t.Errorf("%d wanted", arr[3].(int))
		}
		if (arr[4].(int) != 55) {
			t.Errorf("%d wanted", arr[4].(int))
		}
		if (arr[5] != "Peter") {
			t.Errorf("%s wanted", arr[5])
		}
	}
*/
func TestCrudDb(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	crud := NewCrud(db, "atesting", nil)

	crud.ExecSQL(`drop table if exists atesting`)
	ret := crud.ExecSQL(`drop table if exists testing`)
	if ret != nil {
		t.Errorf("create table testing failed %s", ret.Error())
	}
	ret = crud.ExecSQL(`CREATE TABLE atesting (id int auto_increment, x varchar(255), y varchar(255), primary key (id))`)
	if ret != nil {
		t.Errorf("create table atesting failed")
	}
	hash := make(url.Values)
	hash.Set("x", "a")
	hash.Set("y", "b")
	ret = crud.InsertHash(hash)
	if ret != nil {
		t.Fatal(ret)
	}
	if crud.LastID != 1 {
		t.Errorf("%d wanted", crud.LastID)
	}
	hash.Set("x", "c")
	hash.Set("y", "d")
	ret = crud.InsertHash(hash)
	if ret != nil {
		t.Fatal(ret)
	}
	id := crud.LastID
	if id != 2 {
		t.Errorf("%d wanted", id)
	}
	hash1 := make(url.Values)
	hash1.Set("y", "z")
	ret = crud.UpdateHash(hash1, "id", []interface{}{id})
	if ret != nil {
		t.Errorf("%s update table testing failed", ret.Error())
	}

	lists := make([]map[string]interface{}, 0)
	label := []string{"x", "y"}
	ret = crud.EditHash(&lists, label, "id", []interface{}{id})
	if ret != nil {
		t.Errorf("%s select table testing failed", ret.Error())
	}
	if len(lists) != 1 {
		t.Errorf("%d records returned from edit", len(lists))
	}
	if lists[0]["x"].(string) != "c" {
		t.Errorf("%s c wanted", lists[0]["x"].(string))
	}
	if lists[0]["y"].(string) != "z" {
		t.Errorf("%s z wanted", string(lists[0]["y"].(string)))
	}

	lists = make([]map[string]interface{}, 0)
	ret = crud.TopicsHash(&lists, label)
	if ret != nil {
		t.Errorf("%s select table testing failed", ret.Error())
	}
	if len(lists) != 2 {
		t.Errorf("%d records returned from edit, should be 2", len(lists))
	}
	if string(lists[0]["x"].(string)) != "a" {
		t.Errorf("%s a wanted", string(lists[0]["x"].(string)))
	}
	if string(lists[0]["y"].(string)) != "b" {
		t.Errorf("%s b wanted", string(lists[0]["y"].(string)))
	}
	if string(lists[1]["x"].(string)) != "c" {
		t.Errorf("%s c wanted", string(lists[1]["x"].(string)))
	}
	if string(lists[1]["y"].(string)) != "z" {
		t.Errorf("%s z wanted", string(lists[1]["y"].(string)))
	}

	what := make(map[string]interface{})
	ret = crud.TotalHash(what, "total")
	if ret != nil {
		t.Errorf("%s total table testing failed", ret.Error())
	}
	if what["total"].(int64) != 2 {
		t.Errorf("%d total table testing failed", what["total"].(int64))
	}

	ret = crud.DeleteHash("id", []interface{}{1})
	if ret != nil {
		t.Errorf("%s delete table testing failed", ret.Error())
	}

	lists = make([]map[string]interface{}, 0)
	label = []string{"id", "x", "y"}
	ret = crud.TopicsHash(&lists, label)
	if ret != nil {
		t.Errorf("%s select table testing failed", ret.Error())
	}
	if len(lists) != 1 {
		t.Errorf("%d records returned from edit", len(lists))
	}
	if lists[0]["id"].(int64) != 2 {
		t.Errorf("%d 2 wanted", lists[0]["x"].(int32))
		t.Errorf("%v wanted", lists[0]["x"])
	}
	if string(lists[0]["x"].(string)) != "c" {
		t.Errorf("%s c wanted", string(lists[0]["x"].(string)))
	}
	if string(lists[0]["y"].(string)) != "z" {
		t.Errorf("%s z wanted", string(lists[0]["y"].(string)))
	}

	hash = make(url.Values)
	hash.Set("id", "2")
	hash.Set("x", "a")
	hash.Set("y", "b")
	ret = crud.InsertHash(hash)
	if ret.Error() == "" {
		t.Errorf("%s wanted", ret.Error())
	}

	hash1 = make(url.Values)
	hash1.Set("y", "zz")
	ret = crud.UpdateHash(hash1, "id", []interface{}{3})
	if ret != nil {
		t.Errorf("%s wanted", ret.Error())
	}
	if crud.Affected != 0 {
		t.Errorf("%d wanted", crud.Affected)
	}
	db.Close()
}
