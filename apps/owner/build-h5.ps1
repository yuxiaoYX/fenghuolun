# 生产 H5：输出到 apps/owner/unpackage/dist/build/web
$ErrorActionPreference = 'Stop'
$OwnerDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$CliDir = 'D:\Program Files\HBuilderX\plugins\uniapp-cli-vite'
$NodeExe = 'D:\Program Files\HBuilderX\plugins\node\node.exe'
$UniJs = Join-Path $CliDir 'node_modules\@dcloudio\vite-plugin-uni\bin\uni.js'
if (-not (Test-Path $UniJs)) {
	Write-Error "找不到 $UniJs 。请确认已安装 HBuilderX。"
}
$out = Join-Path $OwnerDir 'unpackage\dist\build\web'
$env:UNI_INPUT_DIR = $OwnerDir
$env:VITE_ROOT_DIR = $OwnerDir
$env:UNI_OUTPUT_DIR = $out
$env:NODE_ENV = 'production'
Set-Location $CliDir
Write-Host "UNI_INPUT_DIR=$env:UNI_INPUT_DIR"
Write-Host "UNI_OUTPUT_DIR=$env:UNI_OUTPUT_DIR"
& $NodeExe $UniJs build -p h5
if ($LASTEXITCODE -ne 0) {
	exit $LASTEXITCODE
}
$ship = Join-Path $OwnerDir 'h5-dist'
if (Test-Path $ship) {
	Remove-Item -Recurse -Force $ship
}
Copy-Item -Recurse $out $ship
Write-Host "H5 dist: $out"
Write-Host "Docker COPY: $ship"
