import sys

with open('internal/storage/engine.go', 'r') as f:
    content = f.read()

content = content.replace('make(chan PutReq, 1000)', 'make(chan PutReq, 10000)')
content = content.replace('time.NewTimer(50 * time.Millisecond)', 'time.NewTimer(10 * time.Millisecond)')
content = content.replace('timer.Reset(50 * time.Millisecond)', 'timer.Reset(10 * time.Millisecond)')
# Revert any batch size 
content = content.replace('if len(batch) >= 5000 {', 'if len(batch) >= 100 {')
content = content.replace('if len(batch) >= 2000 {', 'if len(batch) >= 100 {')
content = content.replace('if len(batch) >= 10000 {', 'if len(batch) >= 100 {')

with open('internal/storage/engine.go', 'w') as f:
    f.write(content)
