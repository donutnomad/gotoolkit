package gsql

import (
	"fmt"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

var Star field.IField = field.NewBase("", "*")

func FALSE() field.Expression {
	return clause.Expr{
		SQL: "FALSE",
	}
}

func NULL() field.Expression {
	return clause.Expr{
		SQL: "NULL",
	}
}

func CURRENT_TIMESTAMP() field.Expression {
	return clause.Expr{
		SQL: "CURRENT_TIMESTAMP",
	}
}

// ==================== 日期和时间函数 ====================

// NOW 返回当前日期和时间 (YYYY-MM-DD HH:MM:SS)
func NOW() field.Expression {
	return clause.Expr{SQL: "NOW()"}
}

// CURRENT_DATE 返回当前日期 (YYYY-MM-DD)
func CURRENT_DATE() field.Expression {
	return clause.Expr{SQL: "CURRENT_DATE()"}
}

// CURDATE 返回当前日期 (YYYY-MM-DD)
func CURDATE() field.Expression {
	return clause.Expr{SQL: "CURDATE()"}
}

// CURRENT_TIME 返回当前时间 (HH:MM:SS)
func CURRENT_TIME() field.Expression {
	return clause.Expr{SQL: "CURRENT_TIME()"}
}

// CURTIME 返回当前时间 (HH:MM:SS)
func CURTIME() field.Expression {
	return clause.Expr{SQL: "CURTIME()"}
}

// UTC_TIMESTAMP 返回当前的 UTC 日期和时间
func UTC_TIMESTAMP() field.Expression {
	return clause.Expr{SQL: "UTC_TIMESTAMP()"}
}

// UNIX_TIMESTAMP 返回当前 Unix 时间戳
func UNIX_TIMESTAMP(date ...field.Expression) field.Expression {
	if len(date) == 0 {
		return clause.Expr{SQL: "UNIX_TIMESTAMP()"}
	}
	return clause.Expr{
		SQL:  "UNIX_TIMESTAMP(?)",
		Vars: []any{date[0]},
	}
}

// DATE_FORMAT 格式化日期/时间为指定字符串
func DATE_FORMAT(date field.Expression, format string) field.Expression {
	return clause.Expr{
		SQL:  "DATE_FORMAT(?, ?)",
		Vars: []any{date, format},
	}
}

// STR_TO_DATE 将字符串按照指定格式转换为日期/时间
func STR_TO_DATE(str string, format string) field.Expression {
	return clause.Expr{
		SQL:  "STR_TO_DATE(?, ?)",
		Vars: []any{str, format},
	}
}

// YEAR 提取年份
func YEAR(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "YEAR(?)",
		Vars: []any{date},
	}
}

// MONTH 提取月份 (1-12)
func MONTH(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "MONTH(?)",
		Vars: []any{date},
	}
}

// DAY 提取日期的一个月中的天数 (1-31)
func DAY(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "DAY(?)",
		Vars: []any{date},
	}
}

// DAYOFMONTH 提取日期的一个月中的天数 (1-31)
func DAYOFMONTH(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "DAYOFMONTH(?)",
		Vars: []any{date},
	}
}

// WEEK 提取日期的周数
func WEEK(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "WEEK(?)",
		Vars: []any{date},
	}
}

// WEEKOFYEAR 提取日期的周数
func WEEKOFYEAR(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "WEEKOFYEAR(?)",
		Vars: []any{date},
	}
}

// HOUR 提取小时
func HOUR(time field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "HOUR(?)",
		Vars: []any{time},
	}
}

// MINUTE 提取分钟
func MINUTE(time field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "MINUTE(?)",
		Vars: []any{time},
	}
}

// SECOND 提取秒数
func SECOND(time field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "SECOND(?)",
		Vars: []any{time},
	}
}

// DAYOFWEEK 返回日期在周中的位置 (1=周日, 2=周一, ...)
func DAYOFWEEK(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "DAYOFWEEK(?)",
		Vars: []any{date},
	}
}

