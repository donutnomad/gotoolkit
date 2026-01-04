package errors

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errBase     = errors.New("base error")
	errMarker   = errors.New("marker error")
	errMarker2  = errors.New("marker2 error")
	errNotFound = errors.New("not found")
)

func TestNew(t *testing.T) {
	err := New("test error")
	require.NotNil(t, err)
	assert.Equal(t, "test error", err.Error())
}

func TestIs(t *testing.T) {
	wrapped := Wrap(errBase, "context")
	assert.True(t, Is(wrapped, errBase))
	assert.False(t, Is(wrapped, errNotFound))
}

func TestFrom_NilSafe(t *testing.T) {
	// From(nil) 应该是安全的，后续操作不会 panic
	result := From(nil).Wrap("msg").Mark(errMarker).Err()
	assert.Nil(t, result)
}

func TestWrap(t *testing.T) {
	t.Run("wrap with message", func(t *testing.T) {
		err := Wrap(errBase, "context message")
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "context message")
		assert.Contains(t, err.Error(), "base error")
		assert.True(t, Is(err, errBase))
	})

	t.Run("wrap with multiple messages", func(t *testing.T) {
		err := Wrap(errBase, "part1", "part2")
		assert.Contains(t, err.Error(), "part1 part2")
	})

	t.Run("wrap nil returns nil", func(t *testing.T) {
		err := Wrap(nil, "message")
		assert.Nil(t, err)
	})
}

func TestWrapf(t *testing.T) {
	err := Wrapf(errBase, "failed with code %d", 500)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed with code 500")
	assert.True(t, Is(err, errBase))
}

func TestMarkPrefixWrap(t *testing.T) {
	t.Run("with marker and message", func(t *testing.T) {
		err := MarkPrefixWrap(errBase, errMarker, "additional context")
		require.NotNil(t, err)
		// 验证可以通过 Is 匹配 marker
		assert.True(t, Is(err, errMarker))
		// 验证原始错误仍可匹配
		assert.True(t, Is(err, errBase))
		// 验证 marker 消息被嵌入
		assert.Contains(t, err.Error(), errMarker.Error())
		assert.Contains(t, err.Error(), "additional context")
	})

	t.Run("with nil marker", func(t *testing.T) {
		err := MarkPrefixWrap(errBase, nil, "context")
		require.NotNil(t, err)
		assert.True(t, Is(err, errBase))
		assert.Contains(t, err.Error(), "context")
	})

	t.Run("with nil base error", func(t *testing.T) {
		err := MarkPrefixWrap(nil, errMarker, "context")
		assert.Nil(t, err)
	})
}

func TestMarkPrefixWrapf(t *testing.T) {
	err := MarkPrefixWrapf(errBase, errMarker, "user %s not found", "alice")
	require.NotNil(t, err)
	assert.True(t, Is(err, errMarker))
	assert.True(t, Is(err, errBase))
	assert.Contains(t, err.Error(), "user alice not found")
}

func TestMarkWrap(t *testing.T) {
	t.Run("marks error without embedding marker text", func(t *testing.T) {
		err := MarkWrap(errBase, errMarker, "context")
		require.NotNil(t, err)
		assert.True(t, Is(err, errMarker))
		assert.True(t, Is(err, errBase))
		assert.Contains(t, err.Error(), "context")
	})

	t.Run("with nil marker", func(t *testing.T) {
		err := MarkWrap(errBase, nil, "context")
		require.NotNil(t, err)
		assert.True(t, Is(err, errBase))
	})
}

func TestMarkWrapf(t *testing.T) {
	err := MarkWrapf(errBase, errMarker, "operation %s failed", "save")
	require.NotNil(t, err)
	assert.True(t, Is(err, errMarker))
	assert.Contains(t, err.Error(), "operation save failed")
}

func TestUnwrapOnce(t *testing.T) {
	wrapped := Wrap(errBase, "layer1")
	cause := UnwrapOnce(wrapped)
	require.NotNil(t, cause)
	assert.True(t, Is(cause, errBase))
}

func TestUnwrapAll(t *testing.T) {
	// 多层包装
	err := Wrap(Wrap(errBase, "layer1"), "layer2")
	root := UnwrapAll(err)
	assert.Equal(t, errBase, root)
}

func TestUnwrapMulti(t *testing.T) {
	// 单错误没有多错误
	err := Wrap(errBase, "msg")
	multi := UnwrapMulti(err)
	assert.Nil(t, multi)
}

// Builder 方法测试

