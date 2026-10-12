import sys
import re

with open('internal/storage/engine.go', 'r') as f:
    content = f.read()

content = re.sub(r'if len\(batch\) >= \d+ \{', 'if len(batch) >= 5000 {', content)

with open('internal/storage/engine.go', 'w') as f:
    f.write(content)
