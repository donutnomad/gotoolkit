package approvegen

import (
	"context"
	"errors"
	"fmt"
)

var ErrUnknownMethod = errors.New("ErrUnknownMethod")
var ErrNoFormatter = errors.New("NoFormatter")

type Caller interface {
	Call(ctx context.Context, arg any, approved bool) (any, error)
	Format(ctx context.Context, arg any, formatter any) (any, error)
	UnmarshalMethodArgs(method string, content string) (any, error)
}

type Callers []Caller

func (c Callers) Call(ctx context.Context, method, content string, approved bool) (any, error) {
	arg, caller, err := c.UnmarshalMethodArgs(method, content)
	if err != nil {
		return nil, err
	}
	return caller.Call(ctx, arg, approved)
}

func (c Callers) Format(ctx context.Context, method, content string, formatter any) (any, error) {
	arg, caller, err := c.UnmarshalMethodArgs(method, content)
	if err != nil {
		return nil, err
	}
	ret, err := caller.Format(ctx, arg, formatter)
	if err != nil {
		if err.Error() == "NoFormatter" {
			return nil, ErrNoFormatter
		}
		return nil, err
	}
	return ret, nil
}

func (c Callers) UnmarshalMethodArgs(method string, content string) (arg any, _ Caller, err error) {
	for _, group := range c {
		arg, err = group.UnmarshalMethodArgs(method, content)
		if err != nil {
			return nil, nil, err
		}
		if arg == nil {
			continue
		}
		return arg, group, nil
	}
	return nil, nil, errors.Join(ErrUnknownMethod, fmt.Errorf("unable to unmarshal method: %s", method))
}
