package gsql

import (
	"fmt"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

var Star field.IField = field.NewBase("", "*")

func Primitive[T primitive](value T) field.ExpressionTo {
	return ExprTo{Expr("?", value)}
}

func StarWith(tableName string) field.IField {
	return field.NewBaseFromSql(Expr("?.*", quoteClause{
		name: tableName,
	}), "")
}

type quoteClause struct {
	name string
}

func (q quoteClause) Build(builder clause.Builder) {
	builder.WriteQuoted(q.name)
}

// FALSE 返回布尔值假
// SELECT FALSE;
// SELECT FALSE = 0;
// SELECT users.* FROM users WHERE users.is_active = FALSE;
func FALSE() field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL: "FALSE",
	}}
}

// NULL 返回空值
// SELECT NULL;
// SELECT IFNULL(users.nickname, NULL) FROM users;
// UPDATE users SET deleted_at = NULL WHERE id = 1;
func NULL() field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL: "NULL",
	}}
}

// CURRENT_TIMESTAMP 返回当前日期和时间 (YYYY-MM-DD HH:MM:SS)，是NOW()的同义词
// SELECT CURRENT_TIMESTAMP;
// SELECT CURRENT_TIMESTAMP();
// INSERT INTO logs (created_at) VALUES (CURRENT_TIMESTAMP);
// UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = 1;
func CURRENT_TIMESTAMP() field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL: "CURRENT_TIMESTAMP",
	}}
}

// ==================== 日期和时间函数 ====================

// NOW 返回当前日期和时间 (YYYY-MM-DD HH:MM:SS)
// SELECT NOW();
// SELECT NOW() + 0;
// INSERT INTO logs (created_at) VALUES (NOW());
// SELECT * FROM orders WHERE order_time > NOW() - INTERVAL 1 DAY;
func NOW() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "NOW()"}}
}

// CURRENT_DATE 返回当前日期 (YYYY-MM-DD)，不包含时间部分
// SELECT CURRENT_DATE;
// SELECT CURRENT_DATE();
// SELECT * FROM users WHERE DATE(created_at) = CURRENT_DATE;
// SELECT DATEDIFF(CURRENT_DATE, users.birthday) FROM users;
func CURRENT_DATE() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "CURRENT_DATE()"}}
}

// CURDATE 返回当前日期 (YYYY-MM-DD)，是CURRENT_DATE的同义词
// SELECT CURDATE();
// SELECT CURDATE() + 0;
// SELECT * FROM events WHERE event_date >= CURDATE();
// SELECT YEAR(CURDATE()), MONTH(CURDATE()), DAY(CURDATE());
func CURDATE() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "CURDATE()"}}
}

// CURRENT_TIME 返回当前时间 (HH:MM:SS)，不包含日期部分
// SELECT CURRENT_TIME;
// SELECT CURRENT_TIME();
// SELECT CURRENT_TIME + 0;
// SELECT * FROM schedules WHERE start_time <= CURRENT_TIME AND end_time >= CURRENT_TIME;
func CURRENT_TIME() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "CURRENT_TIME()"}}
}

// CURTIME 返回当前时间 (HH:MM:SS)，是CURRENT_TIME的同义词
// SELECT CURTIME();
// SELECT CURTIME() + 0;
// SELECT HOUR(CURTIME()), MINUTE(CURTIME()), SECOND(CURTIME());
// SELECT * FROM shifts WHERE shift_start <= CURTIME() AND shift_end >= CURTIME();
func CURTIME() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "CURTIME()"}}
}

// UTC_TIMESTAMP 返回当前的UTC日期和时间 (YYYY-MM-DD HH:MM:SS)
// SELECT UTC_TIMESTAMP;
// SELECT UTC_TIMESTAMP();
// SELECT UTC_TIMESTAMP(), NOW();
// INSERT INTO global_logs (created_at_utc) VALUES (UTC_TIMESTAMP());
func UTC_TIMESTAMP() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "UTC_TIMESTAMP()"}}
}

// UNIX_TIMESTAMP 返回Unix时间戳（秒），如果不提供参数则返回当前时间戳，提供参数则转换指定时间为时间戳
// SELECT UNIX_TIMESTAMP();
// SELECT UNIX_TIMESTAMP('2023-10-26 10:30:00');
// SELECT UNIX_TIMESTAMP(NOW());
// SELECT UNIX_TIMESTAMP(users.created_at) FROM users;
// SELECT * FROM orders WHERE UNIX_TIMESTAMP(order_time) > 1698306600;
func UNIX_TIMESTAMP(date ...field.Expression) field.ExpressionTo {
	if len(date) == 0 {
		return ExprTo{clause.Expr{SQL: "UNIX_TIMESTAMP()"}}
	}
	return ExprTo{clause.Expr{
		SQL:  "UNIX_TIMESTAMP(?)",
		Vars: []any{date[0]},
	}}
}

