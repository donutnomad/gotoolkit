package field

import (
	"fmt"
)

type pointerImpl struct {
	IField
}

func (f pointerImpl) NotNil() Expression {
	return f.operatePointerValue("IS NOT NULL")
}

func (f pointerImpl) IsNil() Expression {
	return f.operatePointerValue("IS NULL")
}

func (f pointerImpl) operatePointerValue(operator string) Expression {
	query, args := f.Column().Unpack()
	return Expression{Query: fmt.Sprintf("%s %s", query, operator), Args: args}
}
