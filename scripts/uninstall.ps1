# paperagent 卸载脚本（Windows）
# 用法：powershell -ExecutionPolicy Bypass -File .\scripts\uninstall.ps1
# 清理内容：
#   - ~/.paperagent/ 目录（含 pandoc 缓存、进度文件）
#   - 环境变量：DEEPSEEK_API_KEY / ZAI_API_KEY / ZHIPU_API_KEY / PANDOC_PATH / LLM_PROVIDER
# 不清理：项目源码目录本身（用户手动删）

$ErrorActionPreference = "Continue"

Write-Host "=== paperagent 卸载 ===" -ForegroundColor Cyan
Write-Host ""

$confirm = Read-Host "将清理 ~/.paperagent/ 目录和相关环境变量，确认？(y/N)"
if ($confirm -ne "y" -and $confirm -ne "Y") {
    Write-Host "已取消"
    exit 0
}

# 1. 删除用户目录
$userDir = Join-Path $env:USERPROFILE ".paperagent"
if (Test-Path $userDir) {
    Remove-Item -Path $userDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "[1/2] 已删除：$userDir" -ForegroundColor Green
} else {
    Write-Host "[1/2] 用户目录不存在，跳过：$userDir" -ForegroundColor Yellow
}

# 2. 清理环境变量
$vars = @("DEEPSEEK_API_KEY", "ZAI_API_KEY", "ZHIPU_API_KEY", "PANDOC_PATH", "LLM_PROVIDER")
$cleaned = 0
foreach ($k in $vars) {
    $v = [Environment]::GetEnvironmentVariable($k, "User")
    if ($v) {
        [Environment]::SetEnvironmentVariable($k, $null, "User")
        Write-Host "  已清除：$k" -ForegroundColor Green
        $cleaned++
    }
}
if ($cleaned -eq 0) {
    Write-Host "[2/2] 无环境变量需清理" -ForegroundColor Yellow
} else {
    Write-Host "[2/2] 已清除 $cleaned 个环境变量" -ForegroundColor Green
}

Write-Host ""
Write-Host "卸载完成。源码目录（如 E:\lunwenagent）未清理，请手动删除。" -ForegroundColor Cyan
Write-Host "提示：关闭当前终端后，环境变量清理才完全生效。" -ForegroundColor Yellow