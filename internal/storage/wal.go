package storage
import (
	"encoding/binary"
	"hash/crc32"
	
	"os"
	"sync"
)
type WAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}
func OpenWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil { return nil, err }
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
func (w *WAL) Recover(mem *Memtable) error { return nil }
func (w *WAL) CloseAndRemove() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.file.Close()
	return os.Remove(w.path)
}