// DAYOFYEAR 返回日期在一年中的位置 (1-366)
func DAYOFYEAR(date field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "DAYOFYEAR(?)",
		Vars: []any{date},
	}
}

// DATE_ADD 在日期上增加一个时间间隔
// 例如: DATE_ADD(NOW(), "1 DAY"), DATE_ADD(NOW(), "2 HOUR")
func DATE_ADD(date field.Expression, interval string) field.Expression {
	return clause.Expr{
		SQL:  fmt.Sprintf("DATE_ADD(?, INTERVAL %s)", interval),
		Vars: []any{date},
	}
}

// DATE_SUB 从日期中减去一个时间间隔
// 例如: DATE_SUB(NOW(), "1 DAY"), DATE_SUB(NOW(), "2 HOUR")
func DATE_SUB(date field.Expression, interval string) field.Expression {
	return clause.Expr{
		SQL:  fmt.Sprintf("DATE_SUB(?, INTERVAL %s)", interval),
		Vars: []any{date},
	}
}

// DATEDIFF 返回两个日期之间的天数
func DATEDIFF(expr1, expr2 field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "DATEDIFF(?, ?)",
		Vars: []any{expr1, expr2},
	}
}

// TIMEDIFF 返回两个时间之间的差值
func TIMEDIFF(expr1, expr2 field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "TIMEDIFF(?, ?)",
		Vars: []any{expr1, expr2},
	}
}

// TIMESTAMPDIFF 返回两个日期时间表达式之间的差值,以指定单位表示
// unit: MICROSECOND, SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER, YEAR
func TIMESTAMPDIFF(unit string, expr1, expr2 field.Expression) field.Expression {
	return clause.Expr{
		SQL:  fmt.Sprintf("TIMESTAMPDIFF(%s, ?, ?)", unit),
		Vars: []any{expr1, expr2},
	}
}

// ==================== 字符串函数 ====================

// CONCAT 拼接多个字符串
func CONCAT(args ...any) field.Expression {
	placeholders := ""
	for i := range args {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
	}
	return clause.Expr{
		SQL:  fmt.Sprintf("CONCAT(%s)", placeholders),
		Vars: args,
	}
}

// CONCAT_WS 用指定分隔符拼接多个字符串 (跳过NULL值)
func CONCAT_WS(separator string, args ...any) field.Expression {
	placeholders := "?"
	allArgs := []any{separator}
	for range args {
		placeholders += ", ?"
	}
	allArgs = append(allArgs, args...)
	return clause.Expr{
		SQL:  fmt.Sprintf("CONCAT_WS(%s)", placeholders),
		Vars: allArgs,
	}
}

// LENGTH 返回字符串的字节长度
func LENGTH(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "LENGTH(?)",
		Vars: []any{str},
	}
}

// CHAR_LENGTH 返回字符串的字符长度 (多字节字符按一个字符计算)
func CHAR_LENGTH(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "CHAR_LENGTH(?)",
		Vars: []any{str},
	}
}

// CHARACTER_LENGTH 返回字符串的字符长度 (多字节字符按一个字符计算)
func CHARACTER_LENGTH(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "CHARACTER_LENGTH(?)",
		Vars: []any{str},
	}
}

// UPPER 转换为大写
func UPPER(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "UPPER(?)",
		Vars: []any{str},
	}
}

// UCASE 转换为大写
func UCASE(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "UCASE(?)",
		Vars: []any{str},
	}
}

// LOWER 转换为小写
func LOWER(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "LOWER(?)",
		Vars: []any{str},
	}
}

// LCASE 转换为小写
func LCASE(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "LCASE(?)",
		Vars: []any{str},
	}
}

// SUBSTRING 从字符串中提取子字符串
func SUBSTRING(str field.Expression, pos, length int) field.Expression {
	return clause.Expr{
		SQL:  "SUBSTRING(?, ?, ?)",
		Vars: []any{str, pos, length},
	}
}

