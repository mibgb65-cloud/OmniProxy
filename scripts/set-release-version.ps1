param([Parameter(Mandatory=$true)][string]$Version)
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
if($Version -notmatch '^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?$'){throw '发布标签必须为完整 SemVer，带 v 前缀'}
$full=$Version.Substring(1)
$numeric=($full -split '-',2)[0]
$parts=$numeric.Split('.')
foreach($part in $parts){if([long]$part -gt 65535){throw 'Windows 版本号超出范围'}}
if($full.Contains('-')){foreach($part in ($full -split '-',2)[1].Split('.')){if($part -eq '' -or ($part -match '^\d+$' -and $part.Length -gt 1 -and $part.StartsWith('0'))){throw '预发布标识不合法'}}}
$source=[IO.File]::ReadAllText((Join-Path $root 'internal/version/version.go'))
if($source -notmatch '(?m)^var Version = "([^"]+)"'){throw '没有找到应用版本'}
if($Matches[1] -ne $full){throw "标签 $Version 与源码版本不一致"}
$package=Get-Content (Join-Path $root 'frontend/package.json') -Raw|ConvertFrom-Json
if($package.version -ne $full){throw '前端版本与标签不一致'}
$utf8=[Text.UTF8Encoding]::new($false)
$configPath=Join-Path $root 'build/config.yml'
$config=[IO.File]::ReadAllText($configPath)
$config=[regex]::Replace($config,'(?m)^  version: "[^"]+"',('  version: "'+$numeric+'"'))
[IO.File]::WriteAllText($configPath,$config,$utf8)
$infoPath=Join-Path $root 'build/windows/info.json'
$info=Get-Content $infoPath -Raw|ConvertFrom-Json
$info.fixed.file_version="$numeric.0"
$info.fixed.product_version="$numeric.0"
$info.info.'0409'.ProductVersion=$full
[IO.File]::WriteAllText($infoPath,($info|ConvertTo-Json -Depth 10)+"`n",$utf8)
$toolsPath=Join-Path $root 'build/windows/nsis/wails_tools.nsh'
$tools=[IO.File]::ReadAllText($toolsPath)
$tools=[regex]::Replace($tools,'!define INFO_PRODUCTVERSION "[^"]+"',('!define INFO_PRODUCTVERSION "'+$numeric+'"'))
# 不改动其他模板定义或安装身份。
[IO.File]::WriteAllText($toolsPath,$tools,$utf8)
Write-Output "已校验 $Version，Windows 数字版本 $numeric.0"
