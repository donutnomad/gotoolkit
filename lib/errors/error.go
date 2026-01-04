package errors

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
)

type Builder struct {
	err error
}

// From 构建错误链。传入 nil 是安全的
func From(err error) *Builder {
	return &Builder{err: err}
}

// 快捷组合函数

// MarkPrefixWrap 是 Mark + Wrap 的组合，marker.Error() 作为消息前缀
func MarkPrefixWrap(err error, marker error, msg ...string) error {
	format := strings.Join(msg, " ")
	if !lo.IsNil(marker) {
		format = marker.Error() + ": " + format
	}
	return MarkWrap(err, marker, format)
}

func MarkPrefixWrapf(err error, marker error, format string, args ...any) error {
	if !lo.IsNil(marker) {
		format = marker.Error() + ": " + format
	}
	return MarkWrapf(err, marker, format, args...)
}

// MarkWrap 是 Mark + Wrap 的组合调用
func MarkWrap(err error, marker error, msg ...string) error {
	return From(err).Mark(marker).Wrap(strings.Join(msg, " ")).Err()
}

// MarkWrapf 是 MarkWrap 的格式化版本
func MarkWrapf(err error, marker error, format string, args ...any) error {
	return From(err).Mark(marker).Wrapf(format, args...).Err()
}

// Wrap wraps an error with a message prefix.
// A stack trace is retained.
//
// Note: the prefix string is assumed to not contain
// PII and is included in Sentry reports.
// Use errors.Wrapf(err, "%s", <unsafestring>) for errors
// strings that may contain PII information.
//
// Detail output:
// - original error message + prefix via `Error()` and formatting using `%v`/`%s`/`%q`.
// - everything when formatting with `%+v`.
// - stack trace and message via `errors.GetSafeDetails()`.
// - stack trace and message in Sentry reports.
func (b *Builder) Wrap(msg string) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.Wrap(b.err, msg)
	return b
}

// Wrapf wraps an error with a formatted message prefix. A stack
// trace is also retained. If the format is empty, no prefix is added,
// but the extra arguments are still processed for reportable strings.
//
// Note: the format string is assumed to not contain
// PII and is included in Sentry reports.
// Use errors.Wrapf(err, "%s", <unsafestring>) for errors
// strings that may contain PII information.
//
// Detail output:
// - original error message + prefix via `Error()` and formatting using `%v`/`%s`/`%q`.
// - everything when formatting with `%+v`.
// - stack trace, format, and redacted details via `errors.GetSafeDetails()`.
// - stack trace, format, and redacted details in Sentry reports.
func (b *Builder) Wrapf(format string, args ...any) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.Wrapf(b.err, format, args...)
	return b
}

// WithMessage 只加文本
// WithMessage annotates err with a new message.
// If err is nil, WithMessage returns nil.
// The message is considered safe for reporting
// and is included in Sentry reports.
func (b *Builder) WithMessage(msg string) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithMessage(b.err, msg)
	return b
}

// WithMessagef annotates err with the format specifier.
// If err is nil, WithMessagef returns nil.
// The message is formatted as per redact.Sprintf,
// to separate safe and unsafe strings for Sentry reporting.
func (b *Builder) WithMessagef(format string, args ...any) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithMessagef(b.err, format, args...)
	return b
}

// WithStack 仅附加堆栈信息
// WithStack annotates err with a stack trace at the point WithStack was called.
//
// Detail is shown:
// - via `errors.GetSafeDetails()`
// - when formatting with `%+v`.
// - in Sentry reports.
// - when innermost stack capture, with `errors.GetOneLineSource()`.
func (b *Builder) WithStack() *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithStack(b.err)
	return b
}

// 标记与元数据方法

// Mark 对应 errors.Mark：给错误打标签（用于 errors.Is 判断）
func (b *Builder) Mark(marker ...error) *Builder {
	if b.isNil() {
		return b
	}
	for _, mk := range marker {
		if lo.IsNil(mk) {
			continue
		}
		b.err = errors.Mark(b.err, mk)
	}
	return b
}

// WithSecondaryError 为错误附加一个次级错误，用于在处理错误时，又发生错误的场景
func (b *Builder) WithSecondaryError(additionalErr error) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithSecondaryError(b.err, additionalErr)
	return b
}

// WithHint 对应 errors.WithHint：添加给用户看的提示信息
func (b *Builder) WithHint(msg string) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithHint(b.err, msg)
	return b
}

// WithHintf 带格式化的 Hint
func (b *Builder) WithHintf(format string, args ...any) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithHintf(b.err, format, args...)
	return b
}

// WithDetail 对应 errors.WithDetail：添加给开发人员看的调试详情
func (b *Builder) WithDetail(msg string) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithDetail(b.err, msg)
	return b
}

// WithDetailf 带格式化的 Detail
func (b *Builder) WithDetailf(format string, args ...any) *Builder {
	if b.isNil() {
		return b
	}
	b.err = errors.WithDetailf(b.err, format, args...)
	return b
}

// 终结方法

// Err 返回最终的 error 对象。
func (b *Builder) Err() error {
	if b.isNil() {
		return nil
	}
	return b.err
}

func (b *Builder) isNil() bool {
	return b == nil || lo.IsNil(b.err)
}
