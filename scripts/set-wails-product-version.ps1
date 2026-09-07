[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [string]$ConfigPath = (Join-Path $PSScriptRoot "..\OmniProxyBackend\wails.json")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$trimmedVersion = $Version.Trim()
if ($trimmedVersion.Length -gt 128) {
    throw "Version is longer than 128 characters."
}

$versionPattern = '^v?(?<major>0|[1-9]\d*)\.(?<minor>0|[1-9]\d*)\.(?<patch>0|[1-9]\d*)(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
$match = [regex]::Match($trimmedVersion, $versionPattern)
if (-not $match.Success) {
    throw "Version must match vMAJOR.MINOR.PATCH or vMAJOR.MINOR.PATCH-PRERELEASE: $Version"
}

$componentNames = @('major', 'minor', 'patch')
foreach ($componentName in $componentNames) {
    [uint32]$componentValue = 0
    if (-not [uint32]::TryParse($match.Groups[$componentName].Value, [ref]$componentValue) -or $componentValue -gt [uint16]::MaxValue) {
        throw "Version component '$componentName' must be between 0 and 65535: $Version"
    }
}

$numericVersion = "{0}.{1}.{2}" -f $match.Groups['major'].Value, $match.Groups['minor'].Value, $match.Groups['patch'].Value
$resolvedConfigPath = [System.IO.Path]::GetFullPath($ConfigPath)
if (-not (Test-Path -LiteralPath $resolvedConfigPath -PathType Leaf)) {
    throw "Wails config was not found: $resolvedConfigPath"
}

$config = Get-Content -LiteralPath $resolvedConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ($null -eq $config.info) {
    $config | Add-Member -MemberType NoteProperty -Name info -Value ([pscustomobject]@{})
}
if ($null -eq $config.info.PSObject.Properties['productVersion']) {
    $config.info | Add-Member -MemberType NoteProperty -Name productVersion -Value $numericVersion
} else {
    $config.info.productVersion = $numericVersion
}

# PE 的固定版本字段只能使用数字。完整的 beta 标签仍通过 main.appVersion 写入应用运行时版本。
$json = $config | ConvertTo-Json -Depth 20
$utf8WithoutBom = [System.Text.UTF8Encoding]::new($false)
[System.IO.File]::WriteAllText($resolvedConfigPath, $json + [Environment]::NewLine, $utf8WithoutBom)

Write-Output "Wails productVersion set to $numericVersion (release tag: $Version)"
