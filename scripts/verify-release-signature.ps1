param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$AssetDirectory,
    [switch]$WriteChecksum
)
$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/windows-signing.ps1"
if ($Version.Length -gt 128 -or $Version -notmatch '^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
    throw '发布标签无效'
}
$name = "OmniProxy-Setup-$Version-windows-amd64.exe"
$expected = @($name, "$name.sha256")
$files = @(Get-ChildItem -LiteralPath $AssetDirectory -File)
if (@($files | Where-Object { $_.Name -notin $expected }).Count -ne 0 -or
    @($files | Where-Object { $_.Name -eq $name }).Count -ne 1 -or
    $files.Count -gt 2 -or (-not $WriteChecksum -and $files.Count -ne 2)) {
    throw '签名目录必须只包含目标安装器及可选的同名校验文件'
}
$path = Join-Path $AssetDirectory $name
Assert-ProjectInstallerMetadata -Version $Version -Path $path
Invoke-WithProjectCertificateTrust {
    param($certificate)
    Assert-ProjectAuthenticode -Path $path -ExpectedCertificate $certificate
}
if ($WriteChecksum) {
    # 签名改变文件内容，必须对已验证的最终签名文件重新计算摘要。
    $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
    [IO.File]::WriteAllText((Join-Path $AssetDirectory "$name.sha256"), "$hash  $name`n", [Text.Encoding]::ASCII)
}
& "$PSScriptRoot/verify-release-assets.ps1" -Version $Version -AssetDirectory $AssetDirectory
Write-Output "自签名完整性、项目证书 pin、版本和 SHA256 校验通过：$Version（不代表客户端系统信任）"
