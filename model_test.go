package genelet

import (
	"net/url"
	"strconv"
	"testing"
)

func TestModelSimple(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	model := new(Model)
	model.DB = db
	model.CurrentTable = "genelet_model_testing"
	model.SORTBY = "sortby"
	model.SORTREVERSE = "sortreverse"
	model.PAGENO = "pageno"
	model.ROWCOUNT = "rowcount"
	model.TOTALNO = "totalno"
	model.MAXPAGENO = "max_pageno"
	model.FIELD = "field"
	model.EMPTIES = "empties"

	ret := model.ExecSQL(`drop table if exists genelet_model_testing`)
	if ret != nil {
		t.Errorf("create table testing failed %s", ret.Error())
	}
	ret = model.ExecSQL(`CREATE TABLE genelet_model_testing (id int auto_increment, x varchar(255), y varchar(255), primary key (id))`)
	if ret != nil {
		t.Errorf("create table testing failed %s", ret.Error())
	}

	args := make(url.Values)
	LISTS := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	storage := make(map[string]interface{})
	model.SetDefaults(args, &LISTS, &other, storage)

	model.CurrentKey = "id"
	model.CurrentIDAuto = "id"
	model.InsertPars = []string{"id", "x", "y"}
	model.TopicsPars = []string{"id", "x", "y"}

	args["x"] = []string{"a"}
	args["y"] = []string{"b"}
	ret = model.Insert()
	if ret != nil {
		t.Errorf("%s insert table testing failed", ret.Error())
	}
	if model.LastID != 1 {
		t.Errorf("%d wanted", model.LastID)
	}
	hash := make(url.Values)
	hash.Set("x", "c")
	hash.Set("y", "d")
	ret = model.InsertHash(hash)
	if ret != nil {
		t.Errorf("%s insert hash table testing failed", ret.Error())
	}
	id := model.LastID
	if id != 2 {
		t.Errorf("%d wanted", id)
	}

	LISTS = LISTS[:0]
	ret = model.Topics()
	if ret != nil {
		t.Errorf("%s topics table testing failed", ret.Error())
	}
	if len(LISTS) != 2 {
		t.Errorf("%d 2 columns wanted", len(LISTS))
	}

	model.UpdatePars = []string{"id", "x", "y"}
	LISTS = LISTS[:0]
	model.EditPars = []string{"id", "x", "y"}
	args.Set("id", "2")
	args["x"] = []string{"c"}
	args["y"] = []string{"z"}
	ret = model.Update()
	if ret != nil {
		t.Errorf("%s update table testing failed", ret.Error())
	}

	LISTS = LISTS[:0]
	ret = model.Edit()
	if ret != nil {
		t.Errorf("%s edit table testing failed", ret.Error())
	}
	if len(LISTS) != 1 {
		t.Errorf("%d records returned from edit", len(LISTS))
	}
	if string(LISTS[0]["x"].(string)) != "c" {
		t.Errorf("%s c wanted", string(LISTS[0]["x"].(string)))
	}
	if string(LISTS[0]["y"].(string)) != "z" {
		t.Errorf("%s z wanted", string(LISTS[0]["y"].(string)))
	}

	LISTS = LISTS[:0]
	ret = model.Topics()
	if ret != nil {
		t.Errorf("%s select table testing failed", ret.Error())
	}
	if len(LISTS) != 2 {
		t.Errorf("%d records returned from topics, should be 2", len(LISTS))
	}
	if string(LISTS[0]["x"].(string)) != "a" {
		t.Errorf("%s a wanted", string(LISTS[0]["x"].(string)))
	}
	if string(LISTS[0]["y"].(string)) != "b" {
		t.Errorf("%s b wanted", string(LISTS[0]["y"].(string)))
	}
	if string(LISTS[1]["x"].(string)) != "c" {
		t.Errorf("%s c wanted", string(LISTS[1]["x"].(string)))
	}
	if string(LISTS[1]["y"].(string)) != "z" {
		t.Errorf("%s z wanted", string(LISTS[1]["y"].(string)))
	}

	args["id"] = []string{"1"}
	ret = model.Delete()
	if ret != nil {
		t.Errorf("%s delete table testing failed", ret.Error())
	}

	LISTS = LISTS[:0]
	ret = model.Topics()
	if ret != nil {
		t.Errorf("%s select table testing failed", ret.Error())
	}
	if len(LISTS) != 1 {
		t.Errorf("%d records returned from edit", len(LISTS))
	}
	if LISTS[0]["id"].(int64) != 2 {
		t.Errorf("%d 2 wanted", LISTS[0]["x"].(int64))
		t.Errorf("%v wanted", LISTS[0]["x"])
	}
	if string(LISTS[0]["x"].(string)) != "c" {
		t.Errorf("%s c wanted", string(LISTS[0]["x"].(string)))
	}
	if string(LISTS[0]["y"].(string)) != "z" {
		t.Errorf("%s z wanted", string(LISTS[0]["y"].(string)))
	}

	args["id"] = []string{"2"}
	ret = model.Insert()
	if ret.Error() == "" {
		t.Errorf("%s wanted", ret.Error())
	}

	args["id"] = []string{"3"}
	args["y"] = []string{"zz"}
	ret = model.Update()
	if ret != nil || model.Affected != 0 {
		t.Errorf("%s %d wanted", ret.Error(), model.Affected)
	}

	model.ExecSQL(`truncate table genelet_model_testing`)
	delete(args, "id")
	for i := 1; i < 100; i++ {
		delete(args, "id")
		args["x"] = []string{"a"}
		args["y"] = []string{"b"}
		LISTS = LISTS[:0]
		ret = model.Insert()
		if ret != nil {
			t.Errorf("%s insert table testing failed", ret.Error())
		}
		if LISTS[0]["id"].(string) != strconv.Itoa(i) {
			t.Errorf("%d %s insert table auto id failed", i, LISTS[0]["id"].(string))
		}
	}

	for i := 1; i < 100; i++ {
		args["id"] = []string{strconv.Itoa(i)}
		args["y"] = []string{"c"}
		LISTS = LISTS[:0]
		ret = model.Update()
		if ret != nil {
			t.Errorf("%s update table testing failed", ret.Error())
		}
		if LISTS[0]["id"].(string) != strconv.Itoa(i) {
			t.Errorf("%d %s update id failed", i, LISTS[0]["id"].(string))
		}
		if LISTS[0]["y"].(string) != "c" {
			t.Errorf("%s update y failed", LISTS[0]["id"].(string))
		}
	}

	for i := 1; i < 100; i++ {
		args["id"] = []string{strconv.Itoa(i)}
		LISTS = LISTS[:0]
		ret = model.Edit()
		if ret != nil {
			t.Errorf("%s edit table testing failed", ret.Error())
		}
		if int(LISTS[0]["id"].(int64)) != i {
			t.Errorf("%d %d edit id failed", i, int(LISTS[0]["id"].(int64)))
		}
		if string(LISTS[0]["y"].(string)) != "c" {
			t.Errorf("%s edit y failed", string(LISTS[0]["id"].(string)))
		}
	}

	args["rowcount"] = []string{"20"}
	model.TotalForce = -1
	LISTS = LISTS[:0]
	ret = model.Topics()
	if ret != nil {
		t.Errorf("%s edit table testing failed", ret.Error())
	}
	a := model.ARGS
	nt, err := strconv.Atoi(a["totalno"][0])
	if err != nil {
		panic(err)
	}
	nm, err := strconv.Atoi(a["max_pageno"][0])
	if err != nil {
		panic(err)
	}
	if nt != 99 {
		t.Errorf("%d total is 99", nt)
	}
	if nm != 5 {
		t.Errorf("%d 5 pages", nm)
	}
	for i := 1; i <= 20; i++ {
		if int(LISTS[i-1]["id"].(int64)) != i {
			t.Errorf("%d %d edit id failed", i, LISTS[i-1]["id"].(int64))
		}
	}

	args["pageno"] = []string{"3"}
	args["rowcount"] = []string{"20"}
	LISTS = LISTS[:0]
	ret = model.Topics()
	if ret != nil {
		t.Errorf("%s topics table testing failed", ret.Error())
	}
	for i := 1; i <= 20; i++ {
		if LISTS[i-1]["id"].(int64) != int64(40+i) {
			t.Errorf("%d %d topics id failed", 40+i, LISTS[i-1]["id"].(int))
		}
	}

	for i := 1; i < 100; i++ {
		args["id"] = []string{strconv.Itoa(i)}
		LISTS = LISTS[:0]
		ret = model.Delete()
		if ret != nil {
			t.Errorf("%s delete table testing failed", ret.Error())
		}
		x := LISTS[0]
		if x["id"].(string) != strconv.Itoa(i) {
			t.Errorf("%d %s delete id failed", i, x["id"].(string))
		}
	}
}
