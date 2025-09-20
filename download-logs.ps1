<#
.SYNOPSIS
    Download and cleanup log files from Fly.io deployment

.DESCRIPTION
    This script downloads JSONL log files from Fly.io persistent volume to local machine,
    with options for merging, filtering, backup, and remote cleanup.

.PARAMETER LocalLogsPath
    Local directory to store downloaded logs. Default: "./logs"

.PARAMETER NoMerge
    Replace existing local files instead of merging. Default: merge mode

.PARAMETER NoDelete
    Keep remote files after download instead of deleting them. Default: delete after download

.PARAMETER DateFilter
    Only process files matching this date pattern (e.g., "2024-01-15")

.PARAMETER Backup
    Create backup of existing local files before merge/replace

.EXAMPLE
    .\download-logs.ps1
    Basic usage: download all logs, merge with existing, delete remote files

.EXAMPLE
    .\download-logs.ps1 -LocalLogsPath "C:\MyLogs" -Backup
    Download to specific path with backup of existing files

.EXAMPLE
    .\download-logs.ps1 -DateFilter "2024-01-15" -NoDelete
    Download only files from specific date, keep remote files

.EXAMPLE
    .\download-logs.ps1 -NoMerge -Backup
    Replace mode with backup (don't merge, backup existing files first)

.NOTES
    - Requires flyctl CLI tool to be installed and authenticated
    - Works with Fly.io persistent volume at /logs path
    - Automatically retries failed downloads (up to 3 times)
    - JSONL files are merged intelligently to avoid duplicates
#>

param(
    [string]$LocalLogsPath = "./logs",
    [switch]$NoMerge = $false,
    [switch]$NoDelete = $false,
    [string]$DateFilter = "",
    [switch]$Backup = $false
)

Write-Host "=== Fly.io Log Download & Cleanup ===" -ForegroundColor Cyan
Write-Host ""

# Display configuration
Write-Host "Configuration:" -ForegroundColor Blue
if ($NoMerge) {
    Write-Host "   - Mode: Replace existing local files" -ForegroundColor Magenta
} else {
    Write-Host "   - Mode: Merge with existing local files" -ForegroundColor Magenta
}

if ($NoDelete) {
    Write-Host "   - Delete: Keep remote files (NoDelete mode)" -ForegroundColor Yellow
} else {
    Write-Host "   - Delete: Remove remote files after download" -ForegroundColor Green
}

if ($DateFilter) {
    Write-Host "   - Filter: Only files matching '$DateFilter'" -ForegroundColor Cyan
} else {
    Write-Host "   - Filter: All log files" -ForegroundColor Gray
}

if ($Backup) {
    Write-Host "   - Backup: Create backup of existing files" -ForegroundColor Green
}

Write-Host "   - Local path: $LocalLogsPath" -ForegroundColor Gray
Write-Host ""

# Create local directory if needed
if (!(Test-Path $LocalLogsPath)) {
    New-Item -ItemType Directory -Path $LocalLogsPath -Force | Out-Null
    Write-Host "Created local logs directory: $LocalLogsPath" -ForegroundColor Green
}

# Get Fly.io machine info
Write-Host "Getting Fly.io machine information..." -ForegroundColor Yellow
$machineList = flyctl machines list --json | ConvertFrom-Json
if (!$machineList -or $machineList.Count -eq 0) {
    Write-Host "ERROR: No running machines found. Exiting..." -ForegroundColor Red
    exit 1
}

$machineId = $machineList[0].id
Write-Host "Found machine: $machineId" -ForegroundColor Yellow

# List log files on remote server
Write-Host "Listing log files on remote server..." -ForegroundColor Yellow
# First try /logs path (production), then /app/logs (fallback)
$logFilesList = flyctl ssh console --machine $machineId -C "find /logs -name '*.jsonl' -type f"
if ([string]::IsNullOrWhiteSpace($logFilesList) -or $logFilesList -match "No such file") {
    Write-Host "   Trying /app/logs path..." -ForegroundColor Gray
    $logFilesList = flyctl ssh console --machine $machineId -C "find /app/logs -name '*.jsonl' -type f"
}

if ([string]::IsNullOrWhiteSpace($logFilesList) -or $logFilesList -match "No such file") {
    Write-Host "ERROR: No .jsonl log files found in /logs/ or /app/logs/" -ForegroundColor Red
    Write-Host "   This might be because:" -ForegroundColor Yellow
    Write-Host "   - No logs have been generated yet" -ForegroundColor Gray
    Write-Host "   - LOG_PATH environment variable not set correctly" -ForegroundColor Gray
    Write-Host "   - Persistent volume not mounted" -ForegroundColor Gray
    exit 1
}

# Filter and clean log files list - only keep actual file paths
$logFiles = @()
if (![string]::IsNullOrWhiteSpace($logFilesList)) {
    $allLines = $logFilesList -split "`n"
    foreach ($line in $allLines) {
        $cleanLine = $line.Trim()
        if ($cleanLine -ne "" -and 
            $cleanLine -notmatch "Connecting" -and 
            $cleanLine -notmatch "Error" -and
            $cleanLine -notmatch "Warning" -and
            $cleanLine -notmatch "complete" -and
            $cleanLine -notmatch "handle is invalid" -and
            $cleanLine.StartsWith("/") -and
            $cleanLine.EndsWith(".jsonl")) {
            $logFiles += $cleanLine
        }
    }
}

# Apply date filter if specified
if ($DateFilter) {
    $logFiles = $logFiles | Where-Object { $_ -like "*$DateFilter*" }
    Write-Host "Applied date filter '$DateFilter': $($logFiles.Count) files match" -ForegroundColor Cyan
}

if ($logFiles.Count -eq 0) {
    Write-Host "ERROR: No valid log files found" -ForegroundColor Red
    exit 1
}

Write-Host "Found $($logFiles.Count) log files:" -ForegroundColor Green
foreach ($file in $logFiles) {
    Write-Host "   - $(Split-Path $file -Leaf)" -ForegroundColor Gray
}

# Initialize counters
$downloadedCount = 0
$deletedCount = 0
$errorCount = 0
$totalFiles = $logFiles.Count

Write-Host ""
Write-Host "Starting download process for $totalFiles files..." -ForegroundColor Green
Write-Host ""

# Process each log file
for ($i = 0; $i -lt $logFiles.Count; $i++) {
    $logFile = $logFiles[$i]
    $fileIndex = $i + 1
    $fileName = Split-Path $logFile -Leaf
    $localPath = Join-Path $LocalLogsPath $fileName
    
    Write-Host "[$fileIndex/$totalFiles] Processing: $fileName" -ForegroundColor White
    
    # Create backup if requested
    if ($Backup -and (Test-Path $localPath)) {
        $backupPath = "$localPath.backup.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
        Copy-Item $localPath $backupPath
        Write-Host "   * Created backup: $(Split-Path $backupPath -Leaf)" -ForegroundColor Yellow
    }
    
    $tempPath = "$localPath.temp"
    
    Write-Host "   * Downloading from remote..." -ForegroundColor Cyan
    
    # Download with retry logic
    $retryCount = 0
    $maxRetries = 3
    $downloadSuccess = $false
    
    while ($retryCount -lt $maxRetries -and !$downloadSuccess) {
        if ($retryCount -gt 0) {
            Write-Host "   * Retry $retryCount/$maxRetries..." -ForegroundColor Yellow
            Start-Sleep -Seconds 2
        }
        
        flyctl ssh sftp get $logFile $tempPath --machine $machineId 2>$null
        
        if ($LASTEXITCODE -eq 0 -and (Test-Path $tempPath) -and (Get-Item $tempPath).Length -gt 0) {
            $downloadSuccess = $true
        } else {
            $retryCount++
        }
    }
    
    if ($downloadSuccess) {
        $fileSize = [math]::Round((Get-Item $tempPath).Length / 1KB, 2)
        Write-Host "   * Download successful ($fileSize KB)" -ForegroundColor Green
        
        # Handle merge/replace logic
        if ((Test-Path $localPath) -and !$NoMerge) {
            try {
                $existingContent = Get-Content $localPath -Raw -ErrorAction Stop
                $newContent = Get-Content $tempPath -Raw -ErrorAction Stop
                
                # Check if content is identical - skip if same
                if ($existingContent -eq $newContent) {
                    Write-Host "   * Content identical - skipping merge" -ForegroundColor Gray
                    Remove-Item $tempPath -Force
                } elseif ($fileName -like "*.jsonl") {
                    Write-Host "   * Merging with existing local file..." -ForegroundColor Yellow
                    
                    $existingLines = if ($existingContent) { $existingContent -split "`n" | Where-Object { $_.Trim() -ne "" } } else { @() }
                    $newLines = if ($newContent) { $newContent -split "`n" | Where-Object { $_.Trim() -ne "" } } else { @() }
                    
                    # Check if new content is subset of existing (already merged)
                    $newUniqueLines = @()
                    foreach ($newLine in $newLines) {
                        if ($existingLines -notcontains $newLine) {
                            $newUniqueLines += $newLine
                        }
                    }
                    
                    if ($newUniqueLines.Count -eq 0) {
                        Write-Host "   * All new content already exists - skipping merge" -ForegroundColor Gray
                        Remove-Item $tempPath -Force
                    } else {
                        # Merge only unique new lines
                        $allLines = @($existingLines) + @($newUniqueLines) | Sort-Object -Unique
                        $mergedContent = $allLines -join "`n"
                        
                        Set-Content $localPath $mergedContent -NoNewline -ErrorAction Stop
                        $newSize = [math]::Round((Get-Item $localPath).Length / 1KB, 2)
                        Write-Host "   * Merged $($newUniqueLines.Count) new lines (now $newSize KB)" -ForegroundColor Green
                        Remove-Item $tempPath -Force
                    }
                } else {
                    # Non-JSONL files - simple check and append
                    Write-Host "   * Appending new content..." -ForegroundColor Yellow
                    Add-Content $localPath $newContent -ErrorAction Stop
                    Write-Host "   * Appended to existing file" -ForegroundColor Green
                    Remove-Item $tempPath -Force
                }
            } catch {
                Write-Host "   * ERROR during merge: $($_.Exception.Message)" -ForegroundColor Red
                $errorCount++
                continue
            }
        } else {
            # Replace mode or no existing file
            if (Test-Path $localPath) {
                Remove-Item $localPath -Force
                Write-Host "   * Replaced existing local file" -ForegroundColor Yellow
            }
            Move-Item $tempPath $localPath
            Write-Host "   * Saved to local file" -ForegroundColor Green
        }
        
        $downloadedCount++
        
        # Delete remote file if requested
        if (!$NoDelete) {
            Write-Host "   * Deleting remote file..." -ForegroundColor Yellow
            
            $deleteRetries = 0
            $maxDeleteRetries = 2
            $deleteSuccess = $false
            
            while ($deleteRetries -lt $maxDeleteRetries -and !$deleteSuccess) {
                # Execute delete command
                $deleteOutput = flyctl ssh console --machine $machineId -C "rm -f '$logFile'" 2>&1
                
                # Give it a moment and then verify the file is gone
                Start-Sleep -Seconds 2
                
                # Check if file still exists by trying to list it
                $fileCheck = flyctl ssh console --machine $machineId -C "ls -la '$logFile'" 2>&1
                
                # If ls command fails with "No such file or directory", the file was deleted
                if ($fileCheck -match "No such file or directory" -or $fileCheck -match "cannot access") {
                    $deleteSuccess = $true
                    $deletedCount++
                    Write-Host "   * Remote file deleted successfully" -ForegroundColor Green
                } else {
                    $deleteRetries++
                    if ($deleteRetries -lt $maxDeleteRetries) {
                        Write-Host "   * Delete retry $deleteRetries/$maxDeleteRetries..." -ForegroundColor Yellow
                        Write-Host "   * Debug - File check output: $fileCheck" -ForegroundColor Gray
                        Start-Sleep -Seconds 2
                    } else {
                        Write-Host "   * Debug - Final file check output: $fileCheck" -ForegroundColor Gray
                    }
                }
            }
            
            if (!$deleteSuccess) {
                Write-Host "   * WARNING: Failed to delete remote file after $maxDeleteRetries retries" -ForegroundColor Red
            }
        } else {
            Write-Host "   * Keeping remote file (NoDelete mode)" -ForegroundColor Gray
        }
    } else {
        Write-Host "   * ERROR: Download failed after $maxRetries retries" -ForegroundColor Red
        $errorCount++
        if (Test-Path $tempPath) {
            Remove-Item $tempPath -Force
        }
    }
    
    Write-Host ""
}

# Summary
Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "DOWNLOAD SUMMARY" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

$successRate = if ($totalFiles -gt 0) { [math]::Round(($downloadedCount / $totalFiles) * 100, 1) } else { 0 }

Write-Host "Files processed: $totalFiles" -ForegroundColor White
Write-Host "Successfully downloaded: $downloadedCount ($successRate%)" -ForegroundColor Green

if (!$NoDelete) {
    Write-Host "Successfully deleted from remote: $deletedCount" -ForegroundColor Green
} else {
    Write-Host "Remote files kept (NoDelete mode)" -ForegroundColor Yellow
}

if ($errorCount -gt 0) {
    Write-Host "Errors encountered: $errorCount" -ForegroundColor Red
}

Write-Host "Local storage: $((Get-Item $LocalLogsPath).FullName)" -ForegroundColor Gray

# Show local files with details
if ($downloadedCount -gt 0) {
    Write-Host ""
    Write-Host "Local log files:" -ForegroundColor Green
    
    $localFiles = Get-ChildItem $LocalLogsPath -Filter "*.jsonl" | Sort-Object LastWriteTime -Descending
    $totalSizeKB = 0
    
    foreach ($file in $localFiles) {
        $sizeKB = [math]::Round($file.Length / 1KB, 2)
        $totalSizeKB += $sizeKB
        $lastModified = $file.LastWriteTime.ToString("yyyy-MM-dd HH:mm:ss")
        Write-Host "   * $($file.Name) ($sizeKB KB, modified: $lastModified)" -ForegroundColor Gray
    }
    
    Write-Host ""
    Write-Host "Total local storage used: $([math]::Round($totalSizeKB, 2)) KB" -ForegroundColor Cyan
    
    # Show backup files if any
    $backupFiles = Get-ChildItem $LocalLogsPath -Filter "*.backup.*" | Sort-Object LastWriteTime -Descending
    if ($backupFiles.Count -gt 0) {
        Write-Host ""
        Write-Host "Backup files created: $($backupFiles.Count)" -ForegroundColor Yellow
        foreach ($backup in $backupFiles) {
            $sizeKB = [math]::Round($backup.Length / 1KB, 2)
            Write-Host "   * $($backup.Name) ($sizeKB KB)" -ForegroundColor Gray
        }
    }
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
if ($errorCount -eq 0) {
    Write-Host "SUCCESS: Log download and cleanup completed!" -ForegroundColor Green
} else {
    Write-Host "COMPLETED: Log download finished with $errorCount errors" -ForegroundColor Yellow
}
Write-Host "============================================================" -ForegroundColor Cyan