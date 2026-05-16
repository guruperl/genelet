package genelet

import (
	"encoding/json"
	"fmt"
	"os"
)

type Component struct {
	Actions map[string]map[string][]string
	Fks     map[string][]string

	Nextpages map[string][]map[string]interface{}

	CurrentTable  string            `json:"current_table"`
	CurrentTables []Table           `json:"current_tables"`
	CurrentKey    string            `json:"current_key"`
	CurrentKeys   []string          `json:"current_keys"`
	CurrentIDAuto string            `json:"current_id_auto"`
	KeyIN         map[string]string `json:"key_in"`

	InsertPars []string          `json:"insert_pars"`
	EditPars   []string          `json:"edit_pars"`
	UpdatePars []string          `json:"update_pars"`
	InsupdPars []string          `json:"insupd_pars"`
	TopicsPars []string          `json:"topics_pars"`
	TopicsHash map[string]string `json:"topics_hash"`

	TotalForce  int `json:"total_force"`
	Empties     string
	Fields      string
	Maxpageno   string
	Totalno     string
	Rowcount    string
	Pageno      string
	Sortreverse string
	Sortby      string
}

func NewComponent(filename string) *Component {
	parsed, err := LoadComponent(filename)
	if err != nil {
		panic(err)
	}
	return parsed
}

func LoadComponent(filename string) (*Component, error) {
	var parsed Component
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("load component %q: %w", filename, err)
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, fmt.Errorf("parse component %q: %w", filename, err)
	}

	if parsed.Sortby == "" {
		parsed.Sortby = "sortby"
	}
	if parsed.Sortreverse == "" {
		parsed.Sortreverse = "sortreverse"
	}
	if parsed.Pageno == "" {
		parsed.Pageno = "pageno"
	}
	if parsed.Totalno == "" {
		parsed.Totalno = "totalno"
	}
	if parsed.Rowcount == "" {
		parsed.Rowcount = "rowcount"
	}
	if parsed.Maxpageno == "" {
		parsed.Maxpageno = "maxpage"
	}
	if parsed.Fields == "" {
		parsed.Fields = "fields"
	}
	if parsed.Empties == "" {
		parsed.Empties = "empties"
	}
	if parsed.TotalForce == 0 {
		parsed.TotalForce = 1
	}
	if err := ValidateComponent(&parsed); err != nil {
		return nil, fmt.Errorf("validate component %q: %w", filename, err)
	}

	return &parsed, nil
}
