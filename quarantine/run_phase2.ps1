Remove-Item -Recurse -Force data1, data2, data3 -ErrorAction SilentlyContinue

Write-Host "--- BOOTING 3 NODES ---"
$p1 = Start-Process -FilePath ".\node.exe" -ArgumentList "-port=50051", "-dir=data1" -PassThru -NoNewWindow
$p2 = Start-Process -FilePath ".\node.exe" -ArgumentList "-port=50052", "-dir=data2" -PassThru -NoNewWindow
$p3 = Start-Process -FilePath ".\node.exe" -ArgumentList "-port=50053", "-dir=data3" -PassThru -NoNewWindow

Start-Sleep -Seconds 2
Write-Host "Nodes running."

$s2 = Start-Process -FilePath ".\script2.exe" -PassThru -NoNewWindow -RedirectStandardOutput "script2_out.txt"

Start-Sleep -Seconds 1
Write-Host "Killing Node 3..."
Stop-Process -Id $p3.Id -Force

Start-Sleep -Seconds 4
Write-Host "Restarting Node 3..."
$p3_new = Start-Process -FilePath ".\node.exe" -ArgumentList "-port=50053", "-dir=data3" -PassThru -NoNewWindow

Start-Sleep -Seconds 6
Write-Host "--- SCRIPT 2 OUTPUT ---"
Get-Content script2_out.txt

Write-Host "Stopping all nodes..."
Stop-Process -Id $p1.Id -Force
Stop-Process -Id $p2.Id -Force
Stop-Process -Id $p3_new.Id -Force

Write-Host "--- Checking Node 3 WAL for hinted handoff replay ---"
if (Test-Path "data3\active.wal") {
    $bytes = [System.IO.File]::ReadAllBytes("data3\active.wal")
    $str = [System.Text.Encoding]::ASCII.GetString($bytes)
    if ($str -match "key_down") {
        Write-Host "SUCCESS: Node 3 received key_down via hinted handoff!"
    } else {
        Write-Host "FAIL: Node 3 did not receive key_down."
    }
} else {
    Write-Host "FAIL: data3\active.wal does not exist."
}