// FROM_UNIXTIME 将Unix时间戳（秒）转换为DATETIME类型，如果提供了format，将转换为VARCHAR类型
// SELECT FROM_UNIXTIME(1698306600, '%Y年%m月%d日 %H时%i分%s秒');
// SELECT FROM_UNIXTIME(1698306600);
// SELECT FROM_UNIXTIME(users.time);
// SELECT FROM_UNIXTIME(users.time + 3600);
func FROM_UNIXTIME(date field.Expression, format ...string) field.ExpressionTo {
	if len(format) > 0 {
		return ExprTo{clause.Expr{
			SQL:  "FROM_UNIXTIME(?, ?)",
			Vars: []any{date, format[0]},
		}}
	}
	return ExprTo{clause.Expr{
		SQL:  "FROM_UNIXTIME(?)",
		Vars: []any{date},
	}}
}

// DATE_FORMAT 格式化日期/时间为指定字符串，支持各种格式化符号
// SELECT DATE_FORMAT(NOW(), '%Y-%m-%d %H:%i:%s');
// SELECT DATE_FORMAT('2023-10-26', '%Y年%m月%d日');
// SELECT DATE_FORMAT(users.birthday, '%W %M %Y') FROM users;
// SELECT DATE_FORMAT(NOW(), '%Y%m%d%H%i%s');
func DATE_FORMAT(date field.Expression, format string) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "DATE_FORMAT(?, ?)",
		Vars: []any{date, format},
	}}
}

// STR_TO_DATE 将字符串按照指定格式转换为日期/时间，格式需要与字符串匹配
// SELECT STR_TO_DATE('2023-10-26', '%Y-%m-%d');
// SELECT STR_TO_DATE('2023年10月26日', '%Y年%m月%d日');
// SELECT STR_TO_DATE('10/26/2023 10:30:45', '%m/%d/%Y %H:%i:%s');
// SELECT * FROM orders WHERE order_date = STR_TO_DATE('20231026', '%Y%m%d');
func STR_TO_DATE(str string, format string) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "STR_TO_DATE(?, ?)",
		Vars: []any{str, format},
	}}
}

// YEAR 提取日期中的年份部分 (1000-9999)
// SELECT YEAR(NOW());
// SELECT YEAR('2023-10-26');
// SELECT * FROM users WHERE YEAR(birthday) = 1990;
// SELECT YEAR(users.created_at) as year FROM users GROUP BY year;
func YEAR(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "YEAR(?)",
		Vars: []any{date},
	}}
}

// MONTH 提取日期中的月份部分 (1-12)
// SELECT MONTH(NOW());
// SELECT MONTH('2023-10-26');
// SELECT * FROM orders WHERE MONTH(order_date) = 10;
// SELECT MONTH(users.birthday), COUNT(*) FROM users GROUP BY MONTH(users.birthday);
func MONTH(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "MONTH(?)",
		Vars: []any{date},
	}}
}

// DAY 提取日期中一个月中的天数 (1-31)，是DAYOFMONTH的同义词
// SELECT DAY(NOW());
// SELECT DAY('2023-10-26');
// SELECT * FROM events WHERE DAY(event_date) = 15;
// SELECT YEAR(date), MONTH(date), DAY(date) FROM logs;
func DAY(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "DAY(?)",
		Vars: []any{date},
	}}
}

// DAYOFMONTH 提取日期中一个月中的天数 (1-31)，是DAY的同义词
// SELECT DAYOFMONTH(NOW());
// SELECT DAYOFMONTH('2023-10-26');
// SELECT * FROM users WHERE DAYOFMONTH(birthday) = 1;
// SELECT DAYOFMONTH(created_at) FROM orders;
func DAYOFMONTH(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "DAYOFMONTH(?)",
		Vars: []any{date},
	}}
}

// WEEK 提取日期在一年中的周数 (0-53)，可选第二个参数指定周开始于周日还是周一
// SELECT WEEK(NOW());
// SELECT WEEK('2023-10-26');
// SELECT WEEK(NOW(), 1);
// SELECT * FROM orders WHERE WEEK(order_date) = WEEK(NOW());
func WEEK(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "WEEK(?)",
		Vars: []any{date},
	}}
}

// WEEKOFYEAR 提取日期在一年中的周数 (1-53)，相当于WEEK(date, 3)
// SELECT WEEKOFYEAR(NOW());
// SELECT WEEKOFYEAR('2023-10-26');
// SELECT * FROM events WHERE WEEKOFYEAR(event_date) = 43;
// SELECT WEEKOFYEAR(created_at), COUNT(*) FROM orders GROUP BY WEEKOFYEAR(created_at);
func WEEKOFYEAR(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "WEEKOFYEAR(?)",
		Vars: []any{date},
	}}
}

