# 用 HBuilderX 自带的 uniapp-cli-vite 在终端跑 H5。
# 编译器 / UTS 报错会打到这个终端。页面里的 console.log 仍在浏览器开发者工具。
$ErrorActionPreference = 'Stop'
$OwnerDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$CliDir = 'D:\Program Files\HBuilderX\plugins\uniapp-cli-vite'
$NodeExe = 'D:\Program Files\HBuilderX\plugins\node\node.exe'
$UniJs = Join-Path $CliDir 'node_modules\@dcloudio\vite-plugin-uni\bin\uni.js'
$Port = '5174'
if ($args.Count -ge 1) {
	$Port = [string]$args[0]
}
if (-not (Test-Path $UniJs)) {
	Write-Error "找不到 $UniJs 。请确认已安装 HBuilderX。"
}
if (-not (Test-Path $NodeExe)) {
	Write-Error "找不到 $NodeExe 。请确认已安装 HBuilderX。"
}
$env:UNI_INPUT_DIR = $OwnerDir
$env:VITE_ROOT_DIR = $OwnerDir
$env:UNI_OUTPUT_DIR = Join-Path $OwnerDir 'unpackage\dist\dev\web'
Set-Location $CliDir
Write-Host "UNI_INPUT_DIR=$env:UNI_INPUT_DIR"
Write-Host "http://127.0.0.1:$Port"
& $NodeExe $UniJs -p h5 --host 127.0.0.1 --port $Port
