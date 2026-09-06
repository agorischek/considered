# Diagnostic only: no release writes, credentials, AV exclusions, or disabled protection.
# The upstream manifest commit and archive hashes are intentionally immutable.
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$out = Join-Path $env:RUNNER_TEMP 'considered-diagnostics'
New-Item -ItemType Directory -Force $out | Out-Null
$started = Get-Date
Start-Transcript -Path (Join-Path $out 'transcript.txt')
$failures = [System.Collections.Generic.List[string]]::new()

function Record-Step([string]$Name, [scriptblock]$Action) {
    Write-Host "=== $Name ==="
    try { & $Action } catch {
        $failures.Add("${Name}: $_")
        Write-Warning "${Name}: $_"
    }
}

try {
    Record-Step 'Defender health and signatures' {
        $status = Get-MpComputerStatus
        $status | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $out 'defender-before.json')
        if (-not $status.AntivirusEnabled -or -not $status.RealTimeProtectionEnabled) {
            throw 'Defender is not active on this runner; this cannot establish a clean result.'
        }
        Update-MpSignature
    }
    $arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq 'Arm64') { 'arm64' } else { 'amd64' }
    $expected = @{
        amd64 = 'af5e6e4f53c7a7ad4db7ddcc5224dc55f46db541e6d5ed5b6a9f5cbf16574130'
        arm64 = 'f30f9216b3c6f14fc7fbe7c40c2e15b33aa71414c20d37076e6bafb38e966d9a'
    }
    $archive = Join-Path $env:RUNNER_TEMP "considered_v0.1.11_windows_${arch}.zip"
    $expanded = Join-Path $env:RUNNER_TEMP 'considered-published-binaries'
    Record-Step 'Verify and scan published archive' {
        Invoke-WebRequest "https://github.com/quitepicky/considered/releases/download/v0.1.11/considered_v0.1.11_windows_${arch}.zip" -OutFile $archive
        $hash = (Get-FileHash $archive -Algorithm SHA256).Hash
        if ($hash -ne $expected[$arch]) { throw "Archive hash mismatch: $hash" }
        Start-MpScan -ScanType CustomScan -ScanPath $archive
        Expand-Archive $archive $expanded
        Get-ChildItem $expanded -Recurse -File | ForEach-Object {
            [pscustomobject]@{Name=$_.Name; SHA256=(Get-FileHash $_.FullName).Hash; Bytes=$_.Length}
        } | ConvertTo-Json | Set-Content (Join-Path $out 'binary-hashes.json')
        Start-MpScan -ScanType CustomScan -ScanPath $expanded
    }
    Record-Step 'Run published executables directly' {
        $bin = Join-Path $expanded "considered_v0.1.11_windows_$arch"
        & (Join-Path $bin 'considered.exe') --version
        if ($LASTEXITCODE -ne 0) { throw "considered version failed: $LASTEXITCODE" }
        $fixture = Join-Path $env:RUNNER_TEMP 'considered-scan-fixture'
        New-Item -ItemType Directory -Force $fixture | Out-Null
        'package example' | Set-Content (Join-Path $fixture 'example.go')
        & (Join-Path $bin 'considered-scc.exe') --json --root $fixture
        if ($LASTEXITCODE -ne 0) { throw "considered-scc failed: $LASTEXITCODE" }
    }
    Record-Step 'Install original manifest through WinGet' {
        if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
            Install-Module Microsoft.WinGet.Client -RequiredVersion 1.29.280 -Force -Repository PSGallery -Scope CurrentUser
            Import-Module Microsoft.WinGet.Client
            Repair-WinGetPackageManager -AllUsers
            $env:PATH += ";$env:LOCALAPPDATA\Microsoft\WindowsApps"
        }
        & winget --info
        if ($LASTEXITCODE -ne 0) { throw 'WinGet unavailable' }
        & winget settings --enable LocalManifestFiles
        if ($LASTEXITCODE -ne 0) { throw 'Unable to enable local manifests' }
        $manifest = Join-Path $out 'manifest'
        New-Item -ItemType Directory -Force $manifest | Out-Null
        foreach ($name in @('QuitePicky.Considered.yaml', 'QuitePicky.Considered.installer.yaml', 'QuitePicky.Considered.locale.en-US.yaml')) {
            Invoke-WebRequest "https://raw.githubusercontent.com/quitepicky/winget-pkgs/ac7f5565afba745a411fd551423525c1bd4791c3/manifests/q/QuitePicky/Considered/0.1.11/$name" -OutFile (Join-Path $manifest $name)
        }
        & winget validate --manifest $manifest
        if ($LASTEXITCODE -ne 0) { throw "Manifest validation failed: $LASTEXITCODE" }
        & winget install --manifest $manifest --accept-package-agreements --accept-source-agreements --disable-interactivity --verbose-logs
        if ($LASTEXITCODE -ne 0) { throw "WinGet installation failed: $LASTEXITCODE" }
        $links = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Links'
        $env:PATH = "$links;$env:PATH"
        & considered --version
        if ($LASTEXITCODE -ne 0) { throw "Installed considered failed: $LASTEXITCODE" }
        Start-MpScan -ScanType CustomScan -ScanPath (Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet')
    }
} finally {
    Record-Step 'Capture Defender findings' {
        Get-MpComputerStatus | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $out 'defender-after.json')
        $detections = @(Get-MpThreatDetection)
        $detections | ConvertTo-Json -Depth 8 | Set-Content (Join-Path $out 'defender-detections.json')
        Get-MpThreat | ConvertTo-Json -Depth 8 | Set-Content (Join-Path $out 'defender-threats.json')
        if ($detections.Count -gt 0) { $failures.Add('Defender reported threat detections; see evidence.') }
    }
    # Absence of events is normal; command errors are recorded in the transcript.
    Get-WinEvent -FilterHashtable @{LogName='Microsoft-Windows-Windows Defender/Operational'; StartTime=$started} -ErrorAction Continue |
        Select-Object TimeCreated, Id, LevelDisplayName, Message |
        ConvertTo-Json -Depth 6 | Set-Content (Join-Path $out 'defender-events.json')
    foreach ($logs in @(
        "$env:LOCALAPPDATA\Packages\Microsoft.DesktopAppInstaller_8wekyb3d8bbwe\LocalState\DiagOutputDir",
        "$env:LOCALAPPDATA\Microsoft\WinGet\DiagOutputDir"
    )) {
        if (Test-Path $logs) { Copy-Item $logs (Join-Path $out ([guid]::NewGuid().ToString())) -Recurse }
    }
    $failures | ConvertTo-Json | Set-Content (Join-Path $out 'failures.json')
    Stop-Transcript
}
if ($failures.Count -gt 0) { throw ($failures -join "`n") }
