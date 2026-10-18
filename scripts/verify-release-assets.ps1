param([Parameter(Mandatory=$true)][string]$Version,[Parameter(Mandatory=$true)][string]$AssetDirectory)
$ErrorActionPreference='Stop'
if($Version -notmatch '^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$'){throw '标签无效'}
$name="OmniProxy-Setup-$Version-windows-amd64.exe"
$expected=@($name,"$name.sha256")
$files=@(Get-ChildItem -LiteralPath $AssetDirectory -File)
if($files.Count -ne 2 -or @($files|Where-Object{$_.Name -notin $expected}).Count -ne 0){throw '发布目录必须且只能包含安装器及其 SHA256 文件'}
$content=(Get-Content -LiteralPath (Join-Path $AssetDirectory "$name.sha256") -Raw).Trim()
if($content -notmatch ('^(?<hash>[a-f0-9]{64})  '+[regex]::Escape($name)+'$')){throw 'SHA256 文件格式不正确'}
$expectedHash=$Matches.hash
$actual=(Get-FileHash -LiteralPath (Join-Path $AssetDirectory $name) -Algorithm SHA256).Hash.ToLowerInvariant()
if($expectedHash -ne $actual){throw 'SHA256 不匹配'}
Write-Output "发布资产校验通过：$Version"
