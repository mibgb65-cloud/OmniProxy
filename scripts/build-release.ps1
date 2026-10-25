param([Parameter(Mandatory=$true)][string]$Version,[string]$OutputDirectory='release-assets')
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$previousPath=$env:PATH
Push-Location $root
try{
 if(-not(Get-Command makensis -ErrorAction SilentlyContinue)){
  $nsis=Join-Path ${env:ProgramFiles(x86)} 'NSIS'
  if(-not(Test-Path -LiteralPath (Join-Path $nsis 'makensis.exe'))){throw '请安装 NSIS 或将 makensis 加入 PATH'}
  $env:PATH="$nsis;$env:PATH"
 }
 & "$PSScriptRoot/set-release-version.ps1" -Version $Version
 if(-not(Test-Path -LiteralPath "docs/releases/$Version.md" -PathType Leaf)){throw '请先添加人工整理的发布说明'}
 # task 会把进度行写到 stderr（其中包含 Windows 上缺少 uname 的提示）；在 Stop 下
 # PowerShell 会把原生 stderr 当成终止错误，导致脚本在质量门开跑前就退出。
 $gateErrorAction=$ErrorActionPreference
 $ErrorActionPreference='Continue'
 & task --force verify
 $verifyCode=$LASTEXITCODE
 & task --force package
 $packageCode=$LASTEXITCODE
 $ErrorActionPreference=$gateErrorAction
 if($verifyCode -ne 0){throw '完整质量门失败'}
 if($packageCode -ne 0){throw '安装器打包失败'}
 $application=Join-Path $root 'bin/omniproxy.exe'
 $installer=Join-Path $root 'bin/OmniProxy-amd64-installer.exe'
 & "$PSScriptRoot/test-windows-metadata.ps1" -Application $application -Installer $installer -Version $Version
 # Windows PowerShell 5.1 没有 GetFullPath(path, basePath) 重载，先拼接再规范化。
 $output=if([IO.Path]::IsPathRooted($OutputDirectory)){$OutputDirectory}else{Join-Path $root $OutputDirectory}
 $output=[IO.Path]::GetFullPath($output)
 if(Test-Path -LiteralPath $output){if(@(Get-ChildItem -LiteralPath $output -Force).Count -gt 0){throw '发布目录非空，拒绝覆盖已有发布资产'}}
 New-Item -ItemType Directory -Path $output -Force|Out-Null
 $name="OmniProxy-Setup-$Version-windows-amd64.exe"
 Copy-Item -LiteralPath $installer -Destination (Join-Path $output $name)
 $hash=(Get-FileHash -LiteralPath (Join-Path $output $name) -Algorithm SHA256).Hash.ToLowerInvariant()
 [IO.File]::WriteAllText((Join-Path $output "$name.sha256"),"$hash  $name`n",[Text.Encoding]::ASCII)
 & "$PSScriptRoot/verify-release-assets.ps1" -Version $Version -AssetDirectory $output
}finally{Pop-Location;$env:PATH=$previousPath}
