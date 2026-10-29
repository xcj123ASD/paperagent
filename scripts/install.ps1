# paperagent 安装检查脚本（Windows）
# 用法：powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1

$ErrorActionPreference = "Stop"

Write-Host "=== paperagent 安装检查 ===" -ForegroundColor Cyan
Write-Host ""

# 1. 检测 Go 环境（exe 分发包可跳过）
$hasGo = $null -ne (Get-Command go -ErrorAction SilentlyContinue)
if ($hasGo) {
    Write-Host "[1/4] Go 环境：已安装" -ForegroundColor Green
} else {
    Write-Host "[1/4] Go 环境：未安装（使用 exe 分发包时不需要）" -ForegroundColor Yellow
}

# 2. 检测 Pandoc
$pandocOK = $false
if ($env:PANDOC_PATH -and (Test-Path $env:PANDOC_PATH)) {
    $pandocOK = $true
    Write-Host "[2/4] Pandoc：已配置（PANDOC_PATH=$env:PANDOC_PATH）" -ForegroundColor Green
} elseif (Get-Command pandoc -ErrorAction SilentlyContinue) {
    $pandocOK = $true
    Write-Host "[2/4] Pandoc：已安装（在 PATH 中）" -ForegroundColor Green
} elseif (Test-Path "$env:USERPROFILE\.paperagent\pandoc\pandoc.exe") {
    $pandocOK = $true
    Write-Host "[2/4] Pandoc：已安装（用户目录）" -ForegroundColor Green
} else {
    Write-Host "[2/4] Pandoc：未安装" -ForegroundColor Yellow
    Write-Host "      推荐安装：winget install JohnMacFarlane.Pandoc"
    Write-Host "      或运行：paperagent --install-pandoc（需外网）"
}

# 3. 检测 LLM API Key
$hasKey = $false
foreach ($k in @("DEEPSEEK_API_KEY", "ZAI_API_KEY", "ZHIPU_API_KEY")) {
    $v = [Environment]::GetEnvironmentVariable($k, "User")
    if ($v) {
        $hasKey = $true
        Write-Host "[3/4] API Key：已配置（$k）" -ForegroundColor Green
        break
    }
}
if (-not $hasKey) {
    Write-Host "[3/4] API Key：未配置" -ForegroundColor Yellow
    Write-Host "      DeepSeek（推荐）：https://platform.deepseek.com/"
    Write-Host "      配置命令：setx DEEPSEEK_API_KEY 你的Key"
}

# 4. 汇总
Write-Host ""
Write-Host "[4/4] 安装检查完成" -ForegroundColor Cyan
Write-Host ""
if ($hasGo) {
    Write-Host "运行方式："
    Write-Host "  cd E:\lunwenagent"
    Write-Host "  . .\load.ps1"
    Write-Host "  go run .\cmd\paper --design design.json --refs refs_real.json --export docx"
} else {
    Write-Host "运行方式：双击 paperagent.exe（exe 分发包）"
}
Write-Host ""
if (-not $pandocOK) {
    Write-Host "注意：Pandoc 未安装，--export docx 会失败。先装 Pandoc 再导出。" -ForegroundColor Yellow
}
if (-not $hasKey) {
    Write-Host "注意：API Key 未配置，会走 Mock 模式（占位内容）。" -ForegroundColor Yellow
}