// HOUR 提取时间中的小时部分 (0-23)
// SELECT HOUR(NOW());
// SELECT HOUR('2023-10-26 14:30:45');
// SELECT * FROM logs WHERE HOUR(log_time) BETWEEN 9 AND 17;
// SELECT HOUR(users.last_login) FROM users;
func HOUR(time field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "HOUR(?)",
		Vars: []any{time},
	}}
}

// MINUTE 提取时间中的分钟部分 (0-59)
// SELECT MINUTE(NOW());
// SELECT MINUTE('2023-10-26 14:30:45');
// SELECT * FROM schedules WHERE MINUTE(start_time) = 0;
// SELECT HOUR(time), MINUTE(time) FROM appointments;
func MINUTE(time field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "MINUTE(?)",
		Vars: []any{time},
	}}
}

// SECOND 提取时间中的秒数部分 (0-59)
// SELECT SECOND(NOW());
// SELECT SECOND('2023-10-26 14:30:45');
// SELECT * FROM events WHERE SECOND(event_time) = 0;
// SELECT HOUR(time), MINUTE(time), SECOND(time) FROM logs;
func SECOND(time field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "SECOND(?)",
		Vars: []any{time},
	}}
}

// DAYOFWEEK 返回日期在一周中的索引 (1=周日, 2=周一, ..., 7=周六)
// SELECT DAYOFWEEK(NOW());
// SELECT DAYOFWEEK('2023-10-26');
// SELECT * FROM events WHERE DAYOFWEEK(event_date) IN (1, 7);
// SELECT CASE DAYOFWEEK(date) WHEN 1 THEN '周日' WHEN 2 THEN '周一' END FROM logs;
func DAYOFWEEK(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "DAYOFWEEK(?)",
		Vars: []any{date},
	}}
}

// DAYOFYEAR 返回日期在一年中的天数 (1-366)
// SELECT DAYOFYEAR(NOW());
// SELECT DAYOFYEAR('2023-10-26');
// SELECT * FROM logs WHERE DAYOFYEAR(log_date) = 1;
// SELECT DAYOFYEAR(created_at), COUNT(*) FROM orders GROUP BY DAYOFYEAR(created_at);
func DAYOFYEAR(date field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "DAYOFYEAR(?)",
		Vars: []any{date},
	}}
}

// DATE_ADD 在日期上增加一个时间间隔，支持多种时间单位
// SELECT DATE_ADD(NOW(), INTERVAL 1 DAY);
// SELECT DATE_ADD('2023-10-26', INTERVAL 2 HOUR);
// SELECT DATE_ADD(users.created_at, INTERVAL 7 DAY) FROM users;
// SELECT * FROM orders WHERE DATE_ADD(order_date, INTERVAL 30 DAY) > NOW();
// 支持单位: MICROSECOND, SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER, YEAR
func DATE_ADD(date field.Expression, interval string) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  fmt.Sprintf("DATE_ADD(?, INTERVAL %s)", interval),
		Vars: []any{date},
	}}
}

// DATE_SUB 从日期中减去一个时间间隔，支持多种时间单位
// SELECT DATE_SUB(NOW(), INTERVAL 1 DAY);
// SELECT DATE_SUB('2023-10-26', INTERVAL 2 HOUR);
// SELECT DATE_SUB(users.expire_date, INTERVAL 1 MONTH) FROM users;
// SELECT * FROM logs WHERE log_date >= DATE_SUB(NOW(), INTERVAL 7 DAY);
// 支持单位: MICROSECOND, SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER, YEAR
func DATE_SUB(date field.Expression, interval string) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  fmt.Sprintf("DATE_SUB(?, INTERVAL %s)", interval),
		Vars: []any{date},
	}}
}

// DATEDIFF 返回两个日期之间相差的天数 (date1 - date2)，只计算日期部分，忽略时间
// SELECT DATEDIFF(NOW(), '2023-01-01');
// SELECT DATEDIFF('2023-10-26', '2023-10-20');
// SELECT users.name, DATEDIFF(NOW(), users.birthday) / 365 as age FROM users;
// SELECT * FROM orders WHERE DATEDIFF(NOW(), order_date) > 30;
func DATEDIFF(expr1, expr2 field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "DATEDIFF(?, ?)",
		Vars: []any{expr1, expr2},
	}}
}

// TIMEDIFF 返回两个时间/日期时间之间的差值，结果为时间格式 (HH:MM:SS)
// SELECT TIMEDIFF(NOW(), '2023-10-26 10:00:00');
// SELECT TIMEDIFF('14:30:00', '10:15:00');
// SELECT TIMEDIFF(end_time, start_time) FROM events;
// SELECT * FROM logs WHERE TIMEDIFF(NOW(), log_time) > '01:00:00';
func TIMEDIFF(expr1, expr2 field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "TIMEDIFF(?, ?)",
		Vars: []any{expr1, expr2},
	}}
}

