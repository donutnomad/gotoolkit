package utils

import "gorm.io/gorm/clause"

// IN Whether a value is within a set of values
type IN struct {
	Column any
	Values []any
}

func (in IN) Build(builder clause.Builder) {
	writeString := func(str string) {
		_, _ = builder.WriteString(str)
	}
	builder.WriteQuoted(in.Column)

	switch len(in.Values) {
	case 0:
		writeString(" IN (NULL)")
	case 1:
		if _, ok := in.Values[0].([]any); !ok {
			writeString(" = ")
			builder.AddVar(builder, in.Values[0])
			break
		}

		fallthrough
	default:
		writeString(" IN ")
		addVarAutoBracket(builder, in.Values)
	}
}

func (in IN) NegationBuild(builder clause.Builder) {
	writeString := func(str string) {
		_, _ = builder.WriteString(str)
	}

	builder.WriteQuoted(in.Column)
	switch len(in.Values) {
	case 0:
		writeString(" IS NOT NULL")
	case 1:
		if _, ok := in.Values[0].([]any); !ok {
			writeString(" <> ")
			builder.AddVar(builder, in.Values[0])
			break
		}

		fallthrough
	default:
		writeString(" NOT IN ")
		addVarAutoBracket(builder, in.Values)
	}
}

func addVarAutoBracket(builder clause.Builder, values []any) {
	needBracket := IsNeedBracket(values)
	if needBracket {
		_ = builder.WriteByte('(')
	}
	builder.AddVar(builder, values...)
	if needBracket {
		_ = builder.WriteByte(')')
	}
}
