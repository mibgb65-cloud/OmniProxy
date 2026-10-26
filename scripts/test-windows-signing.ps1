param([Parameter(Mandatory = $true)][string]$Installer)
$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/windows-signing.ps1"
if (-not (Test-Path -LiteralPath $Installer -PathType Leaf)) { throw '未签名安装器夹具不存在' }
$certificate = Get-ProjectSigningCertificate
try {
    if ($certificate.HasPrivateKey) { throw '公开证书包含私钥' }
    $rejected = $false
    try { Assert-ProjectAuthenticode -Path $Installer -ExpectedCertificate $certificate }
    catch { $rejected = $true }
    if (-not $rejected) { throw '未签名夹具未被拒绝' }
    $systemSignedFile = Join-Path $env:WINDIR 'System32/WindowsPowerShell/v1.0/powershell.exe'
    $rejected = $false
    try { Assert-ProjectAuthenticode -Path $systemSignedFile -ExpectedCertificate $certificate }
    catch { $rejected = $true }
    if (-not $rejected) { throw '其他签名者未被 pin 拒绝' }
    if ($env:GITHUB_ACTIONS -ne 'true') {
        $called = $false
        $rejected = $false
        try { Invoke-WithProjectCertificateTrust { $script:called = $true } }
        catch { $rejected = $true }
        if (-not $rejected -or $called) { throw '本机触发了临时根信任' }
    }
    Write-Output 'PASS: public-only certificate; unsigned/wrong signer rejection; no local trust-store changes'
    Write-Output "Public certificate SHA256: $(Get-CertificateSha256 $certificate)"
} finally { $certificate.Dispose() }