// SUBSTR 从字符串中提取子字符串
func SUBSTR(str field.Expression, pos, length int) field.Expression {
	return clause.Expr{
		SQL:  "SUBSTR(?, ?, ?)",
		Vars: []any{str, pos, length},
	}
}

// LEFT 从字符串左侧提取指定长度
func LEFT(str field.Expression, length int) field.Expression {
	return clause.Expr{
		SQL:  "LEFT(?, ?)",
		Vars: []any{str, length},
	}
}

// RIGHT 从字符串右侧提取指定长度
func RIGHT(str field.Expression, length int) field.Expression {
	return clause.Expr{
		SQL:  "RIGHT(?, ?)",
		Vars: []any{str, length},
	}
}

// LOCATE 返回子字符串在字符串中第一次出现的位置
func LOCATE(substr, str field.Expression, pos ...int) field.Expression {
	if len(pos) > 0 {
		return clause.Expr{
			SQL:  "LOCATE(?, ?, ?)",
			Vars: []any{substr, str, pos[0]},
		}
	}
	return clause.Expr{
		SQL:  "LOCATE(?, ?)",
		Vars: []any{substr, str},
	}
}

// INSTR 返回子字符串在字符串中第一次出现的位置
func INSTR(str, substr field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "INSTR(?, ?)",
		Vars: []any{str, substr},
	}
}

// REPLACE 替换字符串中所有出现的子字符串
func REPLACE(str, fromStr, toStr field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "REPLACE(?, ?, ?)",
		Vars: []any{str, fromStr, toStr},
	}
}

// TRIM 去除字符串两端的空格
func TRIM(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "TRIM(?)",
		Vars: []any{str},
	}
}

// LTRIM 去除字符串左侧的空格
func LTRIM(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "LTRIM(?)",
		Vars: []any{str},
	}
}

// RTRIM 去除字符串右侧的空格
func RTRIM(str field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "RTRIM(?)",
		Vars: []any{str},
	}
}

// ==================== 数值函数 ====================

// ABS 绝对值
func ABS(x field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "ABS(?)",
		Vars: []any{x},
	}
}

// CEIL 向上取整
func CEIL(x field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "CEIL(?)",
		Vars: []any{x},
	}
}

// CEILING 向上取整
func CEILING(x field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "CEILING(?)",
		Vars: []any{x},
	}
}

// FLOOR 向下取整
func FLOOR(x field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "FLOOR(?)",
		Vars: []any{x},
	}
}

// ROUND 四舍五入到 D 位小数 (D 默认为 0)
func ROUND(x field.Expression, d ...int) field.Expression {
	if len(d) > 0 {
		return clause.Expr{
			SQL:  "ROUND(?, ?)",
			Vars: []any{x, d[0]},
		}
	}
	return clause.Expr{
		SQL:  "ROUND(?)",
		Vars: []any{x},
	}
}

// MOD 取模 (余数)
func MOD(n, m field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "MOD(?, ?)",
		Vars: []any{n, m},
	}
}

// POWER X 的 Y 次方
func POWER(x, y field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "POWER(?, ?)",
		Vars: []any{x, y},
	}
}

// POW X 的 Y 次方
func POW(x, y field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "POW(?, ?)",
		Vars: []any{x, y},
	}
}

// SQRT 平方根
func SQRT(x field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "SQRT(?)",
		Vars: []any{x},
	}
}

// RAND 返回 0 到 1 之间的随机浮点数
func RAND() field.Expression {
	return clause.Expr{SQL: "RAND()"}
}

// SIGN 返回 X 的符号 (-1, 0, 1)
func SIGN(x field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "SIGN(?)",
		Vars: []any{x},
	}
}

