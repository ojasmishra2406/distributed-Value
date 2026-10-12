import sys

with open('internal/storage/engine.go', 'r') as f:
    content = f.read()

content = content.replace('if len(batch) >= 100 {', 'if len(batch) >= 1000 {')

with open('internal/storage/engine.go', 'w') as f:
    f.write(content)
