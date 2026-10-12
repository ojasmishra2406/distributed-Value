Remove-Item -Recurse -Force data1, data2, data3 -ErrorAction SilentlyContinue

Write-Host "--- BOOTING 3 NODES ---"
$p1 = Start-Process -FilePath ".\node.exe" -ArgumentList "-port=50051", "-dir=data1" -PassThru -NoNewWindow
$p2 = Start-Process -FilePath ".\node.exe" -ArgumentList "-port=50052", "-dir=data2" -PassThru -NoNewWindow
$p3 = Start-Process -FilePath ".\node.exe" -ArgumentList "-port=50053", "-dir=data3" -PassThru -NoNewWindow
Start-Sleep -Seconds 2

$s3 = Start-Process -FilePath ".\script_repair.exe" -PassThru -NoNewWindow -RedirectStandardOutput "script_repair_out.txt"
Wait-Process -Id $s3.Id

Write-Host "--- SCRIPT REPAIR OUTPUT ---"
Get-Content script_repair_out.txt

Write-Host "Stopping all nodes..."
Stop-Process -Id $p1.Id -Force
Stop-Process -Id $p2.Id -Force
Stop-Process -Id $p3.Id -Force

Write-Host "--- Checking Node 3 WAL for Read Repair ---"
if (Test-Path "data3\active.wal") {
    $bytes = [System.IO.File]::ReadAllBytes("data3\active.wal")
    $str = [System.Text.Encoding]::ASCII.GetString($bytes)
    if ($str -match "new_val") {
        Write-Host "SUCCESS: Node 3 received new_val via READ REPAIR!"
    } else {
        Write-Host "FAIL: Node 3 did not receive new_val."
    }
} else {
    Write-Host "FAIL: data3\active.wal does not exist."
}
