package design

import (
	"encoding/json"
	"fmt"
	"os"

	"paperagent/internal/validate"
)

// Load 从 JSON 文件读取研究设计输入，并做必填校验。
func Load(path string) (DesignInput, error) {
	if err := validate.FileSize(path, 0); err != nil {
		return DesignInput{}, fmt.Errorf("design 文件检查失败: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DesignInput{}, fmt.Errorf("读取 design 文件失败: %w", err)
	}

	var d DesignInput
	if err := json.Unmarshal(data, &d); err != nil {
		return DesignInput{}, fmt.Errorf("解析 design JSON 失败: %w", err)
	}

	if err := d.Validate(); err != nil {
		return DesignInput{}, err
	}
	return d, nil
}