// TIMESTAMPDIFF 返回两个日期时间表达式之间的差值，以指定单位表示 (expr2 - expr1)
// SELECT TIMESTAMPDIFF(SECOND, '2023-10-26 10:00:00', '2023-10-26 10:05:00');
// SELECT TIMESTAMPDIFF(HOUR, start_time, end_time) FROM events;
// SELECT TIMESTAMPDIFF(YEAR, users.birthday, NOW()) as age FROM users;
// SELECT * FROM orders WHERE TIMESTAMPDIFF(DAY, order_date, NOW()) > 30;
// 支持单位: MICROSECOND, SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER, YEAR
func TIMESTAMPDIFF(unit string, expr1, expr2 field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  fmt.Sprintf("TIMESTAMPDIFF(%s, ?, ?)", unit),
		Vars: []any{expr1, expr2},
	}}
}

// ==================== 字符串函数 ====================

// CONCAT 拼接多个字符串，任意参数为NULL则返回NULL
// SELECT CONCAT('Hello', ' ', 'World');
// SELECT CONCAT(users.first_name, ' ', users.last_name) as full_name FROM users;
// SELECT CONCAT('User:', users.id) FROM users;
// SELECT CONCAT(YEAR(NOW()), '-', MONTH(NOW()));
func CONCAT(args ...field.Expression) field.ExpressionTo {
	placeholders := ""
	for i := range args {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
	}
	return ExprTo{clause.Expr{
		SQL:  fmt.Sprintf("CONCAT(%s)", placeholders),
		Vars: lo.ToAnySlice(args),
	}}
}

// CONCAT_WS 用指定分隔符拼接多个字符串，自动跳过NULL值，分隔符为NULL则返回NULL
// SELECT CONCAT_WS(',', 'A', 'B', 'C');
// SELECT CONCAT_WS('-', users.last_name, users.first_name) FROM users;
// SELECT CONCAT_WS('/', YEAR(date), MONTH(date), DAY(date)) FROM logs;
// SELECT CONCAT_WS(', ', city, state, country) FROM addresses;
func CONCAT_WS(separator string, args ...any) field.ExpressionTo {
	placeholders := "?"
	allArgs := []any{separator}
	for range args {
		placeholders += ", ?"
	}
	allArgs = append(allArgs, args...)
	return ExprTo{clause.Expr{
		SQL:  fmt.Sprintf("CONCAT_WS(%s)", placeholders),
		Vars: allArgs,
	}}
}

// LENGTH 返回字符串的字节长度，UTF-8编码中一个中文字符通常占3个字节
// SELECT LENGTH('Hello');
// SELECT LENGTH('你好');
// SELECT users.name, LENGTH(users.name) FROM users;
// SELECT * FROM products WHERE LENGTH(product_code) = 8;
func LENGTH(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "LENGTH(?)",
		Vars: []any{str},
	}}
}

// CHAR_LENGTH 返回字符串的字符长度，多字节字符按一个字符计算，是CHARACTER_LENGTH的同义词
// SELECT CHAR_LENGTH('Hello');
// SELECT CHAR_LENGTH('你好');
// SELECT users.name, CHAR_LENGTH(users.name) FROM users;
// SELECT * FROM articles WHERE CHAR_LENGTH(content) > 1000;
func CHAR_LENGTH(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "CHAR_LENGTH(?)",
		Vars: []any{str},
	}}
}

// CHARACTER_LENGTH 返回字符串的字符长度，多字节字符按一个字符计算，是CHAR_LENGTH的同义词
// SELECT CHARACTER_LENGTH('Hello');
// SELECT CHARACTER_LENGTH('你好世界');
// SELECT CHARACTER_LENGTH(description) FROM products;
// SELECT * FROM posts WHERE CHARACTER_LENGTH(title) < 50;
func CHARACTER_LENGTH(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "CHARACTER_LENGTH(?)",
		Vars: []any{str},
	}}
}

// UPPER 将字符串转换为大写，只对英文字母有效
// SELECT UPPER('hello world');
// SELECT UPPER(users.username) FROM users;
// SELECT * FROM products WHERE UPPER(product_code) = 'ABC123';
// UPDATE users SET username = UPPER(username) WHERE id = 1;
func UPPER(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "UPPER(?)",
		Vars: []any{str},
	}}
}

// UCASE 将字符串转换为大写，是UPPER的同义词
// SELECT UCASE('hello world');
// SELECT UCASE(email) FROM users;
// SELECT * FROM codes WHERE UCASE(code) LIKE 'A%';
// SELECT CONCAT(UCASE(LEFT(name, 1)), SUBSTRING(name, 2)) FROM users;
func UCASE(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "UCASE(?)",
		Vars: []any{str},
	}}
}

// LOWER 将字符串转换为小写，只对英文字母有效
// SELECT LOWER('HELLO WORLD');
// SELECT LOWER(users.email) FROM users;
// SELECT * FROM users WHERE LOWER(username) = 'admin';
// UPDATE users SET email = LOWER(email);
func LOWER(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "LOWER(?)",
		Vars: []any{str},
	}}
}

