package genelet

import (
	"fmt"
	"regexp"
	"strings"
)

var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ValidSQLIdentifier(name string) bool {
	return sqlIdentifierPattern.MatchString(name)
}

func ValidateSQLIdentifier(kind, name string) error {
	if !ValidSQLIdentifier(name) {
		return Err(1071, fmt.Sprintf("invalid SQL %s identifier %q", kind, name))
	}
	return nil
}

func ValidateSQLQualifiedIdentifier(kind, name string) error {
	if name == "" {
		return Err(1071, fmt.Sprintf("empty SQL %s identifier", kind))
	}
	for _, part := range strings.Split(name, ".") {
		if err := ValidateSQLIdentifier(kind, part); err != nil {
			return err
		}
	}
	return nil
}

func ValidateSQLIdentifierList(kind string, names []string) error {
	for _, name := range names {
		if err := ValidateSQLQualifiedIdentifier(kind, name); err != nil {
			return err
		}
	}
	return nil
}

func ValidateSQLSelectExpression(expr string) error {
	if expr == "" {
		return Err(1071, "empty SQL select expression")
	}
	if strings.Contains(expr, ";") || strings.Contains(expr, "--") || strings.Contains(expr, "/*") || strings.Contains(expr, "*/") {
		return Err(1071, fmt.Sprintf("invalid SQL select expression %q", expr))
	}
	return nil
}

func ValidateSQLJoinCondition(cond string) error {
	if cond == "" {
		return Err(1071, "empty SQL join condition")
	}
	for _, r := range cond {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("_.=() \t\n\r", r):
		default:
			return Err(1071, fmt.Sprintf("invalid SQL join condition %q", cond))
		}
	}
	return nil
}

func ValidateSQLOrderBy(order string) error {
	if order == "" {
		return nil
	}
	for _, field := range strings.Split(order, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			return Err(1071, "empty SQL order field")
		}
		if err := ValidateSQLQualifiedIdentifier("order", field); err != nil {
			return err
		}
	}
	return nil
}

func ValidateComponent(comp *Component) error {
	if comp == nil {
		return Err(1071, "nil component")
	}
	if comp.CurrentTable != "" {
		if err := ValidateSQLIdentifier("table", comp.CurrentTable); err != nil {
			return err
		}
	}
	if len(comp.CurrentTables) > 0 {
		if _, err := TableStringSafe(comp.CurrentTables); err != nil {
			return err
		}
	}
	for _, fields := range [][]string{
		comp.CurrentKeys,
		comp.InsertPars,
		comp.EditPars,
		comp.UpdatePars,
		comp.InsupdPars,
		comp.TopicsPars,
	} {
		if err := ValidateSQLIdentifierList("field", fields); err != nil {
			return err
		}
	}
	if comp.CurrentKey != "" {
		if err := ValidateSQLIdentifier("field", comp.CurrentKey); err != nil {
			return err
		}
	}
	if comp.CurrentIDAuto != "" {
		if err := ValidateSQLIdentifier("field", comp.CurrentIDAuto); err != nil {
			return err
		}
	}
	for table, field := range comp.KeyIN {
		if err := ValidateSQLIdentifier("table", table); err != nil {
			return err
		}
		if err := ValidateSQLIdentifier("field", field); err != nil {
			return err
		}
	}
	for expr, label := range comp.TopicsHash {
		if err := ValidateSQLSelectExpression(expr); err != nil {
			return err
		}
		if err := ValidateSQLIdentifier("label", label); err != nil {
			return err
		}
	}
	return nil
}
