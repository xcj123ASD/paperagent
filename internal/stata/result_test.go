package stata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempCSV 在临时目录写一个 CSV，返回路径。
func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "result.csv")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写 CSV 失败: %v", err)
	}
	return path
}

func TestParseCSV_Normal(t *testing.T) {
	path := writeTempCSV(t, "variable,coef,se,pvalue\nDigital,0.052,0.015,0.001\n")
	r, err := ParseCSV(path)
	if err != nil {
		t.Fatalf("ParseCSV 失败: %v", err)
	}
	if r.Variable != "Digital" {
		t.Errorf("Variable 期望 Digital，实际 %q", r.Variable)
	}
	if r.Coef != 0.052 {
		t.Errorf("Coef 期望 0.052，实际 %v", r.Coef)
	}
	if r.SE != 0.015 {
		t.Errorf("SE 期望 0.015，实际 %v", r.SE)
	}
	if r.PValue != 0.001 {
		t.Errorf("PValue 期望 0.001，实际 %v", r.PValue)
	}
}

func TestParseCSV_FileNotFound(t *testing.T) {
	_, err := ParseCSV(`C:\no\such\result.csv`)
	if err == nil {
		t.Fatal("文件不存在应报错")
	}
}

func TestParseCSV_EmptyOrHeaderOnly(t *testing.T) {
	path := writeTempCSV(t, "variable,coef,se,pvalue\n")
	_, err := ParseCSV(path)
	if err == nil {
		t.Fatal("只有表头无数据应报错")
	}
}

func TestParseCSV_MissingColumn(t *testing.T) {
	path := writeTempCSV(t, "variable,coef\nDigital,0.052\n")
	_, err := ParseCSV(path)
	if err == nil {
		t.Fatal("缺列应报错")
	}
	if !strings.Contains(err.Error(), "缺少列") {
		t.Errorf("错误信息应含'缺少列'，实际: %v", err)
	}
}

func TestParseCSV_StataMissingValue(t *testing.T) {
	path := writeTempCSV(t, "variable,coef,se,pvalue\nDigital,.,0.015,0.001\n")
	_, err := ParseCSV(path)
	if err == nil {
		t.Fatal("Stata 缺失值 . 应报错，不能当 0")
	}
	if !strings.Contains(err.Error(), "缺失值") {
		t.Errorf("错误信息应含'缺失值'，实际: %v", err)
	}
}

func TestParseCSV_InvalidNumber(t *testing.T) {
	path := writeTempCSV(t, "variable,coef,se,pvalue\nDigital,abc,0.015,0.001\n")
	_, err := ParseCSV(path)
	if err == nil {
		t.Fatal("非法数值应报错")
	}
}

func TestParseCSV_ColumnOrderIndependent(t *testing.T) {
	// 表头顺序打乱也能解析（按列名取，不按位置）
	path := writeTempCSV(t, "coef,se,pvalue,variable\n0.052,0.015,0.001,Digital\n")
	r, err := ParseCSV(path)
	if err != nil {
		t.Fatalf("列顺序打乱不应报错: %v", err)
	}
	if r.Variable != "Digital" || r.Coef != 0.052 {
		t.Errorf("列顺序打乱后解析错误: %+v", r)
	}
}
