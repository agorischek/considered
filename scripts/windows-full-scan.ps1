param([Parameter(Mandatory)][string]$Evidence)
$ErrorActionPreference = 'Stop'
$scanRequested = Get-Date
Write-Host "Starting full post-install scan at $scanRequested"
Start-MpScan -ScanType FullScan
$deadline = $scanRequested.AddMinutes(65)
do {
    $status = Get-MpComputerStatus
    if ($status.FullScanStartTime -ge $scanRequested.AddSeconds(-2) -and
        $status.FullScanEndTime -ge $status.FullScanStartTime) {
        $status | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $Evidence 'full-scan-completed.json')
        Write-Host "Full scan completed at $($status.FullScanEndTime)"
        return
    }
    if ((Get-Date) -gt $deadline) { throw 'Full Defender scan did not complete within 65 minutes' }
    Start-Sleep -Seconds 15
} while ($true)