// LCASE 将字符串转换为小写，是LOWER的同义词
// SELECT LCASE('HELLO WORLD');
// SELECT LCASE(company_name) FROM companies;
// SELECT * FROM domains WHERE LCASE(domain) = 'example.com';
// SELECT LCASE(TRIM(email)) FROM users;
func LCASE(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "LCASE(?)",
		Vars: []any{str},
	}}
}

// SUBSTRING 从字符串中提取子字符串，位置从1开始，是SUBSTR的同义词
// SELECT SUBSTRING('Hello World', 1, 5);
// SELECT SUBSTRING('Hello World', 7);
// SELECT SUBSTRING(users.email, 1, LOCATE('@', users.email) - 1) FROM users;
// SELECT SUBSTRING(product_code, 4, 3) FROM products;
func SUBSTRING(str field.Expression, pos, length int) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "SUBSTRING(?, ?, ?)",
		Vars: []any{str, pos, length},
	}}
}

// SUBSTR 从字符串中提取子字符串，位置从1开始，是SUBSTRING的同义词
// SELECT SUBSTR('Hello World', 1, 5);
// SELECT SUBSTR('Hello World', 7);
// SELECT SUBSTR(description, 1, 100) FROM articles;
// SELECT SUBSTR(phone, -4) FROM users;
func SUBSTR(str field.Expression, pos, length int) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "SUBSTR(?, ?, ?)",
		Vars: []any{str, pos, length},
	}}
}

// LEFT 从字符串左侧提取指定长度的子字符串
// SELECT LEFT('Hello World', 5);
// SELECT LEFT(users.name, 1) as initial FROM users;
// SELECT * FROM products WHERE LEFT(product_code, 2) = 'AB';
// SELECT LEFT(email, LOCATE('@', email) - 1) FROM users;
func LEFT(str field.Expression, length int) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "LEFT(?, ?)",
		Vars: []any{str, length},
	}}
}

// RIGHT 从字符串右侧提取指定长度的子字符串
// SELECT RIGHT('Hello World', 5);
// SELECT RIGHT(phone, 4) as last_four FROM users;
// SELECT * FROM files WHERE RIGHT(filename, 4) = '.pdf';
// SELECT RIGHT(product_code, 3) FROM products;
func RIGHT(str field.Expression, length int) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "RIGHT(?, ?)",
		Vars: []any{str, length},
	}}
}

// LOCATE 返回子字符串在字符串中第一次出现的位置（从1开始），未找到返回0，可选起始位置
// SELECT LOCATE('World', 'Hello World');
// SELECT LOCATE('o', 'Hello World');
// SELECT LOCATE('o', 'Hello World', 6);
// SELECT * FROM users WHERE LOCATE('@', email) > 0;
func LOCATE(substr, str field.Expression, pos ...int) field.ExpressionTo {
	if len(pos) > 0 {
		return ExprTo{clause.Expr{
			SQL:  "LOCATE(?, ?, ?)",
			Vars: []any{substr, str, pos[0]},
		}}
	}
	return ExprTo{clause.Expr{
		SQL:  "LOCATE(?, ?)",
		Vars: []any{substr, str},
	}}
}

// INSTR 返回子字符串在字符串中第一次出现的位置（从1开始），未找到返回0
// SELECT INSTR('Hello World', 'World');
// SELECT INSTR('Hello World', 'o');
// SELECT * FROM urls WHERE INSTR(url, 'https://') = 1;
// SELECT INSTR(email, '@') as at_position FROM users;
func INSTR(str, substr field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "INSTR(?, ?)",
		Vars: []any{str, substr},
	}}
}

// REPLACE 替换字符串中所有出现的子字符串
// SELECT REPLACE('Hello World', 'World', 'MySQL');
// SELECT REPLACE('www.example.com', 'www', 'mail');
// SELECT REPLACE(phone, '-', ”) FROM users;
// UPDATE products SET description = REPLACE(description, 'old', 'new');
func REPLACE(str, fromStr, toStr field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "REPLACE(?, ?, ?)",
		Vars: []any{str, fromStr, toStr},
	}}
}

// TRIM 去除字符串两端的空格，也可指定去除的字符
// SELECT TRIM('  Hello World  ');
// SELECT TRIM(BOTH 'x' FROM 'xxxHelloxxx');
// SELECT TRIM(users.username) FROM users;
// UPDATE users SET email = TRIM(email);
func TRIM(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "TRIM(?)",
		Vars: []any{str},
	}}
}

// LTRIM 去除字符串左侧的空格
// SELECT LTRIM('  Hello World  ');
// SELECT LTRIM(users.name) FROM users;
// SELECT * FROM products WHERE LTRIM(code) != code;
// UPDATE users SET username = LTRIM(username);
func LTRIM(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "LTRIM(?)",
		Vars: []any{str},
	}}
}

