package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"paperagent/internal/cite"
	"paperagent/internal/config"
	"paperagent/internal/design"
	"paperagent/internal/export"
	"paperagent/internal/flow"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
	"paperagent/internal/quality"
	"paperagent/internal/render"
)

//go:embed index.html
var indexHTML []byte

type task struct {
	ID       string
	Status   string
	Logs     []string
	Err      string
	Output   string
	Started  time.Time
	Finished time.Time
}

var (
	tasks   = map[string]*task{}
	tasksMu sync.Mutex
)

func updateTask(id, status, errMsg, output string, logs []string) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	t, ok := tasks[id]
	if !ok {
		return
	}
	t.Status = status
	t.Err = errMsg
	if output != "" {
		t.Output = output
	}
	if logs != nil {
		t.Logs = logs
	}
	if status == "done" || status == "failed" {
		t.Finished = time.Now()
	}
}

func appendLog(id, msg string) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	if t, ok := tasks[id]; ok {
		t.Logs = append(t.Logs, msg)
	}
}

func getTask(id string) *task {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	t, ok := tasks[id]
	if !ok {
		return nil
	}
	cp := *t
	cp.Logs = make([]string, len(t.Logs))
	copy(cp.Logs, t.Logs)
	return &cp
}

func executeTask(id string, form url.Values) {
	defer func() {
		if r := recover(); r != nil {
			updateTask(id, "failed", fmt.Sprintf("panic: %v", r), "", nil)
		}
	}()

	topic := form.Get("topic")
	designPath := form.Get("design")
	refsPath := form.Get("refs")
	mode := form.Get("mode")
	exportFmt := form.Get("export")

	cfg := config.Config{Mock: true}
	cfg.Mode = model.Mode(mode)
	if cfg.Mode == "" {
		cfg.Mode = model.ModeThesis
	}

	if designPath != "" {
		d, err := design.Load(designPath)
		if err != nil {
			updateTask(id, "failed", "load design: "+err.Error(), "", nil)
			return
		}
		cfg.Design = &d
		appendLog(id, "design 加载成功: "+filepath.Base(designPath))
	}

	if refsPath != "" {
		refs, err := cite.LoadRefs(refsPath)
		if err != nil {
			updateTask(id, "failed", "load refs: "+err.Error(), "", nil)
			return
		}
		cfg.Refs = refs
		appendLog(id, fmt.Sprintf("refs 加载成功: %d 篇", len(refs)))
	}

	orch := flow.New(cfg, nil)
	input := map[string]interface{}{}
	if topic != "" {
		input["topic"] = topic
	}
	if cfg.Design != nil && cfg.Design.Topic != "" && topic == "" {
		input["topic"] = cfg.Design.Topic
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		updateTask(id, "failed", "marshal input: "+err.Error(), "", nil)
		return
	}

	ctx := pipeline.NewContext(cfg.Mock, cfg.Strict)

	step1JSON, err := orch.Run(inputJSON)
	if err != nil {
		updateTask(id, "failed", "step1: "+err.Error(), "", nil)
		return
	}
	appendLog(id, "step1 完成")

	st2 := pipeline.Registry["step2_frame"]
	if st2 == nil {
		updateTask(id, "failed", "step2_frame not registered", "", nil)
		return
	}
	step2JSON, err := st2.Run(ctx, step1JSON)
	if err != nil {
		updateTask(id, "failed", "step2: "+err.Error(), "", nil)
		return
	}
	appendLog(id, "step2 完成")

	st3 := pipeline.Registry["step3_litreview"]
	if st3 == nil {
		updateTask(id, "failed", "step3_litreview not registered", "", nil)
		return
	}
	step3JSON, err := st3.Run(ctx, step2JSON)
	if err != nil {
		updateTask(id, "failed", "step3: "+err.Error(), "", nil)
		return
	}
	appendLog(id, "step3 完成")

	var frame model.Frame
	if err := json.Unmarshal(step3JSON, &frame); err != nil {
		updateTask(id, "failed", "parse frame: "+err.Error(), "", nil)
		return
	}

	for _, entry := range ctx.Snapshot() {
		appendLog(id, "["+entry.Stage+"] "+entry.Msg)
	}

	var refsText string
	if refsPath != "" {
		refsText, err = cite.ProcessRefs(refsPath, false, "")
		if err != nil {
			appendLog(id, "refs 格式化失败（继续）: "+err.Error())
		}
	}

	md := render.Markdown(frame, refsText)

	// 质量检测，结果写入日志
	report := quality.Check(md, cfg.Refs)
	qualityStr := "FAIL"
	if report.AllPass() {
		qualityStr = "PASS"
	}
	appendLog(id, fmt.Sprintf("质量检测: %s, 占位符 %d 处", qualityStr, report.PlaceholderCount))
	for _, h := range report.LitReviewIssues {
		appendLog(id, "综述问题: "+h)
	}
	for _, h := range report.CitationIssues {
		appendLog(id, "引用问题: "+h)
	}
	for _, h := range report.CitationWarnings {
		appendLog(id, "引用警告: "+h)
	}

	outputFile := fmt.Sprintf("draft_web_%s.md", id)
	if err := os.WriteFile(outputFile, []byte(md), 0644); err != nil {
		updateTask(id, "failed", "write output: "+err.Error(), "", nil)
		return
	}
	appendLog(id, "已写出: "+outputFile)

	if exportFmt != "" {
		outPath := replaceExt(outputFile, "."+exportFmt)
		e := export.NewPandocExporter()
		if err := e.Export(context.Background(), outputFile, outPath); err != nil {
			updateTask(id, "failed", "export: "+err.Error(), outputFile, nil)
			return
		}
		appendLog(id, "已导出: "+outPath)
	}

	updateTask(id, "done", "", outputFile, nil)
	appendLog(id, "全部完成")
}

func replaceExt(path, newExt string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[:i] + newExt
		}
	}
	return path + newExt
}

func main() {
	port := flag.Int("port", 8080, "Web 服务端口")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})

	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		id := fmt.Sprintf("%d", time.Now().UnixNano())
		t := &task{ID: id, Status: "pending", Started: time.Now()}
		tasksMu.Lock()
		tasks[id] = t
		tasksMu.Unlock()

		form := r.Form
		go executeTask(id, form)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
	})

	mux.HandleFunc("/api/progress", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		t := getTask(id)
		if t == nil {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       t.ID,
			"status":   t.Status,
			"logs":     t.Logs,
			"error":    t.Err,
			"output":   t.Output,
			"started":  t.Started.Format(time.RFC3339),
			"finished": t.Finished.Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		if file == "" {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		clean := filepath.Clean(file)
		if clean != file || filepath.IsAbs(file) || strings.Contains(clean, "..") {
			http.Error(w, "invalid file", http.StatusBadRequest)
			return
		}
		if _, err := os.Stat(clean); err != nil {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename="+url.QueryEscape(clean))
		http.ServeFile(w, r, clean)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Printf("paperagent web 启动：http://%s\n", addr)
	fmt.Println("按 Ctrl+C 停止")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
