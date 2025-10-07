package gsql

import (
	"fmt"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
)

var Star field.IField = field.NewBase("", "*")

func FALSE() field.Expression {
	return field.Expression{
		Query: "FALSE",
	}
}

func NULL() field.Expression {
	return field.Expression{
		Query: "NULL",
	}
}

func CURRENT_TIMESTAMP() field.Expression {
	return field.Expression{
		Query: "CURRENT_TIMESTAMP",
	}
}

// ==================== 日期和时间函数 ====================

// NOW 返回当前日期和时间 (YYYY-MM-DD HH:MM:SS)
func NOW() field.Expression {
	return field.Expression{Query: "NOW()"}
}

// CURRENT_DATE 返回当前日期 (YYYY-MM-DD)
func CURRENT_DATE() field.Expression {
	return field.Expression{Query: "CURRENT_DATE()"}
}

// CURDATE 返回当前日期 (YYYY-MM-DD)
func CURDATE() field.Expression {
	return field.Expression{Query: "CURDATE()"}
}

// CURRENT_TIME 返回当前时间 (HH:MM:SS)
func CURRENT_TIME() field.Expression {
	return field.Expression{Query: "CURRENT_TIME()"}
}

// CURTIME 返回当前时间 (HH:MM:SS)
func CURTIME() field.Expression {
	return field.Expression{Query: "CURTIME()"}
}

// UTC_TIMESTAMP 返回当前的 UTC 日期和时间
func UTC_TIMESTAMP() field.Expression {
	return field.Expression{Query: "UTC_TIMESTAMP()"}
}

// UNIX_TIMESTAMP 返回当前 Unix 时间戳
func UNIX_TIMESTAMP(date ...field.Expression) field.Expression {
	if len(date) == 0 {
		return field.Expression{Query: "UNIX_TIMESTAMP()"}
	}
	return field.Expression{
		Query: "UNIX_TIMESTAMP(?)",
		Args:  []any{date[0]},
	}
}

// DATE_FORMAT 格式化日期/时间为指定字符串
func DATE_FORMAT(date field.Expression, format string) field.Expression {
	return field.Expression{
		Query: "DATE_FORMAT(?, ?)",
		Args:  []any{date, format},
	}
}

// STR_TO_DATE 将字符串按照指定格式转换为日期/时间
func STR_TO_DATE(str string, format string) field.Expression {
	return field.Expression{
		Query: "STR_TO_DATE(?, ?)",
		Args:  []any{str, format},
	}
}

// YEAR 提取年份
func YEAR(date field.Expression) field.Expression {
	return field.Expression{
		Query: "YEAR(?)",
		Args:  []any{date},
	}
}

// MONTH 提取月份 (1-12)
func MONTH(date field.Expression) field.Expression {
	return field.Expression{
		Query: "MONTH(?)",
		Args:  []any{date},
	}
}

// DAY 提取日期的一个月中的天数 (1-31)
func DAY(date field.Expression) field.Expression {
	return field.Expression{
		Query: "DAY(?)",
		Args:  []any{date},
	}
}

// DAYOFMONTH 提取日期的一个月中的天数 (1-31)
func DAYOFMONTH(date field.Expression) field.Expression {
	return field.Expression{
		Query: "DAYOFMONTH(?)",
		Args:  []any{date},
	}
}

// WEEK 提取日期的周数
func WEEK(date field.Expression) field.Expression {
	return field.Expression{
		Query: "WEEK(?)",
		Args:  []any{date},
	}
}

// WEEKOFYEAR 提取日期的周数
func WEEKOFYEAR(date field.Expression) field.Expression {
	return field.Expression{
		Query: "WEEKOFYEAR(?)",
		Args:  []any{date},
	}
}

// HOUR 提取小时
func HOUR(time field.Expression) field.Expression {
	return field.Expression{
		Query: "HOUR(?)",
		Args:  []any{time},
	}
}

// MINUTE 提取分钟
func MINUTE(time field.Expression) field.Expression {
	return field.Expression{
		Query: "MINUTE(?)",
		Args:  []any{time},
	}
}

// SECOND 提取秒数
func SECOND(time field.Expression) field.Expression {
	return field.Expression{
		Query: "SECOND(?)",
		Args:  []any{time},
	}
}

