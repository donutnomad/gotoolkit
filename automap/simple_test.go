package automap

import (
	"testing"
)

func TestSimpleCall(t *testing.T) {
	// 直接调用mod.go中的函数
	result, err := mapAToBTest()
	if err != nil {
		t.Fatalf("从mod.go调用失败: %v", err)
	}

	// 验证结果
	if result.FuncSignature.FuncName != "MapAToB" {
		t.Errorf("期望函数名: MapAToB, 实际: %s", result.FuncSignature.FuncName)
	}

	if result.AType.Name != "A" {
		t.Errorf("期望A类型名: A, 实际: %s", result.AType.Name)
	}

	if result.BType.Name != "B" {
		t.Errorf("期望B类型名: B, 实际: %s", result.BType.Name)
	}

	if !result.HasExportPatch {
		t.Error("期望有ExportPatch方法")
	}

	t.Logf("成功解析函数: %s", result.FuncSignature.FuncName)
	t.Logf("A类型字段数: %d", len(result.AType.Fields))
	t.Logf("B类型字段数: %d", len(result.BType.Fields))
}
