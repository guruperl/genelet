package genelet

import (
	"net/url"
	"testing"
)

func TestSQLIdentifierRejection(t *testing.T) {
	if err := ValidateSQLIdentifier("field", "safe_name_1"); err != nil {
		t.Fatalf("safe identifier rejected: %v", err)
	}
	if err := ValidateSQLIdentifier("field", "name;DROP"); err == nil {
		t.Fatal("unsafe identifier accepted")
	}

	extra := url.Values{}
	extra.Set("name_gsql", "1=1")
	if _, _, err := SelectConditionStringSafe(extra); err == nil {
		t.Fatal("_gsql condition accepted")
	}

	extra = url.Values{}
	extra.Set("name;DROP", "x")
	if _, _, err := SelectConditionStringSafe(extra); err == nil {
		t.Fatal("unsafe condition field accepted")
	}
}

func TestOrderStringRejectsUnsafeSort(t *testing.T) {
	model := &Model{
		ARGS:       url.Values{},
		SORTBY:     "sortby",
		ROWCOUNT:   "rowcount",
		PAGENO:     "pageno",
		CurrentKey: "id",
	}
	model.CurrentTable = "users"
	model.ARGS.Set("sortby", "id;DROP")

	if _, err := model.GetOrderStringChecked(); err == nil {
		t.Fatal("unsafe sort accepted")
	}
}
