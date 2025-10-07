package field

import (
	"fmt"
	"strings"
)

// mysqlReservedWords MySQL 保留字集合
var mysqlReservedWords = map[string]struct{}{"add": {}, "all": {}, "alter": {}, "analyze": {}, "and": {}, "as": {}, "asc": {}, "before": {}, "between": {}, "bigint": {}, "binary": {}, "blob": {}, "both": {}, "by": {}, "call": {}, "cascade": {}, "case": {}, "change": {}, "char": {}, "character": {}, "check": {}, "collate": {}, "column": {}, "condition": {}, "constraint": {}, "continue": {}, "convert": {}, "create": {}, "cross": {}, "current_date": {}, "current_time": {}, "current_timestamp": {}, "current_user": {}, "cursor": {}, "database": {}, "databases": {}, "day_hour": {}, "day_microsecond": {}, "day_minute": {}, "day_second": {}, "dec": {}, "decimal": {}, "declare": {}, "default": {}, "delayed": {}, "delete": {}, "desc": {}, "describe": {}, "deterministic": {}, "distinct": {}, "distinctrow": {}, "div": {}, "double": {}, "drop": {}, "dual": {}, "each": {}, "else": {}, "elseif": {}, "enclosed": {}, "escaped": {}, "exists": {}, "exit": {}, "explain": {}, "false": {}, "fetch": {}, "float": {}, "float4": {}, "float8": {}, "for": {}, "force": {}, "foreign": {}, "from": {}, "fulltext": {}, "grant": {}, "group": {}, "having": {}, "high_priority": {}, "hour_microsecond": {}, "hour_minute": {}, "hour_second": {}, "if": {}, "ignore": {}, "in": {}, "index": {}, "infile": {}, "inner": {}, "inout": {}, "insensitive": {}, "insert": {}, "int": {}, "int1": {}, "int2": {}, "int3": {}, "int4": {}, "int8": {}, "integer": {}, "interval": {}, "into": {}, "is": {}, "iterate": {}, "join": {}, "key": {}, "keys": {}, "kill": {}, "leading": {}, "leave": {}, "left": {}, "like": {}, "limit": {}, "linear": {}, "lines": {}, "load": {}, "localtime": {}, "localtimestamp": {}, "lock": {}, "long": {}, "longblob": {}, "longtext": {}, "loop": {}, "low_priority": {}, "match": {}, "mediumblob": {}, "mediumint": {}, "mediumtext": {}, "middleint": {}, "minute_microsecond": {}, "minute_second": {}, "mod": {}, "modifies": {}, "natural": {}, "not": {}, "no_write_to_binlog": {}, "null": {}, "numeric": {}, "on": {}, "optimize": {}, "option": {}, "optionally": {}, "or": {}, "order": {}, "out": {}, "outer": {}, "outfile": {}, "precision": {}, "primary": {}, "procedure": {}, "purge": {}, "range": {}, "read": {}, "reads": {}, "read_write": {}, "real": {}, "references": {}, "regexp": {}, "release": {}, "rename": {}, "repeat": {}, "replace": {}, "require": {}, "restrict": {}, "return": {}, "revoke": {}, "right": {}, "rlike": {}, "schema": {}, "schemas": {}, "second_microsecond": {}, "select": {}, "sensitive": {}, "separator": {}, "set": {}, "show": {}, "smallint": {}, "spatial": {}, "specific": {}, "sql": {}, "sqlexception": {}, "sqlstate": {}, "sqlwarning": {}, "sql_big_result": {}, "sql_calc_found_rows": {}, "sql_small_result": {}, "ssl": {}, "starting": {}, "straight_join": {}, "table": {}, "terminated": {}, "then": {}, "tinyblob": {}, "tinyint": {}, "tinytext": {}, "to": {}, "trailing": {}, "trigger": {}, "true": {}, "undo": {}, "union": {}, "unique": {}, "unlock": {}, "unsigned": {}, "update": {}, "usage": {}, "use": {}, "using": {}, "utc_date": {}, "utc_time": {}, "utc_timestamp": {}, "values": {}, "varbinary": {}, "varchar": {}, "varcharacter": {}, "varying": {}, "when": {}, "where": {}, "while": {}, "with": {}, "write": {}, "x509": {}, "xor": {}, "year_month": {}, "zerofill": {}}

// WrapIfReserved 如果是 MySQL 保留字则用反引号包裹
func WrapIfReserved(name string) string {
	if _, ok := mysqlReservedWords[strings.ToLower(name)]; ok {
		return fmt.Sprintf("`%s`", name)
	}
	return name
}

func AS(name, alias string) string {
	if alias != "" {
		return fmt.Sprintf("%s AS %s", WrapIfReserved(name), alias)
	}
	return WrapIfReserved(name)
}

func ExtractColumn(field IField) string {
	q := field.Column().Query
	var name string
	if strings.Contains(q, "AS") {
		name = strings.Split(q, " AS ")[1]
	} else if idx := strings.Index(q, "."); idx >= 0 {
		name = q
	} else {
		name = q
	}
	return name
}

func sliceToAny[T any](input []T) []any {
	var ret = make([]any, len(input))
	for i, item := range input {
		ret[i] = item
	}
	return ret
}

func requireNoArgs(query string, args []any) string {
	if len(args) > 0 {
		panic("has args")
	}
	return query
}
