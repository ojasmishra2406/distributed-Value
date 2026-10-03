import sys

with open('internal/storage/sstable.go', 'r') as f:
    lines = f.readlines()

new_lines = []
for line in lines:
    if 'type SSTable struct {' in line:
        new_lines.append(line)
        new_lines.append('\tdataEnd int64\n')
    elif 'minTs := int64(binary.LittleEndian.Uint64(footerData[16:24]))' in line:
        new_lines.append(line)
        new_lines.append('\tdataEnd := bOff\n')
    elif 'return &SSTable{' in line:
        if 'bloomOffset' in ''.join(lines): # very hacky but wait
            pass
        new_lines.append(line)
    elif 'file: f,' in line:
        new_lines.append(line)
    elif 'maxTimestamp: maxTs,' in line:
        new_lines.append(line)
        if 'dataEnd := bOff' in ''.join(new_lines):
            # need a way to distinguish OpenSSTable and Flush
            pass
        
