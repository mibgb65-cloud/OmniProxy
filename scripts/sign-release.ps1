param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$AssetDirectory
)
$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/windows-signing.ps1"
if ($Version.Length -gt 128 -or $Version -notmatch '^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') { throw '发布版本无效' }
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_OS -ne 'Windows' -or -not $env:RUNNER_TEMP) {
    throw '签名脚本只用于隔离的 GitHub Actions Windows Runner'
}
if ([string]::IsNullOrWhiteSpace($env:OMNIPROXY_SIGNING_PFX) -or $env:OMNIPROXY_SIGNING_PFX.Length -gt 65536 -or
    [string]::IsNullOrWhiteSpace($env:OMNIPROXY_SIGNING_PASSWORD) -or $env:OMNIPROXY_SIGNING_PASSWORD.Length -gt 256) {
    throw '项目自签名 encrypted Secrets 未配置'
}
$name = "OmniProxy-Setup-$Version-windows-amd64.exe"
$path = Join-Path $AssetDirectory $name
& "$PSScriptRoot/verify-release-assets.ps1" -Version $Version -AssetDirectory $AssetDirectory
Assert-ProjectInstallerMetadata -Version $Version -Path $path
$expected = Get-ProjectSigningCertificate
$pfxPath = Join-Path $env:RUNNER_TEMP ("omniproxy-signing-" + [guid]::NewGuid().ToString('N') + '.pfx')
$imported = $null
$removeImported = $false
try {
    $bytes = [Convert]::FromBase64String($env:OMNIPROXY_SIGNING_PFX)
    $password = ConvertTo-SecureString $env:OMNIPROXY_SIGNING_PASSWORD -AsPlainText -Force
    $env:OMNIPROXY_SIGNING_PFX = $null
    $env:OMNIPROXY_SIGNING_PASSWORD = $null
    try { [IO.File]::WriteAllBytes($pfxPath, $bytes) }
    finally { [Security.Cryptography.CryptographicOperations]::ZeroMemory($bytes) }
    $removeImported = -not (Test-Path -LiteralPath "Cert:\CurrentUser\My\$($expected.Thumbprint)")
    $imported = Import-PfxCertificate -FilePath $pfxPath -Password $password -CertStoreLocation 'Cert:\CurrentUser\My'
    if (-not $imported.HasPrivateKey -or (Get-CertificateSha256 $imported) -ne (Get-CertificateSha256 $expected)) {
        throw '签名凭据与已提交的公开证书不匹配'
    }
    Invoke-WithProjectCertificateTrust {
        param($certificate)
        $signature = Set-AuthenticodeSignature -LiteralPath $path -Certificate $imported -HashAlgorithm SHA256
        if ($signature.Status -ne 'Valid') { throw '安装器自签名失败' }
        Assert-ProjectAuthenticode -Path $path -ExpectedCertificate $certificate
        # 在真实签名文件的被哈希代码区翻转一字节，确保验签不是只检查证书是否存在。
        $tampered = Join-Path $env:RUNNER_TEMP ("omniproxy-tampered-" + [guid]::NewGuid().ToString('N') + '.exe')
        try {
            Copy-Item -LiteralPath $path -Destination $tampered
            $stream = [IO.File]::Open($tampered, 'Open', 'ReadWrite', 'None')
            try {
                if ($stream.Length -le 4096) { throw '安装器夹具过小' }
                $stream.Position = 4096
                $original = $stream.ReadByte()
                $stream.Position = 4096
                $stream.WriteByte($original -bxor 1)
            } finally { $stream.Dispose() }
            $rejected = $false
            try { Assert-ProjectAuthenticode -Path $tampered -ExpectedCertificate $certificate }
            catch { $rejected = $true }
            if (-not $rejected) { throw '篡改安装器未被拒绝' }
        } finally { if (Test-Path -LiteralPath $tampered) { Remove-Item -LiteralPath $tampered } }
    }
    & "$PSScriptRoot/verify-release-signature.ps1" -Version $Version -AssetDirectory $AssetDirectory -WriteChecksum
} finally {
    if ($imported) { $imported.Dispose() }
    if ($removeImported -and $expected.Thumbprint -match '^[0-9A-Fa-f]{40}$' -and
        (Test-Path -LiteralPath "Cert:\CurrentUser\My\$($expected.Thumbprint)")) {
        Remove-Item -LiteralPath "Cert:\CurrentUser\My\$($expected.Thumbprint)" -DeleteKey -Force
    }
    if (Test-Path -LiteralPath $pfxPath) { Remove-Item -LiteralPath $pfxPath }
    if ($password) { $password.Dispose() }
    $expected.Dispose()
}
