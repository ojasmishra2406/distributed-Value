import sys
import re

with open('internal/storage/flat_vector.go', 'r') as f:
    content = f.read()

replacement = """func (h *FlatVectorIndex) InsertBatch(nodes []*VectorNode) {
	h.mu.Lock()
	defer h.mu.Unlock()

	totalSize := 0
	for _, node := range nodes {
		h.nodes[node.ID] = &VectorNode{
			ID:     node.ID,
			Key:    append([]byte(nil), node.Key...),
			Value:  append([]byte(nil), node.Value...),
			Vector: append([]float32(nil), node.Vector...),
		}
		
		totalSize += 4 + 4 + 4 + 4 // lengths
		totalSize += len(node.ID) + len(node.Key) + len(node.Value) + (len(node.Vector) * 4)
	}

	if h.file != nil {
		buf := make([]byte, 0, totalSize)
		temp := make([]byte, 4)
		for _, node := range nodes {
			binary.LittleEndian.PutUint32(temp, uint32(len(node.ID)))
			buf = append(buf, temp...)
			binary.LittleEndian.PutUint32(temp, uint32(len(node.Key)))
			buf = append(buf, temp...)
			binary.LittleEndian.PutUint32(temp, uint32(len(node.Value)))
			buf = append(buf, temp...)
			binary.LittleEndian.PutUint32(temp, uint32(len(node.Vector)))
			buf = append(buf, temp...)

			buf = append(buf, []byte(node.ID)...)
			buf = append(buf, node.Key...)
			buf = append(buf, node.Value...)

			for _, v := range node.Vector {
				binary.LittleEndian.PutUint32(temp, math.Float32bits(v))
				buf = append(buf, temp...)
			}
		}
		h.file.Write(buf)
		h.file.Sync()
	}
}"""

# We need to make sure `math` package is imported. Let's do a naive replace for import if needed.
if '"math"' not in content:
    content = content.replace('import (', 'import (\n\t"math"\n', 1)

content = re.sub(r'func \(h \*FlatVectorIndex\) InsertBatch\(nodes \[\]\*VectorNode\) \{.*?\n\}\n', replacement + "\n", content, flags=re.DOTALL)

with open('internal/storage/flat_vector.go', 'w') as f:
    f.write(content)