// TRUNCATE 截断到 D 位小数
func TRUNCATE(x field.Expression, d int) field.Expression {
	return clause.Expr{
		SQL:  "TRUNCATE(?, ?)",
		Vars: []any{x, d},
	}
}

// ==================== 聚合函数 ====================

// COUNT 计算非 NULL 值的数量
func COUNT(expr ...field.IField) field.Expression {
	if len(expr) == 0 {
		return clause.Expr{SQL: "COUNT(*)"}
	}
	return clause.Expr{
		SQL:  "COUNT(?)",
		Vars: []any{expr[0].ToExpr()},
	}
}

func COUNT_DISTINCT(expr field.IField) field.Expression {
	return clause.Expr{
		SQL:  "COUNT(DISTINCT ?)",
		Vars: []any{expr.ToExpr()},
	}
}

// SUM 计算数值列的总和
func SUM(expr field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "SUM(?)",
		Vars: []any{expr},
	}
}

// AVG 计算数值列的平均值
func AVG(expr field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "AVG(?)",
		Vars: []any{expr},
	}
}

// MAX 获取列的最大值
func MAX(expr field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "MAX(?)",
		Vars: []any{expr},
	}
}

// MIN 获取列的最小值
func MIN(expr field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "MIN(?)",
		Vars: []any{expr},
	}
}

// GROUP_CONCAT 将组内的字符串连接起来
func GROUP_CONCAT(expr field.Expression, separator ...string) field.Expression {
	if len(separator) > 0 {
		return clause.Expr{
			SQL:  fmt.Sprintf("GROUP_CONCAT(? SEPARATOR '%s')", separator[0]),
			Vars: []any{expr},
		}
	}
	return clause.Expr{
		SQL:  "GROUP_CONCAT(?)",
		Vars: []any{expr},
	}
}

// ==================== 流程控制函数 ====================

// IF 如果条件为真,返回第一个值,否则返回第二个值
func IF(condition field.Expression, valueIfTrue, valueIfFalse any) field.Expression {
	return clause.Expr{
		SQL:  "IF(?, ?, ?)",
		Vars: []any{condition, valueIfTrue, valueIfFalse},
	}
}

// IFNULL 如果 expr1 不为 NULL,返回 expr1,否则返回 expr2
func IFNULL(expr1, expr2 any) field.Expression {
	return clause.Expr{
		SQL:  "IFNULL(?, ?)",
		Vars: []any{expr1, expr2},
	}
}

// NULLIF 如果 expr1 = expr2,返回 NULL,否则返回 expr1
func NULLIF(expr1, expr2 any) field.Expression {
	return clause.Expr{
		SQL:  "NULLIF(?, ?)",
		Vars: []any{expr1, expr2},
	}
}

// ==================== 其它常用函数 ====================

// DATABASE 返回当前数据库名
func DATABASE() field.Expression {
	return clause.Expr{SQL: "DATABASE()"}
}

// USER 返回当前用户
func USER() field.Expression {
	return clause.Expr{SQL: "USER()"}
}

// CURRENT_USER 返回当前用户
func CURRENT_USER() field.Expression {
	return clause.Expr{SQL: "CURRENT_USER()"}
}

// VERSION 返回 MySQL 服务器版本
func VERSION() field.Expression {
	return clause.Expr{SQL: "VERSION()"}
}

// UUID 生成一个通用的唯一标识符 (UUID)
func UUID() field.Expression {
	return clause.Expr{SQL: "UUID()"}
}

// INET_ATON 将点分十进制的 IPv4 地址转换为数值表示
func INET_ATON(expr string) field.Expression {
	return clause.Expr{
		SQL:  "INET_ATON(?)",
		Vars: []any{expr},
	}
}

// INET_NTOA 将数值表示的 IPv4 地址转换为点分十进制
func INET_NTOA(expr field.Expression) field.Expression {
	return clause.Expr{
		SQL:  "INET_NTOA(?)",
		Vars: []any{expr},
	}
}
