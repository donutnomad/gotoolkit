package automap

import (
	"testing"
)

// TestFromMod 从mod.go文件调用的测试
func TestFromMod(t *testing.T) {
	// 这个函数会从test文件中调用，但我们需要从mod.go中的函数调用
	// 所以我们直接调用mapAToBTest函数
	result, err := mapAToBTest()
	if err != nil {
		t.Fatalf("测试失败: %v", err)
	}

	if result.AType.Name != "A" {
		t.Errorf("期望A类型名: A, 实际: %s", result.AType.Name)
	}

	if result.BType.Name != "B" {
		t.Errorf("期望B类型名: B, 实际: %s", result.BType.Name)
	}
}