// RTRIM 去除字符串右侧的空格
// SELECT RTRIM('  Hello World  ');
// SELECT RTRIM(description) FROM products;
// SELECT * FROM users WHERE RTRIM(email) != email;
// UPDATE articles SET title = RTRIM(title);
func RTRIM(str field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "RTRIM(?)",
		Vars: []any{str},
	}}
}

// ==================== 数值函数 ====================

// ABS 返回数值的绝对值
// SELECT ABS(-10);
// SELECT ABS(10);
// SELECT ABS(users.balance) FROM users;
// SELECT * FROM transactions WHERE ABS(amount) > 1000;
func ABS(x field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "ABS(?)",
		Vars: []any{x},
	}}
}

// CEIL 向上取整，返回大于或等于X的最小整数，是CEILING的同义词
// SELECT CEIL(4.3);
// SELECT CEIL(4.9);
// SELECT CEIL(-4.3);
// SELECT CEIL(price * 1.1) FROM products;
func CEIL(x field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "CEIL(?)",
		Vars: []any{x},
	}}
}

// CEILING 向上取整，返回大于或等于X的最小整数，是CEIL的同义词
// SELECT CEILING(4.3);
// SELECT CEILING(4.9);
// SELECT CEILING(-4.3);
// SELECT CEILING(total / 10) * 10 FROM orders;
func CEILING(x field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "CEILING(?)",
		Vars: []any{x},
	}}
}

// FLOOR 向下取整，返回小于或等于X的最大整数
// SELECT FLOOR(4.3);
// SELECT FLOOR(4.9);
// SELECT FLOOR(-4.3);
// SELECT FLOOR(price * 0.9) FROM products;
func FLOOR(x field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "FLOOR(?)",
		Vars: []any{x},
	}}
}

// ROUND 四舍五入到指定小数位数，默认四舍五入到整数
// SELECT ROUND(4.567);
// SELECT ROUND(4.567, 2);
// SELECT ROUND(4.567, 0);
// SELECT ROUND(price, 2) FROM products;
// SELECT ROUND(123.456, -1);
func ROUND(x field.Expression, d ...int) field.ExpressionTo {
	if len(d) > 0 {
		return ExprTo{clause.Expr{
			SQL:  "ROUND(?, ?)",
			Vars: []any{x, d[0]},
		}}
	}
	return ExprTo{clause.Expr{
		SQL:  "ROUND(?)",
		Vars: []any{x},
	}}
}

// MOD 返回N除以M的余数（模运算）
// SELECT MOD(10, 3);
// SELECT MOD(234, 10);
// SELECT MOD(-10, 3);
// SELECT * FROM users WHERE MOD(id, 2) = 0;
func MOD(n, m field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "MOD(?, ?)",
		Vars: []any{n, m},
	}}
}

// POWER 返回X的Y次幂，是POW的同义词
// SELECT POWER(2, 3);
// SELECT POWER(10, 2);
// SELECT POWER(5, -1);
// SELECT POWER(users.level, 2) FROM users;
func POWER(x, y field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "POWER(?, ?)",
		Vars: []any{x, y},
	}}
}

// POW 返回X的Y次幂，是POWER的同义词
// SELECT POW(2, 3);
// SELECT POW(10, 2);
// SELECT POW(distance, 2) FROM locations;
// SELECT SQRT(POW(x2 - x1, 2) + POW(y2 - y1, 2)) as distance FROM points;
func POW(x, y field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "POW(?, ?)",
		Vars: []any{x, y},
	}}
}

// SQRT 返回X的平方根，X必须为非负数
// SELECT SQRT(4);
// SELECT SQRT(16);
// SELECT SQRT(2);
// SELECT SQRT(area) as side_length FROM squares;
func SQRT(x field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "SQRT(?)",
		Vars: []any{x},
	}}
}

// RAND 返回0到1之间的随机浮点数，可选种子参数
// SELECT RAND();
// SELECT RAND() * 100;
// SELECT RAND(123);
// SELECT * FROM users ORDER BY RAND() LIMIT 10;
func RAND() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "RAND()"}}
}

// SIGN 返回数值的符号：负数返回-1，零返回0，正数返回1
// SELECT SIGN(-10);
// SELECT SIGN(0);
// SELECT SIGN(10);
// SELECT SIGN(balance) FROM accounts;
func SIGN(x field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "SIGN(?)",
		Vars: []any{x},
	}}
}

// TRUNCATE 截断数值到指定小数位数，不进行四舍五入
// SELECT TRUNCATE(4.567, 2);
// SELECT TRUNCATE(4.567, 0);
// SELECT TRUNCATE(123.456, -1);
// SELECT TRUNCATE(price, 2) FROM products;
func TRUNCATE(x field.Expression, d int) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "TRUNCATE(?, ?)",
		Vars: []any{x, d},
	}}
}

// ==================== 聚合函数 ====================

