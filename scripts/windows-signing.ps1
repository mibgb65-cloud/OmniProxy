$ErrorActionPreference = 'Stop'

function Get-ProjectSigningCertificate {
    $path = Join-Path $PSScriptRoot '../build/windows/codesign.cer'
    if ((Get-Item -LiteralPath $path).Length -gt 16384) { throw '项目公开证书超过大小限制' }
    $pem = [IO.File]::ReadAllText($path)
    $match = [regex]::Match($pem, '\A-----BEGIN CERTIFICATE-----\s+(?<data>[A-Za-z0-9+/=\s]+)-----END CERTIFICATE-----\s*\z')
    if ($pem.Length -gt 16384 -or -not $match.Success) { throw '项目公开证书 PEM 格式无效' }
    # CreateFromPemFile 同时寻找私钥；这里只解码公开 DER，绝不要求或读取源码中的私钥。
    $certificate = [Security.Cryptography.X509Certificates.X509Certificate2]::new(
        [Convert]::FromBase64String($match.Groups['data'].Value))
    $eku = @($certificate.Extensions | Where-Object { $_.Oid.Value -eq '2.5.29.37' })
    if ($certificate.HasPrivateKey -or $certificate.Subject -ne 'CN=OmniProxy Self-Signed' -or
        $certificate.Issuer -ne $certificate.Subject -or $eku.Count -ne 1 -or
        '1.3.6.1.5.5.7.3.3' -notin $eku[0].EnhancedKeyUsages.Value -or
        $certificate.NotBefore.ToUniversalTime() -gt [DateTime]::UtcNow -or
        $certificate.NotAfter.ToUniversalTime() -le [DateTime]::UtcNow) {
        $certificate.Dispose()
        throw '项目公开代码签名证书无效'
    }
    return $certificate
}

function Get-CertificateSha256($Certificate) {
    return [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($Certificate.RawData)).ToLowerInvariant()
}

function Invoke-WithProjectCertificateTrust([scriptblock]$Action) {
    if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_OS -ne 'Windows' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted') {
        throw '临时证书信任只允许用于隔离的 GitHub Actions Windows Runner'
    }
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw 'Runner 临时机器信任需要其现有管理员身份，不自动请求提权'
    }
    $certificate = Get-ProjectSigningCertificate
    $stores = [Collections.Generic.List[object]]::new()
    try {
        # 仅在临时 Runner 中信任精确 pin 的公开证书，以使用 Windows 原生验签检查文件完整性。
        # 安装器和用户应用不执行此逻辑，验证后仅删除本次加入的证书。
        foreach ($name in @('Root', 'TrustedPublisher')) {
            # 托管 Runner 的 CurrentUser Root 写入可能弹交互确认；仅在该临时 VM 使用机器库。
            Write-Output "Signing stage: temporary hosted-runner machine trust ($name)"
            $store = [Security.Cryptography.X509Certificates.X509Store]::new($name, 'LocalMachine')
            $store.Open([Security.Cryptography.X509Certificates.OpenFlags]::ReadWrite)
            $existing = $store.Certificates.Find('FindByThumbprint', $certificate.Thumbprint, $false)
            $added = $existing.Count -eq 0
            if ($added) { $store.Add($certificate) }
            $stores.Add([pscustomobject]@{ Store = $store; Added = $added })
        }
        & $Action $certificate
    } finally {
        foreach ($entry in $stores) {
            if ($entry.Added) { $entry.Store.Remove($certificate) }
            $entry.Store.Close()
        }
        $certificate.Dispose()
    }
}

function Assert-ProjectInstallerMetadata([string]$Version, [string]$Path) {
    if ($Version.Length -gt 128 -or $Version -notmatch '^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
        throw '发布版本无效'
    }
    $metadata = (Get-Item -LiteralPath $Path).VersionInfo
    $numeric = ($Version.Substring(1) -split '-', 2)[0]
    if ($metadata.ProductName -ne 'OmniProxy' -or $metadata.CompanyName -ne 'OmniProxy' -or $metadata.ProductVersion -ne $numeric) {
        throw '安装器产品或版本不匹配'
    }
}

function Assert-ProjectAuthenticode([string]$Path, $ExpectedCertificate) {
    $signature = Get-AuthenticodeSignature -LiteralPath $Path
    if ($signature.Status -ne 'Valid' -or -not $signature.SignerCertificate -or
        (Get-CertificateSha256 $signature.SignerCertificate) -ne (Get-CertificateSha256 $ExpectedCertificate)) {
        throw '安装器完整性、签名或项目证书 pin 验证失败'
    }
}
