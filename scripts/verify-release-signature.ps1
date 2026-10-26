param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$AssetDirectory,
    [switch]$WriteChecksum
)
$ErrorActionPreference = 'Stop'
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
$signature = Get-AuthenticodeSignature -LiteralPath $path
if ($signature.Status -ne 'Valid' -or -not $signature.SignerCertificate) {
    throw '安装器没有有效的 Authenticode 签名，拒绝公开发布'
}
if ($signature.SignerCertificate.Subject -notmatch '(^|,\s*)CN=SignPath Foundation(,|$)') {
    throw '安装器签名者不符合原仓库 SignPath Foundation 策略'
}
$metadata = (Get-Item -LiteralPath $path).VersionInfo
$numeric = ($Version.Substring(1) -split '-', 2)[0]
if ($metadata.ProductName -ne 'OmniProxy' -or $metadata.CompanyName -ne 'OmniProxy' -or
    $metadata.ProductVersion -ne $numeric) {
    throw '已签名安装器的产品或版本资源不匹配'
}
if ($WriteChecksum) {
    # SignPath 改变文件内容，旧摘要绝不能沿用；只对已验证的签名产物生成新摘要。
    $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
    [IO.File]::WriteAllText((Join-Path $AssetDirectory "$name.sha256"), "$hash  $name`n", [Text.Encoding]::ASCII)
}
& "$PSScriptRoot/verify-release-assets.ps1" -Version $Version -AssetDirectory $AssetDirectory
Write-Output "签名、签名者、产品版本和 SHA256 校验通过：$Version"