// DAYOFWEEK 返回日期在周中的位置 (1=周日, 2=周一, ...)
func DAYOFWEEK(date field.Expression) field.Expression {
	return field.Expression{
		Query: "DAYOFWEEK(?)",
		Args:  []any{date},
	}
}

// DAYOFYEAR 返回日期在一年中的位置 (1-366)
func DAYOFYEAR(date field.Expression) field.Expression {
	return field.Expression{
		Query: "DAYOFYEAR(?)",
		Args:  []any{date},
	}
}

// DATE_ADD 在日期上增加一个时间间隔
// 例如: DATE_ADD(NOW(), "1 DAY"), DATE_ADD(NOW(), "2 HOUR")
func DATE_ADD(date field.Expression, interval string) field.Expression {
	return field.Expression{
		Query: fmt.Sprintf("DATE_ADD(?, INTERVAL %s)", interval),
		Args:  []any{date},
	}
}

// DATE_SUB 从日期中减去一个时间间隔
// 例如: DATE_SUB(NOW(), "1 DAY"), DATE_SUB(NOW(), "2 HOUR")
func DATE_SUB(date field.Expression, interval string) field.Expression {
	return field.Expression{
		Query: fmt.Sprintf("DATE_SUB(?, INTERVAL %s)", interval),
		Args:  []any{date},
	}
}

// DATEDIFF 返回两个日期之间的天数
func DATEDIFF(expr1, expr2 field.Expression) field.Expression {
	return field.Expression{
		Query: "DATEDIFF(?, ?)",
		Args:  []any{expr1, expr2},
	}
}

// TIMEDIFF 返回两个时间之间的差值
func TIMEDIFF(expr1, expr2 field.Expression) field.Expression {
	return field.Expression{
		Query: "TIMEDIFF(?, ?)",
		Args:  []any{expr1, expr2},
	}
}

