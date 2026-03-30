package steps

import (
	"reflect"
	"testing"
)

func makeTestOptions() []dbMultiOption {
	return []dbMultiOption{
		{ID: "", Name: "None", Exclusive: true, Checked: true},
		{ID: "postgres", Name: "PostgreSQL", Exclusive: false},
		{ID: "redis", Name: "Redis", Exclusive: false},
		{ID: "sqlite", Name: "SQLite", Exclusive: true},
	}
}

func TestToggle_SelectNetworkDB_DeselectsNone(t *testing.T) {
	opts := makeTestOptions()
	result := toggle(opts, 1) // select postgres
	if !result[1].Checked {
		t.Error("postgres should be checked")
	}
	if result[0].Checked {
		t.Error("None should be unchecked after selecting postgres")
	}
}

func TestToggle_SelectNone_DeselectsAll(t *testing.T) {
	opts := makeTestOptions()
	opts[1].Checked = true  // postgres was on
	opts[0].Checked = false // none was off
	result := toggle(opts, 0) // select None
	if !result[0].Checked {
		t.Error("None should be checked")
	}
	if result[1].Checked {
		t.Error("postgres should be unchecked after selecting None")
	}
}

func TestToggle_SelectSQLite_DeselectsNetworkDB(t *testing.T) {
	opts := makeTestOptions()
	opts[1].Checked = true  // postgres on
	opts[0].Checked = false // none off
	result := toggle(opts, 3) // select SQLite
	if !result[3].Checked {
		t.Error("sqlite should be checked")
	}
	if result[1].Checked {
		t.Error("postgres should be unchecked after selecting SQLite")
	}
}

func TestToggle_SelectTwoNetworkDBs(t *testing.T) {
	opts := makeTestOptions()
	opts = toggle(opts, 1) // select postgres
	opts = toggle(opts, 2) // select redis
	if !opts[1].Checked {
		t.Error("postgres should be checked")
	}
	if !opts[2].Checked {
		t.Error("redis should be checked")
	}
	if opts[0].Checked {
		t.Error("None should be unchecked")
	}
}

func TestToggle_DeselectOption(t *testing.T) {
	opts := makeTestOptions()
	opts = toggle(opts, 1) // select postgres
	opts = toggle(opts, 1) // deselect postgres
	if opts[1].Checked {
		t.Error("postgres should be unchecked")
	}
}

func TestSelectedIDs_NoneOnly(t *testing.T) {
	opts := makeTestOptions() // None is checked by default
	ids := selectedIDs(opts)
	if len(ids) != 0 {
		t.Errorf("expected empty slice when only None is checked, got %v", ids)
	}
}

func TestSelectedIDs_MultipleDBs(t *testing.T) {
	opts := makeTestOptions()
	opts = toggle(opts, 1) // postgres
	opts = toggle(opts, 2) // redis
	ids := selectedIDs(opts)
	want := []string{"postgres", "redis"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("expected %v, got %v", want, ids)
	}
}

func TestAnyChecked_InitialState(t *testing.T) {
	opts := makeTestOptions() // None is checked by default
	if !anyChecked(opts) {
		t.Error("expected anyChecked=true with None selected")
	}
}

func TestAnyChecked_NothingChecked(t *testing.T) {
	opts := makeTestOptions()
	for i := range opts {
		opts[i].Checked = false
	}
	if anyChecked(opts) {
		t.Error("expected anyChecked=false")
	}
}
