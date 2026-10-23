package stata

import (
	"fmt"
	"strings"
)

// StataRequest 描述一次 Stata 回归任务。
//
// 安全设计：本结构只接收结构化字段（变量名、路径、模型类型），
// 不接收原始 Stata 命令字符串。所有命令由模板生成，模板外的
// 命令无法注入，这是任务 7 静态检查能做的前提。
type StataRequest struct {
	// 输入/输出
	DataPath   string `json:"data_path"`   // .dta 输入文件路径
	OutputPath string `json:"output_path"` // 结果 CSV 输出路径

	// 变量（仅变量名，不带命令）
	Dependent string   `json:"dependent"` // 被解释变量
	Core      string   `json:"core"`      // 核心解释变量
	Controls  []string `json:"controls"`  // 控制变量

	// 固定效应与聚类
	FixedEffects []string `json:"fixed_effects"` // 如 ["firm","year"]，可空
	Cluster      string   `json:"cluster"`       // 聚类变量，可空

	// 模型类型：ols / fe / did / iv
	Model string `json:"model"`

	// IV 专用（Model=="iv" 时有效）
	Instrument string `json:"instrument"` // 工具变量名

	// DID 专用（Model=="did" 时有效）
	DIDTreatment string `json:"did_treatment"` // 处理组变量
	DIDTime      string `json:"did_time"`      // 政策时间变量

	// 描述信息（只进注释，不进命令）
	Label string `json:"label"` // 任务标签，如"基准回归"
}

// Validate 检查必填字段与模型类型，防止生成空命令或未支持的模型。
//
// 分模型校验：
//   - did 必须提供 did_treatment 与 did_time，否则交互项语法会渲染成 c.##i.
//   - iv  必须提供 instrument，否则 2sls 括号内为空
func (r StataRequest) Validate() error {
	var missing []string
	if strings.TrimSpace(r.DataPath) == "" {
		missing = append(missing, "data_path")
	}
	if strings.TrimSpace(r.Dependent) == "" {
		missing = append(missing, "dependent")
	}
	if strings.TrimSpace(r.Core) == "" {
		missing = append(missing, "core")
	}
	if strings.TrimSpace(r.Model) == "" {
		missing = append(missing, "model")
	}
	if len(missing) > 0 {
		return fmt.Errorf("stata: 缺少必填字段: %s", strings.Join(missing, ", "))
	}

	switch r.Model {
	case "ols", "fe":
	case "did":
		if strings.TrimSpace(r.DIDTreatment) == "" {
			return fmt.Errorf("stata: model=did 时 did_treatment 必填")
		}
		if strings.TrimSpace(r.DIDTime) == "" {
			return fmt.Errorf("stata: model=did 时 did_time 必填")
		}
	case "iv":
		if strings.TrimSpace(r.Instrument) == "" {
			return fmt.Errorf("stata: model=iv 时 instrument 必填")
		}
	default:
		return fmt.Errorf("stata: 不支持的模型类型 %q，应为 ols/fe/did/iv", r.Model)
	}
	return nil
}