// COUNT 计算行数或非NULL值的数量，不提供参数时统计所有行（包括NULL）
// SELECT COUNT(*) FROM users;
// SELECT COUNT(email) FROM users;
// SELECT COUNT(users.id) FROM users;
// SELECT status, COUNT(*) FROM orders GROUP BY status;
func COUNT(expr ...field.IField) field.ExpressionTo {
	if len(expr) == 0 {
		return ExprTo{clause.Expr{SQL: "COUNT(*)"}}
	}
	return ExprTo{clause.Expr{
		SQL:  "COUNT(?)",
		Vars: []any{expr[0].ToExpr()},
	}}
}

// COUNT_DISTINCT 计算不重复的非NULL值的数量
// SELECT COUNT(DISTINCT city) FROM users;
// SELECT COUNT(DISTINCT country) FROM addresses;
// SELECT user_id, COUNT(DISTINCT product_id) FROM orders GROUP BY user_id;
// SELECT COUNT(DISTINCT email) FROM subscribers;
func COUNT_DISTINCT(expr field.IField) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "COUNT(DISTINCT ?)",
		Vars: []any{expr.ToExpr()},
	}}
}

// SUM 计算数值列的总和，忽略NULL值
// SELECT SUM(amount) FROM orders;
// SELECT SUM(price * quantity) FROM order_items;
// SELECT user_id, SUM(points) FROM transactions GROUP BY user_id;
// SELECT SUM(IF(status = 'completed', amount, 0)) FROM orders;
func SUM(expr field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "SUM(?)",
		Vars: []any{expr},
	}}
}

// AVG 计算数值列的平均值，忽略NULL值
// SELECT AVG(price) FROM products;
// SELECT AVG(age) FROM users;
// SELECT category, AVG(price) FROM products GROUP BY category;
// SELECT AVG(TIMESTAMPDIFF(YEAR, birthday, NOW())) FROM users;
func AVG(expr field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "AVG(?)",
		Vars: []any{expr},
	}}
}

// MAX 返回列的最大值，可用于数值、字符串、日期等类型
// SELECT MAX(price) FROM products;
// SELECT MAX(created_at) FROM orders;
// SELECT MAX(LENGTH(description)) FROM articles;
// SELECT category, MAX(price) FROM products GROUP BY category;
func MAX(expr field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "MAX(?)",
		Vars: []any{expr},
	}}
}

// MIN 返回列的最小值，可用于数值、字符串、日期等类型
// SELECT MIN(price) FROM products;
// SELECT MIN(created_at) FROM users;
// SELECT MIN(stock) FROM inventory;
// SELECT category, MIN(price) FROM products GROUP BY category;
func MIN(expr field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "MIN(?)",
		Vars: []any{expr},
	}}
}

// GROUP_CONCAT 将分组内的字符串连接起来，默认用逗号分隔，可指定分隔符
// SELECT GROUP_CONCAT(name) FROM users;
// SELECT GROUP_CONCAT(name SEPARATOR ';') FROM users;
// SELECT user_id, GROUP_CONCAT(product_name) FROM orders GROUP BY user_id;
// SELECT category, GROUP_CONCAT(DISTINCT tag ORDER BY tag) FROM products GROUP BY category;
func GROUP_CONCAT(expr field.Expression, separator ...string) field.ExpressionTo {
	if len(separator) > 0 {
		return ExprTo{clause.Expr{
			SQL:  fmt.Sprintf("GROUP_CONCAT(? SEPARATOR '%s')", separator[0]),
			Vars: []any{expr},
		}}
	}
	return ExprTo{clause.Expr{
		SQL:  "GROUP_CONCAT(?)",
		Vars: []any{expr},
	}}
}

// ==================== 流程控制函数 ====================

// IF 条件判断函数，如果条件为真返回第一个值，否则返回第二个值
// SELECT IF(score >= 60, '及格', '不及格') FROM students;
// SELECT IF(stock > 0, 'In Stock', 'Out of Stock') FROM products;
// SELECT name, IF(age >= 18, '成年', '未成年') FROM users;
// SELECT SUM(IF(status = 'completed', amount, 0)) FROM orders;
func IF(condition, valueIfTrue, valueIfFalse field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "IF(?, ?, ?)",
		Vars: []any{condition, valueIfTrue, valueIfFalse},
	}}
}

// IFNULL 如果第一个表达式不为NULL则返回它，否则返回第二个表达式
// SELECT IFNULL(nickname, username) FROM users;
// SELECT IFNULL(discount, 0) FROM products;
// SELECT IFNULL(email, 'no-email') FROM contacts;
// SELECT name, IFNULL(phone, 'N/A') FROM users;
func IFNULL(expr1, expr2 any) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "IFNULL(?, ?)",
		Vars: []any{expr1, expr2},
	}}
}