func TestBuilder_Wrap(t *testing.T) {
	err := From(errBase).Wrap("context").Err()
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "context")
	assert.True(t, Is(err, errBase))
}

func TestBuilder_Wrapf(t *testing.T) {
	err := From(errBase).Wrapf("code: %d", 404).Err()
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "code: 404")
}

func TestBuilder_WithMessage(t *testing.T) {
	err := From(errBase).WithMessage("extra info").Err()
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "extra info")
}

func TestBuilder_WithMessagef(t *testing.T) {
	err := From(errBase).WithMessagef("user: %s", "bob").Err()
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "user: bob")
}

func TestBuilder_WithStack(t *testing.T) {
	err := From(errBase).WithStack().Err()
	require.NotNil(t, err)
	assert.True(t, Is(err, errBase))
}

func TestBuilder_Mark(t *testing.T) {
	t.Run("single marker", func(t *testing.T) {
		err := From(errBase).Mark(errMarker).Err()
		require.NotNil(t, err)
		assert.True(t, Is(err, errMarker))
		assert.True(t, Is(err, errBase))
	})

	t.Run("multiple markers", func(t *testing.T) {
		err := From(errBase).Mark(errMarker, errMarker2).Err()
		require.NotNil(t, err)
		assert.True(t, Is(err, errMarker))
		assert.True(t, Is(err, errMarker2))
		assert.True(t, Is(err, errBase))
	})

	t.Run("nil builder", func(t *testing.T) {
		var b *Builder
		result := b.Mark(errMarker)
		assert.Nil(t, result.Err())
	})
}

func TestBuilder_WithSecondaryError(t *testing.T) {
	secondary := errors.New("cleanup failed")
	err := From(errBase).WithSecondaryError(secondary).Err()
	require.NotNil(t, err)
	assert.True(t, Is(err, errBase))
	// cockroachdb/errors 的 secondary error 可以通过 GetAllDetails 获取
}

func TestBuilder_WithHint(t *testing.T) {
	err := From(errBase).WithHint("try again later").Err()
	require.NotNil(t, err)
	hints := GetAllHints(err)
	assert.Contains(t, hints, "try again later")
}

func TestBuilder_WithHintf(t *testing.T) {
	err := From(errBase).WithHintf("contact %s for help", "admin").Err()
	require.NotNil(t, err)
	hints := GetAllHints(err)
	assert.Contains(t, hints, "contact admin for help")
}

func TestBuilder_WithDetail(t *testing.T) {
	err := From(errBase).WithDetail("debug: request_id=123").Err()
	require.NotNil(t, err)
	details := GetAllDetails(err)
	assert.Contains(t, details, "debug: request_id=123")
}

func TestBuilder_WithDetailf(t *testing.T) {
	err := From(errBase).WithDetailf("request_id=%d", 123).Err()
	require.NotNil(t, err)
}

func TestBuilder_Chaining(t *testing.T) {
	// 测试链式调用
	err := From(errBase).
		Mark(errMarker).
		Wrap("context").
		WithHint("hint").
		WithDetail("detail").
		Err()

	require.NotNil(t, err)
	assert.True(t, Is(err, errBase))
	assert.True(t, Is(err, errMarker))
	assert.Contains(t, err.Error(), "context")

	// 验证 Hint 提取
	hints := GetAllHints(err)
	assert.Contains(t, hints, "hint")
	assert.Contains(t, FlattenHints(err), "hint")

	// 验证 Detail 提取
	details := GetAllDetails(err)
	assert.Contains(t, details, "detail")
	assert.Contains(t, FlattenDetails(err), "detail")
}

func TestBuilder_NilError(t *testing.T) {
	// 所有方法在 nil error 时都应该安全
	result := From(nil).
		Wrap("msg").
		Wrapf("fmt %d", 1).
		WithMessage("msg").
		WithMessagef("fmt %s", "s").
		WithStack().
		Mark(errMarker).
		WithSecondaryError(io.EOF).
		WithHint("hint").
		WithHintf("fmt %s", "s").
		WithDetail("detail").
		WithDetailf("fmt %d", 1).
		Err()

	assert.Nil(t, result)
}

func TestBuilder_isNil(t *testing.T) {
	t.Run("nil builder", func(t *testing.T) {
		var b *Builder
		assert.True(t, b.isNil())
	})

	t.Run("builder with nil error", func(t *testing.T) {
		b := From(nil)
		assert.True(t, b.isNil())
	})

	t.Run("builder with error", func(t *testing.T) {
		b := From(errBase)
		assert.False(t, b.isNil())
	})
}
