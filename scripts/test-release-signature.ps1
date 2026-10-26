param(
    [Parameter(Mandatory = $true)][string]$Installer,
    [string]$Version = 'v2.0.0'
)
$ErrorActionPreference = 'Stop'
if ($Version.Length -gt 128 -or $Version -notmatch '^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
    throw 'Invalid signature-test version'
}
if (-not (Test-Path -LiteralPath $Installer -PathType Leaf)) {
    throw 'Signature-test installer fixture does not exist'
}
$directory = Join-Path ([IO.Path]::GetTempPath()) ("omniproxy-signature-test-" + [guid]::NewGuid().ToString('N'))
$name = "OmniProxy-Setup-$Version-windows-amd64.exe"
$null = New-Item -ItemType Directory -Path $directory
Copy-Item -LiteralPath $Installer -Destination (Join-Path $directory $name)
$verifier = Join-Path $PSScriptRoot 'verify-release-signature.ps1'

# 真实未签名文件必须被拒绝；后续 mock 只验证门禁分支，不代表真实签名验收。
$rejected = $false
try { & $verifier -Version $Version -AssetDirectory $directory -WriteChecksum }
catch { $rejected = $_.Exception.Message -like '*没有有效的 Authenticode*' }
if (-not $rejected) { throw 'Unsigned installer was not rejected' }
if (Test-Path -LiteralPath (Join-Path $directory "$name.sha256")) { throw 'Unsigned installer received a checksum' }

& {
    function Get-AuthenticodeSignature {
        param([string]$LiteralPath)
        [pscustomobject]@{ Status = 'Valid'; SignerCertificate = [pscustomobject]@{ Subject = 'CN=Untrusted Test Publisher' } }
    }
    $rejected = $false
    try { & $verifier -Version $Version -AssetDirectory $directory -WriteChecksum }
    catch { $rejected = $_.Exception.Message -like '*签名者不符合*' }
    if (-not $rejected) { throw 'Wrong publisher was not rejected' }
}

& {
    function Get-AuthenticodeSignature {
        param([string]$LiteralPath)
        [pscustomobject]@{ Status = 'Valid'; SignerCertificate = [pscustomobject]@{ Subject = 'CN=SignPath Foundation, O=SignPath Foundation' } }
    }
    & $verifier -Version $Version -AssetDirectory $directory -WriteChecksum
    & $verifier -Version $Version -AssetDirectory $directory
    [IO.File]::WriteAllText((Join-Path $directory "$name.sha256"), ("0" * 64) + "  $name`n")
    $rejected = $false
    try { & $verifier -Version $Version -AssetDirectory $directory }
    catch { $rejected = $_.Exception.Message -like '*SHA256 不匹配*' }
    if (-not $rejected) { throw 'Stale checksum was not rejected' }
}
Write-Output 'PASS: real unsigned rejection; mocked publisher gate, signed-file rehash and stale-checksum rejection'
Write-Output "Only synthetic test outputs were written to: $directory"