// NULLIF 如果两个表达式相等则返回NULL，否则返回第一个表达式
// SELECT NULLIF(10, 10);
// SELECT NULLIF(10, 5);
// SELECT NULLIF(username, ”) FROM users;
// SELECT 100 / NULLIF(quantity, 0) FROM inventory;
func NULLIF(expr1, expr2 any) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "NULLIF(?, ?)",
		Vars: []any{expr1, expr2},
	}}
}

// ==================== 其它常用函数 ====================

// DATABASE 返回当前使用的数据库名，如果未选择数据库则返回NULL
// SELECT DATABASE();
// INSERT INTO logs (db_name) VALUES (DATABASE());
// SELECT DATABASE() as current_db;
// SELECT * FROM information_schema.tables WHERE table_schema = DATABASE();
func DATABASE() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "DATABASE()"}}
}

// USER 返回当前MySQL用户名和主机名，格式为 'user@host'
// SELECT USER();
// INSERT INTO audit_logs (user) VALUES (USER());
// SELECT USER() as current_user;
// SELECT * FROM connections WHERE user = USER();
func USER() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "USER()"}}
}

// CURRENT_USER 返回当前MySQL用户名和主机名，与USER()相同
// SELECT CURRENT_USER();
// SELECT CURRENT_USER;
// INSERT INTO access_logs (accessed_by) VALUES (CURRENT_USER());
// SELECT CURRENT_USER() as authenticated_user;
func CURRENT_USER() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "CURRENT_USER()"}}
}

// VERSION 返回MySQL服务器的版本号
// SELECT VERSION();
// SELECT VERSION() as mysql_version;
// INSERT INTO system_info (version) VALUES (VERSION());
// SELECT IF(VERSION() LIKE '8.%', 'MySQL 8', 'Older') as version_check;
func VERSION() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "VERSION()"}}
}

// UUID 生成一个符合RFC 4122标准的通用唯一标识符（36字符的字符串）
// SELECT UUID();
// INSERT INTO records (id) VALUES (UUID());
// SELECT UUID() as unique_id;
// UPDATE sessions SET session_id = UUID() WHERE session_id IS NULL;
func UUID() field.ExpressionTo {
	return ExprTo{clause.Expr{SQL: "UUID()"}}
}

// INET_ATON 将点分十进制的IPv4地址转换为整数形式（网络字节序）
// SELECT INET_ATON('192.168.1.1');
// SELECT INET_ATON('10.0.0.1');
// INSERT INTO ip_logs (ip_num) VALUES (INET_ATON('192.168.1.100'));
// SELECT * FROM ip_ranges WHERE INET_ATON('192.168.1.50') BETWEEN start_ip AND end_ip;
func INET_ATON(expr string) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "INET_ATON(?)",
		Vars: []any{expr},
	}}
}

// INET_NTOA 将整数形式的IP地址转换为点分十进制字符串
// SELECT INET_NTOA(3232235777);
// SELECT INET_NTOA(167772161);
// SELECT INET_NTOA(ip_address) FROM access_logs;
// SELECT user_id, INET_NTOA(last_ip) FROM users;
func INET_NTOA(expr field.Expression) field.ExpressionTo {
	return ExprTo{clause.Expr{
		SQL:  "INET_NTOA(?)",
		Vars: []any{expr},
	}}
}

// ==================== JSON 函数 ====================

// JSON_OBJECT 创建 JSON 对象，接受成对的键值参数（key1, value1, key2, value2, ...）
// SELECT JSON_OBJECT('name', 'John', 'age', 30);
// SELECT JSON_OBJECT('id', users.id, 'name', users.name) FROM users;
// SELECT JSON_OBJECT('total', COUNT(*), 'sum', SUM(amount)) FROM orders;
// SELECT JSON_OBJECT('user', users.name, 'email', users.email, 'status', users.status) FROM users;
func JSON_OBJECT(pairs ...lo.Entry[string, field.Expression]) *jsonObjectBuilder {
	return &jsonObjectBuilder{
		pairs: lo.FromEntries(pairs),
	}
}

type jsonObjectBuilder struct {
	pairs map[string]field.Expression
}

func (j *jsonObjectBuilder) Add(key string, value field.Expression) *jsonObjectBuilder {
	j.pairs[key] = value
	return j
}

func (j *jsonObjectBuilder) toExpr() field.ExpressionTo {
	placeholders := ""
	for i := range len(j.pairs) * 2 {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
	}
	var unpack = make([]any, 0, len(j.pairs)*2)
	for k, v := range j.pairs {
		unpack = append(unpack, k, v)
	}
	return ExprTo{clause.Expr{
		SQL:  fmt.Sprintf("JSON_OBJECT(%s)", placeholders),
		Vars: unpack,
	}}
}

func (j *jsonObjectBuilder) Build(builder clause.Builder) {
	j.toExpr().Build(builder)
}

func (j *jsonObjectBuilder) AsF(name ...string) field.IField {
	return j.toExpr().AsF(name...)
}
