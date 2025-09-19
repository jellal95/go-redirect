param(
    [string]$LocalLogsPath = "./logs",
    [switch]$NoMerge = $false
)

Write-Host "Starting Fly.io Log Download & Cleanup..." -ForegroundColor Cyan
if ($NoMerge) {
    Write-Host "Replace mode: Will replace existing local files" -ForegroundColor Magenta
} else {
    Write-Host "Merge mode: Will automatically merge with existing local files" -ForegroundColor Magenta
}

if (!(Test-Path $LocalLogsPath)) {
    New-Item -ItemType Directory -Path $LocalLogsPath -Force | Out-Null
    Write-Host "Created local logs directory: $LocalLogsPath" -ForegroundColor Green
}

$machineList = flyctl machines list --json | ConvertFrom-Json
if (!$machineList -or $machineList.Count -eq 0) {
    Write-Host "No running machines found. Exiting..." -ForegroundColor Red
    exit 1
}

$machineId = $machineList[0].id
Write-Host "Found machine: $machineId" -ForegroundColor Yellow

Write-Host "Listing log files..." -ForegroundColor Yellow
$logFilesList = flyctl ssh console --machine $machineId -C "find /app/logs -name '*.jsonl' -type f"

if ([string]::IsNullOrWhiteSpace($logFilesList) -or $logFilesList -match "No such file") {
    Write-Host "No .jsonl log files found in /app/logs/" -ForegroundColor Red
    exit 1
}

$logFiles = $logFilesList -split "`n" | Where-Object { $_ -ne "" -and $_ -notmatch "Connecting" -and $_ -notmatch "Error" }

if ($logFiles.Count -eq 0) {
    Write-Host "No valid log files found" -ForegroundColor Red
    exit 1
}

Write-Host "Found $($logFiles.Count) log files:" -ForegroundColor Green
foreach ($file in $logFiles) {
    Write-Host "   - $(Split-Path $file -Leaf)" -ForegroundColor Gray
}

$downloadedCount = 0
$deletedCount = 0

foreach ($logFile in $logFiles) {
    $fileName = Split-Path $logFile -Leaf
    $localPath = Join-Path $LocalLogsPath $fileName
    
    $tempPath = "$localPath.temp"
    
    Write-Host "Downloading: $fileName" -ForegroundColor Cyan
    flyctl ssh sftp get $logFile $tempPath
    
    if ($LASTEXITCODE -eq 0 -and (Test-Path $tempPath)) {
        if ((Test-Path $localPath) -and !$NoMerge) {
            Write-Host "Merging with existing local file: $fileName" -ForegroundColor Yellow
            # Read existing and new content, merge and remove duplicates
            $existingContent = Get-Content $localPath -Raw
            $newContent = Get-Content $tempPath -Raw
            
            # For JSONL files, merge line by line to avoid duplicates
            if ($fileName -like "*.jsonl") {
                $existingLines = if ($existingContent) { $existingContent -split "`n" | Where-Object { $_.Trim() -ne "" } } else { @() }
                $newLines = if ($newContent) { $newContent -split "`n" | Where-Object { $_.Trim() -ne "" } } else { @() }
                
                # Combine and remove duplicates (assuming each line is unique JSON)
                $allLines = @($existingLines) + @($newLines) | Sort-Object -Unique
                $mergedContent = $allLines -join "`n"
                
                Set-Content $localPath $mergedContent -NoNewline
                $oldSize = [math]::Round((Get-Item $tempPath).Length / 1KB, 2)
                $newSize = [math]::Round((Get-Item $localPath).Length / 1KB, 2) 
                Write-Host "Merged: $fileName (was $oldSize KB, now $newSize KB)" -ForegroundColor Green
            } else {
                # For non-JSONL files, simple append
                Add-Content $localPath $newContent
                Write-Host "Appended to existing file: $fileName" -ForegroundColor Green
            }
            
            Remove-Item $tempPath -Force
        } else {
            if (Test-Path $localPath) {
                Remove-Item $localPath -Force
                Write-Host "Replaced existing local file: $fileName" -ForegroundColor Yellow
            }
            Move-Item $tempPath $localPath
        }
        
        $downloadedCount++
        Write-Host "Downloaded: $fileName" -ForegroundColor Green
        
        Write-Host "Deleting remote file: $fileName" -ForegroundColor Yellow
        flyctl ssh console --machine $machineId -C "rm `"$logFile`""
        
        if ($LASTEXITCODE -eq 0) {
            $deletedCount++
            Write-Host "Deleted remote file: $fileName" -ForegroundColor Green
        } else {
            Write-Host "Failed to delete remote file: $fileName" -ForegroundColor Red
        }
    } else {
        Write-Host "Failed to download: $fileName" -ForegroundColor Red
        if (Test-Path $tempPath) {
            Remove-Item $tempPath -Force
        }
    }
    
    Write-Host ""
}

Write-Host "Summary:" -ForegroundColor Cyan
Write-Host "   - Downloaded: $downloadedCount files" -ForegroundColor Gray
Write-Host "   - Deleted from remote: $deletedCount files" -ForegroundColor Gray
Write-Host "   - Local path: $((Get-Item $LocalLogsPath).FullName)" -ForegroundColor Gray

if ($downloadedCount -gt 0) {
    Write-Host ""
    Write-Host "Downloaded files:" -ForegroundColor Green
    Get-ChildItem $LocalLogsPath -Filter "*.jsonl" | Sort-Object LastWriteTime -Descending | ForEach-Object {
        $sizeKB = [math]::Round($_.Length / 1KB, 2)
        Write-Host "   - $($_.Name) ($sizeKB KB)" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "Log download and cleanup completed!" -ForegroundColor Green