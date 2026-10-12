import sys
import re

with open('internal/storage/wal.go', 'r') as f:
    content = f.read()

replacement = """func (w *WAL) AppendBatch(entries []Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	totalSize := 0
	for _, entry := range entries {
		totalSize += 4 + 17 + len(entry.Key) + len(entry.Value)
	}

	batchPayload := make([]byte, 0, totalSize)
	for _, entry := range entries {
		kLen := uint32(len(entry.Key))
		vLen := uint32(len(entry.Value))
		tombByte := byte(0)
		if entry.Tombstone { tombByte = 1 }

		payload := make([]byte, 17+kLen+vLen)
		binary.LittleEndian.PutUint64(payload[0:8], uint64(entry.Timestamp))
		payload[8] = tombByte
		binary.LittleEndian.PutUint32(payload[9:13], kLen)
		binary.LittleEndian.PutUint32(payload[13:17], vLen)
		copy(payload[17:], entry.Key)
		copy(payload[17+kLen:], entry.Value)

		checksum := crc32.ChecksumIEEE(payload)
		header := make([]byte, 4)
		binary.LittleEndian.PutUint32(header, checksum)

		batchPayload = append(batchPayload, header...)
		batchPayload = append(batchPayload, payload...)
	}

	if _, err := w.file.Write(batchPayload); err != nil {
		return err
	}
	return w.file.Sync()
}"""

content = re.sub(r'func \(w \*WAL\) AppendBatch\(entries \[\]Entry\) error \{.*?\n\}\n', replacement + "\n", content, flags=re.DOTALL)

with open('internal/storage/wal.go', 'w') as f:
    f.write(content)
