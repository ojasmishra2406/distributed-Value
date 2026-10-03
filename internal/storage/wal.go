package storage

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

type WAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func OpenWAL(path string) (*WAL, error) {
	// Removing O_APPEND so we can truncate on windows if needed
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil { return nil, err }
	f.Seek(0, io.SeekEnd)
	return &WAL{file: f, path: path}, nil
}

func (w *WAL) Append(key, value []byte, timestamp int64, tombstone bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	kLen := uint32(len(key))
	vLen := uint32(len(value))
	tombByte := byte(0)
	if tombstone { tombByte = 1 }

	payload := make([]byte, 17+kLen+vLen)
	binary.LittleEndian.PutUint64(payload[0:8], uint64(timestamp))
	payload[8] = tombByte
	binary.LittleEndian.PutUint32(payload[9:13], kLen)
	binary.LittleEndian.PutUint32(payload[13:17], vLen)
	copy(payload[17:], key)
	copy(payload[17+kLen:], value)

	checksum := crc32.ChecksumIEEE(payload)
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, checksum)

	if _, err := w.file.Write(header); err != nil { return err }
	if _, err := w.file.Write(payload); err != nil { return err }
	return w.file.Sync()
}

func (w *WAL) Recover(mem *Memtable) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil { return err }

	validPos := int64(0)
	header := make([]byte, 4)
	meta := make([]byte, 17)

	for {
		if _, err := io.ReadFull(w.file, header); err != nil {
			if err == io.EOF { break }
			break // Corrupted tail
		}
		
		if _, err := io.ReadFull(w.file, meta); err != nil { break }

		kLen := binary.LittleEndian.Uint32(meta[9:13])
		vLen := binary.LittleEndian.Uint32(meta[13:17])

		data := make([]byte, kLen+vLen)
		if _, err := io.ReadFull(w.file, data); err != nil { break }

		payload := append(meta, data...)
		expectedCrc := crc32.ChecksumIEEE(payload)
		actualCrc := binary.LittleEndian.Uint32(header)

		if expectedCrc != actualCrc { break } // CRC mismatch

		timestamp := int64(binary.LittleEndian.Uint64(meta[0:8]))
		tombstone := meta[8] == 1
		key := data[:kLen]
		value := data[kLen:]

		mem.Put(key, value, timestamp)
		if tombstone {
			mem.Delete(key, timestamp)
		}
		validPos += int64(4 + 17 + kLen + vLen)
	}

	w.file.Truncate(validPos) // Best effort cleanup
	_, err := w.file.Seek(validPos, io.SeekStart)
	return err
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func (w *WAL) CloseAndRemove() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.file.Close()
	return os.Remove(w.path)
}
