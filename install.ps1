[CmdletBinding()]
param(
    [ValidateSet("auto", "claude", "claude-desktop", "codex", "hermes", "opencode", "aider")]
    [string]$Tools = "auto",
    [string]$Model = "",
    [switch]$Yes,
    [switch]$TokenStdin,
    [switch]$DryRun,
    [switch]$SkipApiCheck
)

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$PrimaryBase = "https://modelapi.aiaiaiaiai.cloud/aiengine-setup/current"
$FallbackBase = "https://raw.githubusercontent.com/1351468279/aiengine-switch/setup-assets"

if ([Environment]::Is64BitOperatingSystem -eq $false) {
    throw "仅支持 64 位 Windows"
}
$ProcessorArchitecture = $env:PROCESSOR_ARCHITEW6432
if (-not $ProcessorArchitecture) {
    $ProcessorArchitecture = $env:PROCESSOR_ARCHITECTURE
}
$Architecture = switch ($ProcessorArchitecture) {
    "ARM64" { "arm64" }
    "AMD64" { "amd64" }
    default { throw "不支持的 CPU 架构: $ProcessorArchitecture" }
}
$Archive = "aiengine-setup_windows_${Architecture}.zip"
$SetupTemp = Join-Path ([IO.Path]::GetTempPath()) ("aiengine-setup-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $SetupTemp | Out-Null

function Get-Release([string]$Base) {
    $Checksums = Join-Path $SetupTemp "CHECKSUMS.txt"
    $Package = Join-Path $SetupTemp $Archive
    Remove-Item -Force -ErrorAction SilentlyContinue $Checksums, $Package
    try {
        Invoke-WebRequest -UseBasicParsing -Uri "$Base/CHECKSUMS.txt" -OutFile $Checksums
        Invoke-WebRequest -UseBasicParsing -Uri "$Base/$Archive" -OutFile $Package
        $Line = Get-Content $Checksums | Where-Object { $_ -match ("^[0-9a-fA-F]{64}\s+\*?" + [Regex]::Escape($Archive) + "$") } | Select-Object -First 1
        if (-not $Line) { return $false }
        $Expected = ($Line -split "\s+")[0].ToLowerInvariant()
        $Actual = (Get-FileHash -Algorithm SHA256 $Package).Hash.ToLowerInvariant()
        return $Actual -eq $Expected
    }
    catch {
        return $false
    }
}

try {
    Write-Host "正在获取适用于 Windows/$Architecture 的 AiEngine 安装器..."
    if (Get-Release $PrimaryBase) {
        Write-Host "下载源: AiEngine"
    }
    elseif (Get-Release $FallbackBase) {
        Write-Host "下载源: GitHub 备用源（AiEngine 下载源不可用）"
    }
    else {
        throw "主下载源和 GitHub 备用源均下载或校验失败"
    }

    $Extract = Join-Path $SetupTemp "extract"
    Expand-Archive -Path (Join-Path $SetupTemp $Archive) -DestinationPath $Extract
    $Binary = Join-Path $Extract "aiengine-setup.exe"
    if (-not (Test-Path -LiteralPath $Binary -PathType Leaf)) {
        throw "发布包中缺少 aiengine-setup.exe"
    }
    $BinaryArgs = @("install", "--tools", $Tools)
    if ($Model) { $BinaryArgs += @("--model", $Model) }
    if ($Yes) { $BinaryArgs += "--yes" }
    if ($TokenStdin) { $BinaryArgs += "--token-stdin" }
    if ($DryRun) { $BinaryArgs += "--dry-run" }
    if ($SkipApiCheck) { $BinaryArgs += "--skip-api-check" }
    & $Binary @BinaryArgs
    if ($LASTEXITCODE -ne 0) { throw "AiEngine 安装器返回错误 $LASTEXITCODE" }
}
finally {
    Remove-Item -LiteralPath $SetupTemp -Recurse -Force -ErrorAction SilentlyContinue
}
