# 在仓库根目录生成 .env（密钥 + 首次管理员密码）。已有 .env 时不覆盖。
#   pwsh -File deploy/init-env.ps1
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (Test-Path ".env") {
    Write-Host "[OK] 已有 $root\.env ，不覆盖"
    exit 0
}

if (-not (Test-Path ".env.example")) {
    Write-Error "找不到 .env.example"
}

function New-RandomBytes([int]$Count) {
    $bytes = New-Object byte[] $Count
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    return $bytes
}

$kek = [Convert]::ToBase64String((New-RandomBytes 48))
$passBytes = New-RandomBytes 16
$pass = -join ($passBytes | ForEach-Object { $_.ToString("x2") })

$content = @"
IMAGE_TAG=latest
FENGHUOLUN_PUBLISH=8088:8088
FENGHUOLUN_DATA_DIR=./data
FENGHUOLUN_TOKEN_KEK=$kek
FENGHUOLUN_CORS_ORIGINS=
FENGHUOLUN_ADMIN_BOOTSTRAP_USER=admin
FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD=$pass
"@

$utf8 = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText((Join-Path $root ".env"), $content.Trim() + "`n", $utf8)

Write-Host "[OK] 已写入 $root\.env"
Write-Host ""
Write-Host "----------------------------------------"
Write-Host "  首次管理员"
Write-Host "  用户: admin"
Write-Host "  密码: $pass"
Write-Host "----------------------------------------"
Write-Host ""
Write-Host "有域名请编辑 .env 的 FENGHUOLUN_CORS_ORIGINS。"
Write-Host "登录改密后，可从 .env 删除 BOOTSTRAP 两行。"
Write-Host "然后："
Write-Host "  docker compose up -d"
