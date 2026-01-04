package errdefer

import (
	"errors"
	"io"
	"testing"

	cerrors "github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errBase      = errors.New("base error")
	errMarker    = errors.New("marker")
	errRetryable = errors.New("retryable")
	errNotFound  = errors.New("not found")
)

func TestMark(t *testing.T) {
	t.Run("marks error with marker", func(t *testing.T) {
		err := errBase
		Mark(&err, errMarker)
		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errMarker))
		assert.True(t, cerrors.Is(err, errBase))
	})

	t.Run("nil error pointer", func(t *testing.T) {
		// 不应该 panic
		Mark(nil, errMarker)
	})

	t.Run("nil error value", func(t *testing.T) {
		var err error
		Mark(&err, errMarker)
		assert.Nil(t, err)
	})

	t.Run("defer usage pattern", func(t *testing.T) {
		fn := func() (err error) {
			defer Mark(&err, errRetryable)
			return errBase
		}

		err := fn()
		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errRetryable))
		assert.True(t, cerrors.Is(err, errBase))
	})

	t.Run("defer with no error", func(t *testing.T) {
		fn := func() (err error) {
			defer Mark(&err, errRetryable)
			return nil
		}

		err := fn()
		assert.Nil(t, err)
	})
}

func TestMarkIf(t *testing.T) {
	t.Run("marks when predicate returns true", func(t *testing.T) {
		err := errBase
		MarkIf(&err, func(e error) bool {
			return e == errBase
		}, errMarker)

		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errMarker))
	})

	t.Run("does not mark when predicate returns false", func(t *testing.T) {
		err := errBase
		MarkIf(&err, func(e error) bool {
			return false
		}, errMarker)

		require.NotNil(t, err)
		assert.False(t, cerrors.Is(err, errMarker))
		assert.True(t, cerrors.Is(err, errBase))
	})

	t.Run("with message", func(t *testing.T) {
		err := errBase
		MarkIf(&err, func(e error) bool {
			return true
		}, errMarker, "additional", "context")

		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errMarker))
		assert.Contains(t, err.Error(), "additional context")
	})

	t.Run("without message", func(t *testing.T) {
		err := errBase
		MarkIf(&err, func(e error) bool {
			return true
		}, errMarker)

		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errMarker))
		assert.True(t, cerrors.Is(err, errBase))
	})

	t.Run("nil error pointer", func(t *testing.T) {
		MarkIf(nil, func(e error) bool {
			return true
		}, errMarker)
	})

	t.Run("nil error value", func(t *testing.T) {
		var err error
		MarkIf(&err, func(e error) bool {
			return true
		}, errMarker)
		assert.Nil(t, err)
	})

	t.Run("defer usage with condition", func(t *testing.T) {
		isTemporary := func(err error) bool {
			return errors.Is(err, io.ErrShortWrite)
		}

		fn := func(shouldFail bool) (err error) {
			defer MarkIf(&err, isTemporary, errRetryable)
			if shouldFail {
				return io.ErrShortWrite
			}
			return nil
		}

		// 条件匹配时打标
		err := fn(true)
		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errRetryable))

		// 无错误时不打标
		err = fn(false)
		assert.Nil(t, err)
	})
}

func TestMarkIfIs(t *testing.T) {
	t.Run("marks when error matches target", func(t *testing.T) {
		err := io.EOF
		MarkIfIs(&err, io.EOF, errNotFound)

		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errNotFound))
		assert.True(t, cerrors.Is(err, io.EOF))
	})

	t.Run("does not mark when error does not match", func(t *testing.T) {
		err := io.ErrUnexpectedEOF
		MarkIfIs(&err, io.EOF, errNotFound)

		require.NotNil(t, err)
		assert.False(t, cerrors.Is(err, errNotFound))
		assert.True(t, cerrors.Is(err, io.ErrUnexpectedEOF))
	})

	t.Run("with message", func(t *testing.T) {
		err := io.EOF
		MarkIfIs(&err, io.EOF, errNotFound, "config file missing")

		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errNotFound))
		assert.Contains(t, err.Error(), "config file missing")
	})

	t.Run("nil error pointer", func(t *testing.T) {
		MarkIfIs(nil, io.EOF, errNotFound)
	})

	t.Run("nil error value", func(t *testing.T) {
		var err error
		MarkIfIs(&err, io.EOF, errNotFound)
		assert.Nil(t, err)
	})

	t.Run("defer usage pattern", func(t *testing.T) {
		fn := func() (err error) {
			defer MarkIfIs(&err, io.EOF, errNotFound, "user data")
			return io.EOF
		}

		err := fn()
		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errNotFound))
		assert.True(t, cerrors.Is(err, io.EOF))
	})

	t.Run("wrapped error matching", func(t *testing.T) {
		// 包装后的错误也应该能匹配
		wrappedEOF := cerrors.Wrap(io.EOF, "read failed")
		err := wrappedEOF
		MarkIfIs(&err, io.EOF, errNotFound)

		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errNotFound))
		assert.True(t, cerrors.Is(err, io.EOF))
	})
}

func TestDeferPattern_RealWorld(t *testing.T) {
	// 模拟真实的 defer 使用场景

	t.Run("database operation with retryable marking", func(t *testing.T) {
		errDBConnection := errors.New("connection refused")

		isConnectionError := func(err error) bool {
			return errors.Is(err, errDBConnection)
		}

		queryDB := func(fail bool) (err error) {
			defer MarkIf(&err, isConnectionError, errRetryable, "db query")
			if fail {
				return errDBConnection
			}
			return nil
		}

		err := queryDB(true)
		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errRetryable))
		assert.Contains(t, err.Error(), "db query")
	})

	t.Run("file operation with not found marking", func(t *testing.T) {
		errFileNotExist := errors.New("file does not exist")
		errConfigMissing := errors.New("config missing")

		loadConfig := func(exists bool) (err error) {
			defer MarkIfIs(&err, errFileNotExist, errConfigMissing)
			if !exists {
				return errFileNotExist
			}
			return nil
		}

		err := loadConfig(false)
		require.NotNil(t, err)
		assert.True(t, cerrors.Is(err, errConfigMissing))
	})
}
