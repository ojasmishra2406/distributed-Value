$env:PATH += ";$env:TEMP\go_portable3\go\bin"
$env:GOROOT = "$env:TEMP\go_portable3\go"
Remove-Item -Path "bench_data1","bench_data2","bench_data3" -Recurse -Force -ErrorAction SilentlyContinue
go build -o node.exe ./cmd/node
$p1 = Start-Process -NoNewWindow -PassThru -FilePath ".\node.exe" -ArgumentList "-port=50051","-dir=bench_data1"
Start-Sleep -Seconds 3
go run cmd/bench/main.go > bench.txt 2>&1
Stop-Process -Id $p1.Id -Force
