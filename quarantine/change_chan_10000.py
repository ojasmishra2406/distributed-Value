import sys

with open('internal/storage/engine.go', 'r') as f:
    content = f.read()

content = content.replace('make(chan PutReq, 1000)', 'make(chan PutReq, 10000)')

with open('internal/storage/engine.go', 'w') as f:
    f.write(content)
