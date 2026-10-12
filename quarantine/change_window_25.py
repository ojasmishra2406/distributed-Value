import sys
import re

with open('internal/storage/engine.go', 'r') as f:
    content = f.read()

content = re.sub(r'time\.NewTimer\(\d+ \* time\.Millisecond\)', 'time.NewTimer(25 * time.Millisecond)', content)
content = re.sub(r'timer\.Reset\(\d+ \* time\.Millisecond\)', 'timer.Reset(25 * time.Millisecond)', content)

with open('internal/storage/engine.go', 'w') as f:
    f.write(content)