// TIMESTAMPDIFF 返回两个日期时间表达式之间的差值,以指定单位表示
// unit: MICROSECOND, SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER, YEAR
func TIMESTAMPDIFF(unit string, expr1, expr2 field.Expression) field.Expression {
	return field.Expression{
		Query: fmt.Sprintf("TIMESTAMPDIFF(%s, ?, ?)", unit),
		Args:  []any{expr1, expr2},
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
	return field.Expression{
		Query: fmt.Sprintf("CONCAT(%s)", placeholders),
		Args:  args,
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
	return field.Expression{
		Query: fmt.Sprintf("CONCAT_WS(%s)", placeholders),
		Args:  allArgs,
	}
}

// LENGTH 返回字符串的字节长度
func LENGTH(str field.Expression) field.Expression {
	return field.Expression{
		Query: "LENGTH(?)",
		Args:  []any{str},
	}
}

// CHAR_LENGTH 返回字符串的字符长度 (多字节字符按一个字符计算)
func CHAR_LENGTH(str field.Expression) field.Expression {
	return field.Expression{
		Query: "CHAR_LENGTH(?)",
		Args:  []any{str},
	}
}

// CHARACTER_LENGTH 返回字符串的字符长度 (多字节字符按一个字符计算)
func CHARACTER_LENGTH(str field.Expression) field.Expression {
	return field.Expression{
		Query: "CHARACTER_LENGTH(?)",
		Args:  []any{str},
	}
}

// UPPER 转换为大写
func UPPER(str field.Expression) field.Expression {
	return field.Expression{
		Query: "UPPER(?)",
		Args:  []any{str},
	}
}

// UCASE 转换为大写
func UCASE(str field.Expression) field.Expression {
	return field.Expression{
		Query: "UCASE(?)",
		Args:  []any{str},
	}
}

// LOWER 转换为小写
func LOWER(str field.Expression) field.Expression {
	return field.Expression{
		Query: "LOWER(?)",
		Args:  []any{str},
	}
}

// LCASE 转换为小写
func LCASE(str field.Expression) field.Expression {
	return field.Expression{
		Query: "LCASE(?)",
		Args:  []any{str},
	}
}

// SUBSTRING 从字符串中提取子字符串
func SUBSTRING(str field.Expression, pos, length int) field.Expression {
	return field.Expression{
		Query: "SUBSTRING(?, ?, ?)",
		Args:  []any{str, pos, length},
	}
}

// SUBSTR 从字符串中提取子字符串
func SUBSTR(str field.Expression, pos, length int) field.Expression {
	return field.Expression{
		Query: "SUBSTR(?, ?, ?)",
		Args:  []any{str, pos, length},
	}
}

// LEFT 从字符串左侧提取指定长度
func LEFT(str field.Expression, length int) field.Expression {
	return field.Expression{
		Query: "LEFT(?, ?)",
		Args:  []any{str, length},
	}
}

// RIGHT 从字符串右侧提取指定长度
func RIGHT(str field.Expression, length int) field.Expression {
	return field.Expression{
		Query: "RIGHT(?, ?)",
		Args:  []any{str, length},
	}
}

// LOCATE 返回子字符串在字符串中第一次出现的位置
func LOCATE(substr, str field.Expression, pos ...int) field.Expression {
	if len(pos) > 0 {
		return field.Expression{
			Query: "LOCATE(?, ?, ?)",
			Args:  []any{substr, str, pos[0]},
		}
	}
	return field.Expression{
		Query: "LOCATE(?, ?)",
		Args:  []any{substr, str},
	}
}

// INSTR 返回子字符串在字符串中第一次出现的位置
func INSTR(str, substr field.Expression) field.Expression {
	return field.Expression{
		Query: "INSTR(?, ?)",
		Args:  []any{str, substr},
	}
}

// REPLACE 替换字符串中所有出现的子字符串
func REPLACE(str, fromStr, toStr field.Expression) field.Expression {
	return field.Expression{
		Query: "REPLACE(?, ?, ?)",
		Args:  []any{str, fromStr, toStr},
	}
}

// TRIM 去除字符串两端的空格
func TRIM(str field.Expression) field.Expression {
	return field.Expression{
		Query: "TRIM(?)",
		Args:  []any{str},
	}
}

// LTRIM 去除字符串左侧的空格
func LTRIM(str field.Expression) field.Expression {
	return field.Expression{
		Query: "LTRIM(?)",
		Args:  []any{str},
	}
}

// RTRIM 去除字符串右侧的空格
func RTRIM(str field.Expression) field.Expression {
	return field.Expression{
		Query: "RTRIM(?)",
		Args:  []any{str},
	}
}

// ==================== 数值函数 ====================

// ABS 绝对值
func ABS(x field.Expression) field.Expression {
	return field.Expression{
		Query: "ABS(?)",
		Args:  []any{x},
	}
}

// CEIL 向上取整
func CEIL(x field.Expression) field.Expression {
	return field.Expression{
		Query: "CEIL(?)",
		Args:  []any{x},
	}
}

// CEILING 向上取整
func CEILING(x field.Expression) field.Expression {
	return field.Expression{
		Query: "CEILING(?)",
		Args:  []any{x},
	}
}

// FLOOR 向下取整
func FLOOR(x field.Expression) field.Expression {
	return field.Expression{
		Query: "FLOOR(?)",
		Args:  []any{x},
	}
}

// ROUND 四舍五入到 D 位小数 (D 默认为 0)
func ROUND(x field.Expression, d ...int) field.Expression {
	if len(d) > 0 {
		return field.Expression{
			Query: "ROUND(?, ?)",
			Args:  []any{x, d[0]},
		}
	}
	return field.Expression{
		Query: "ROUND(?)",
		Args:  []any{x},
	}
}

// MOD 取模 (余数)
func MOD(n, m field.Expression) field.Expression {
	return field.Expression{
		Query: "MOD(?, ?)",
		Args:  []any{n, m},
	}
}

// POWER X 的 Y 次方
func POWER(x, y field.Expression) field.Expression {
	return field.Expression{
		Query: "POWER(?, ?)",
		Args:  []any{x, y},
	}
}

// POW X 的 Y 次方
func POW(x, y field.Expression) field.Expression {
	return field.Expression{
		Query: "POW(?, ?)",
		Args:  []any{x, y},
	}
}

// SQRT 平方根
func SQRT(x field.Expression) field.Expression {
	return field.Expression{
		Query: "SQRT(?)",
		Args:  []any{x},
	}
}

// RAND 返回 0 到 1 之间的随机浮点数
func RAND() field.Expression {
	return field.Expression{Query: "RAND()"}
}

// SIGN 返回 X 的符号 (-1, 0, 1)
func SIGN(x field.Expression) field.Expression {
	return field.Expression{
		Query: "SIGN(?)",
		Args:  []any{x},
	}
}

// TRUNCATE 截断到 D 位小数
func TRUNCATE(x field.Expression, d int) field.Expression {
	return field.Expression{
		Query: "TRUNCATE(?, ?)",
		Args:  []any{x, d},
	}
}

// ==================== 聚合函数 ====================

// COUNT 计算非 NULL 值的数量
func COUNT(expr ...field.IField) field.Expression {
	if len(expr) == 0 {
		return field.Expression{Query: "COUNT(*)"}
	}
	return field.Expression{
		Query: "COUNT(?)",
		Args:  []any{expr[0].Column()},
	}
}

func COUNT_DISTINCT(expr field.IField) field.Expression {
	return field.Expression{
		Query: "COUNT(DISTINCT ?)",
		Args:  []any{expr.Column()},
	}
}

// SUM 计算数值列的总和
func SUM(expr field.Expression) field.Expression {
	return field.Expression{
		Query: "SUM(?)",
		Args:  []any{expr},
	}
}

// AVG 计算数值列的平均值
func AVG(expr field.Expression) field.Expression {
	return field.Expression{
		Query: "AVG(?)",
		Args:  []any{expr},
	}
}

// MAX 获取列的最大值
func MAX(expr field.Expression) field.Expression {
	return field.Expression{
		Query: "MAX(?)",
		Args:  []any{expr},
	}
}

// MIN 获取列的最小值
func MIN(expr field.Expression) field.Expression {
	return field.Expression{
		Query: "MIN(?)",
		Args:  []any{expr},
	}
}

// GROUP_CONCAT 将组内的字符串连接起来
func GROUP_CONCAT(expr field.Expression, separator ...string) field.Expression {
	if len(separator) > 0 {
		return field.Expression{
			Query: fmt.Sprintf("GROUP_CONCAT(? SEPARATOR '%s')", separator[0]),
			Args:  []any{expr},
		}
	}
	return field.Expression{
		Query: "GROUP_CONCAT(?)",
		Args:  []any{expr},
	}
}

// ==================== 流程控制函数 ====================

// IF 如果条件为真,返回第一个值,否则返回第二个值
func IF(condition field.Expression, valueIfTrue, valueIfFalse any) field.Expression {
	return field.Expression{
		Query: "IF(?, ?, ?)",
		Args:  []any{condition, valueIfTrue, valueIfFalse},
	}
}

// IFNULL 如果 expr1 不为 NULL,返回 expr1,否则返回 expr2
func IFNULL(expr1, expr2 any) field.Expression {
	return field.Expression{
		Query: "IFNULL(?, ?)",
		Args:  []any{expr1, expr2},
	}
}

// NULLIF 如果 expr1 = expr2,返回 NULL,否则返回 expr1
func NULLIF(expr1, expr2 any) field.Expression {
	return field.Expression{
		Query: "NULLIF(?, ?)",
		Args:  []any{expr1, expr2},
	}
}

// ==================== 其它常用函数 ====================

// DATABASE 返回当前数据库名
func DATABASE() field.Expression {
	return field.Expression{Query: "DATABASE()"}
}

// USER 返回当前用户
func USER() field.Expression {
	return field.Expression{Query: "USER()"}
}

// CURRENT_USER 返回当前用户
func CURRENT_USER() field.Expression {
	return field.Expression{Query: "CURRENT_USER()"}
}

// VERSION 返回 MySQL 服务器版本
func VERSION() field.Expression {
	return field.Expression{Query: "VERSION()"}
}

// UUID 生成一个通用的唯一标识符 (UUID)
func UUID() field.Expression {
	return field.Expression{Query: "UUID()"}
}

// INET_ATON 将点分十进制的 IPv4 地址转换为数值表示
func INET_ATON(expr string) field.Expression {
	return field.Expression{
		Query: "INET_ATON(?)",
		Args:  []any{expr},
	}
}

// INET_NTOA 将数值表示的 IPv4 地址转换为点分十进制
func INET_NTOA(expr field.Expression) field.Expression {
	return field.Expression{
		Query: "INET_NTOA(?)",
		Args:  []any{expr},
	}
}
