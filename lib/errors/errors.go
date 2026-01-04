package errors

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errbase"
)

func Wrap(err error, msg ...string) error {
	return From(err).Wrap(strings.Join(msg, " ")).Err()
}

func Wrapf(err error, format string, args ...any) error {
	return From(err).Wrapf(format, args...).Err()
}

// New creates an error with a simple error message.
// A stack trace is retained.
//
// Note: the message string is assumed to not contain
// PII and is included in Sentry reports.
// Use errors.Newf("%s", <unsafestring>) for errors
// strings that may contain PII information.
//
// Detail output:
// - message via `Error()` and formatting using `%v`/`%s`/`%q`.
// - everything when formatting with `%+v`.
// - stack trace and message via `errors.GetSafeDetails()`.
// - stack trace and message in Sentry reports.
func New(msg string) error {
	return errors.New(msg)
}

// Is determines whether one of the causes of the given error or any
// of its causes is equivalent to some reference error.
//
// As in the Go standard library, an error is considered to match a
// reference error if it is equal to that target or if it implements a
// method Is(error) bool such that Is(reference) returns true.
//
// Note: the inverse is not true - making an Is(reference) method
// return false does not imply that errors.Is() also returns
// false. Errors can be equal because their network equality marker is
// the same. To force errors to appear different to Is(), use
// errors.Mark().
//
// Note: if any of the error types has been migrated from a previous
// package location or a different type, ensure that
// RegisterTypeMigration() was called prior to Is().
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// UnwrapOnce accesses the direct cause of the error if any, otherwise
// returns nil.
//
// It supports both errors implementing causer (`Cause()` method, from
// github.com/pkg/errors) and `Wrapper` (`Unwrap()` method, from the
// Go 2 error proposal).
//
// UnwrapOnce treats multi-errors (those implementing the
// `Unwrap() []error` interface as leaf-nodes since they cannot
// reasonably be iterated through to a single cause. These errors
// are typically constructed as a result of `fmt.Errorf` which results
// in a `wrapErrors` instance that contains an interpolated error
// string along with a list of causes.
//
// The go stdlib does not define output on `Unwrap()` for a multi-cause
// error, so we default to nil here.
func UnwrapOnce(err error) (cause error) {
	return errbase.UnwrapOnce(err)
}

// UnwrapAll accesses the root cause object of the error.
// If the error has no cause (leaf error), it is returned directly.
// UnwrapAll treats multi-errors as leaf nodes.
func UnwrapAll(err error) error {
	return errbase.UnwrapAll(err)
}

// UnwrapMulti access the slice of causes that an error contains, if it is a
// multi-error.
func UnwrapMulti(err error) []error {
	return errbase.UnwrapMulti(err)
}

// GetAllHints retrieves the hints from the error using in post-order
// traversal. The hints are de-duplicated. Assertion failures, issue
// links and unimplemented errors are detected and receive standard
// hints.
func GetAllHints(err error) []string { return errors.GetAllHints(err) }

// FlattenHints retrieves the hints as per GetAllHints() and
// concatenates them into a single string.
func FlattenHints(err error) string { return errors.FlattenHints(err) }

// GetAllDetails retrieves the details from the error using in post-order
// traversal.
func GetAllDetails(err error) []string { return errors.GetAllDetails(err) }

// FlattenDetails retrieves the details as per GetAllDetails() and
// concatenates them into a single string.
func FlattenDetails(err error) string { return errors.FlattenDetails(err) }
