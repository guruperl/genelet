package genelet

import (
	"strconv"
	"strings"
)

func NormalizeDriver(driver string) string {
	switch strings.ToLower(driver) {
	case "postgres", "postgresql", "pg", "pq":
		return "postgres"
	case "sqlite", "sqlite3":
		return "sqlite3"
	case "mysql", "mariadb":
		return "mysql"
	default:
		return driver
	}
}

func RebindSQL(driver, query string) string {
	if NormalizeDriver(driver) != "postgres" {
		return query
	}
	var out strings.Builder
	out.Grow(len(query) + 8)
	arg := 1
	var quote rune
	escaped := false
	runes := []rune(query)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote != 0 {
			out.WriteRune(r)
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				if i+1 < len(runes) && runes[i+1] == quote {
					i++
					out.WriteRune(runes[i])
					continue
				}
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"', '`':
			quote = r
			out.WriteRune(r)
		case '?':
			out.WriteByte('$')
			out.WriteString(strconv.Itoa(arg))
			arg++
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
