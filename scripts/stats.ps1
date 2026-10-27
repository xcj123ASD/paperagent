param(
    [int]$Runs = 3,
    [string]$Csv = "stats.csv"
)

$ErrorActionPreference = "Continue"
$results = @()
New-Item -ItemType Directory -Path runs -Force | Out-Null

for ($i = 1; $i -le $Runs; $i++) {
    Write-Host "=== Run $i / $Runs ==="
    $start = Get-Date
    $out = & go run .\cmd\paper --design design.json --mode journal --refs refs_real.json 2>&1 | Out-String
    $elapsed = ((Get-Date) - $start).TotalSeconds
    $out | Out-File "runs\run$i.txt" -Encoding UTF8
    if (Test-Path .\draft.md) { Copy-Item .\draft.md "runs\run$i.md" -Force }

    $pass = $out -match "QUALITY_STATUS=PASS"
    $fail = $out -match "QUALITY_STATUS=FAIL"
    $phMatch = [regex]::Match($out, "QUALITY_PLACEHOLDERS=(\d+)")
    $phCount = if ($phMatch.Success) { [int]$phMatch.Groups[1].Value } else { 0 }

    $intro = 0; $lit = 0; $concl = 0
    if (Test-Path .\draft.md) {
        $lines = Get-Content .\draft.md -Encoding UTF8
        $cur = ""; $len = 0
        $sectionLens = @{}
        foreach ($l in $lines) {
            if ($l -match "^## ") {
                if ($cur) { $sectionLens[$cur] = $len }
                $cur = $l; $len = 0
            } else { $len += $l.Length }
        }
        if ($cur) { $sectionLens[$cur] = $len }
        if ($sectionLens.ContainsKey("## 引言")) { $intro = $sectionLens["## 引言"] }
        if ($sectionLens.ContainsKey("## 文献综述与研究假设")) { $lit = $sectionLens["## 文献综述与研究假设"] }
        if ($sectionLens.ContainsKey("## 结论与政策启示")) { $concl = $sectionLens["## 结论与政策启示"] }
    }

    $status = if ($pass) { "PASS" } elseif ($fail) { "FAIL" } else { "UNKNOWN" }
    $results += [PSCustomObject]@{
        Run      = $i
        Status   = $status
        Intro    = $intro
        Lit      = $lit
        Concl    = $concl
        Placeholder = $phCount
        Seconds  = [math]::Round($elapsed, 1)
    }
    Write-Host "  $status  intro=$intro lit=$lit concl=$concl ph=$phCount ${elapsed}s"
}

$results | Export-Csv -Path $Csv -NoTypeInformation -Encoding UTF8
Write-Host ""
Write-Host "=== Summary ==="
$passCount = ($results | Where-Object { $_.Status -eq "PASS" }).Count
$rate = [math]::Round(100.0 * $passCount / $Runs, 1)
Write-Host "Runs: $Runs  Pass: $passCount  Rate: $rate%"
$results | Format-Table -AutoSize