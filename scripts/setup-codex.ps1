param(
    [switch]$SkipLogin,
    [switch]$ApiKeyStdin
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$RelayBaseUrl = "https://modelapi.aiaiaiaiai.cloud/v1"

if ($SkipLogin -and $ApiKeyStdin) {
    throw "-SkipLogin and -ApiKeyStdin cannot be used together"
}

if ([string]::IsNullOrWhiteSpace($env:CODEX_HOME)) {
    $CodexDir = Join-Path ([Environment]::GetFolderPath("UserProfile")) ".codex"
} else {
    $CodexDir = $env:CODEX_HOME
}

if ([string]::IsNullOrWhiteSpace($CodexDir)) {
    throw "HOME or CODEX_HOME must be set"
}

$ConfigPath = Join-Path $CodexDir "config.toml"
New-Item -ItemType Directory -Path $CodexDir -Force | Out-Null

if ((Test-Path -LiteralPath $ConfigPath) -and -not (Test-Path -LiteralPath $ConfigPath -PathType Leaf)) {
    throw "$ConfigPath exists but is not a regular file"
}

$ConfigExisted = Test-Path -LiteralPath $ConfigPath -PathType Leaf
if ($ConfigExisted) {
    $BackupStamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $BackupPath = "$ConfigPath.bak.$BackupStamp"
    $BackupIndex = 0
    while (Test-Path -LiteralPath $BackupPath) {
        $BackupIndex++
        $BackupPath = "$ConfigPath.bak.$BackupStamp-$BackupIndex"
    }
    Copy-Item -LiteralPath $ConfigPath -Destination $BackupPath
    Write-Host "Backed up existing config to $BackupPath"
}

if ($ConfigExisted) {
    $Content = [IO.File]::ReadAllText($ConfigPath)
    $Lines = $Content -split "\r?\n"
} else {
    $Lines = @()
}

$Output = [Collections.Generic.List[string]]::new()
$InTable = $false
$Inserted = $false
$FoundProvider = $false
$FoundUrl = $false

function Add-MissingSettings {
    if (-not $script:FoundProvider) {
        $script:Output.Add('model_provider = "openai"')
        $script:FoundProvider = $true
    }
    if (-not $script:FoundUrl) {
        $script:Output.Add("openai_base_url = `"$RelayBaseUrl`")
        $script:FoundUrl = $true
    }
}

foreach ($Line in $Lines) {
    if ($Line -match '^\s*\[') {
        if (-not $Inserted) {
            Add-MissingSettings
            $Inserted = $true
        }
        $InTable = $true
    }

    if (-not $InTable -and $Line -match '^\s*model_provider\s*=') {
        $Output.Add('model_provider = "openai"')
        $FoundProvider = $true
        continue
    }

    if (-not $InTable -and $Line -match '^\s*openai_base_url\s*=') {
        $Output.Add("openai_base_url = `"$RelayBaseUrl`")
        $FoundUrl = $true
        continue
    }

    $Output.Add($Line)
}

if (-not $Inserted) {
    Add-MissingSettings
}

$NewContent = ($Output -join [Environment]::NewLine) + [Environment]::NewLine
$Utf8NoBom = [Text.UTF8Encoding]::new($false)
[IO.File]::WriteAllText($ConfigPath, $NewContent, $Utf8NoBom)
Write-Host "Codex config written to $ConfigPath"

if ($SkipLogin) {
    Write-Host "Skipped API key login. Run this script again without -SkipLogin after installing Codex CLI."
    exit 0
}

$ApiKey = [string]$env:AIARE_CODEX_API_KEY
if ($ApiKeyStdin) {
    $ApiKey = [Console]::In.ReadLine()
} elseif ([string]::IsNullOrWhiteSpace($ApiKey)) {
    $SecureApiKey = Read-Host "Enter your AIARE NewAPI API key (input is hidden; press Enter to skip)" -AsSecureString
    $Pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureApiKey)
    try {
        $ApiKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($Pointer)
    } finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($Pointer)
    }
}

if ([string]::IsNullOrWhiteSpace($ApiKey)) {
    Write-Host "Config updated. API key login was skipped."
    exit 0
}

if (-not (Get-Command codex -ErrorAction SilentlyContinue)) {
    Remove-Variable ApiKey -ErrorAction SilentlyContinue
    throw "Codex CLI was not found. Install Codex CLI, then run this script again."
}

$ApiKey | & codex login --with-api-key
if ($LASTEXITCODE -ne 0) {
    Remove-Variable ApiKey -ErrorAction SilentlyContinue
    throw "Codex API key login failed; config.toml was still updated."
}

Remove-Variable ApiKey -ErrorAction SilentlyContinue
Write-Host "Codex API key login completed."
