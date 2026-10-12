import sys
import re

with open('internal/storage/engine.go', 'r') as f:
    content = f.read()

content = re.sub(r'time\.NewTimer\(\d+ \* time\.Millisecond\)', 'time.NewTimer(10 * time.Millisecond)', content)
content = re.sub(r'timer\.Reset\(\d+ \* time\.Millisecond\)', 'timer.Reset(10 * time.Millisecond)', content)
content = re.sub(r'if len\(batch\) >= \d+ \{', 'if len(batch) >= 100 {', content)

with open('internal/storage/engine.go', 'w') as f:
    f.write(content)
