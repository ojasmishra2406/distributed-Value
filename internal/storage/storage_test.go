package storage
import (
	
	"os"
	"path/filepath"
	"testing"
	"time"
)
func TestMemtable_Operations(t *testing.T) {
	mem := NewMemtable()
	mem.Put([]byte("key1"), []byte("val1"), 100)
	mem.Put([]byte("key3"), []byte("val3"), 102)
	mem.Put([]byte("key2"), []byte("val2"), 101)
	val, _, _, found := mem.Get([]byte("key2"))
	if !found || string(val) != "val2" { t.Fatalf("Expected val2, got %s", string(val)) }
	mem.Delete([]byte("key2"), 103)
	_, tomb, _, found := mem.Get([]byte("key2"))
	if !found || !tomb { t.Fatalf("Expected tombstone") }
}
func TestWAL_RecoveryAndCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")
	wal, _ := OpenWAL(path)
	wal.Append([]byte("k1"), []byte("v1"), 10, false)
	wal.Close()
	
	mem := NewMemtable()
	wal2, _ := OpenWAL(path)
	wal2.Recover(mem)
	val, _, _, found := mem.Get([]byte("k1"))
	if !found || string(val) != "v1" { t.Fatalf("Failed to recover WAL") }
	wal2.Close()
	
	// Corrupt file
	f, _ := os.OpenFile(path, os.O_RDWR, 0644)
	f.WriteAt([]byte("junk"), 0)
	f.Close()
	
	mem2 := NewMemtable()
	wal3, _ := OpenWAL(path)
	err := wal3.Recover(mem2) // should truncate best effort
	if err != nil { t.Logf("Expected corruption handled: %v", err) }
	wal3.Close()
}
func TestSSTable_ReadWrite(t *testing.T) {
	dir := t.TempDir()
	mem := NewMemtable()
	mem.Put([]byte("k1"), []byte("v1"), 10)
	sst, err := FlushMemtableToSSTable(mem, dir, 1)
	defer sst.file.Close()
	if err != nil { t.Fatal(err) }
	val, tomb, _, found, _ := sst.Get([]byte("k1"))
	if !found || tomb || string(val) != "v1" { t.Fatalf("SSTable read failed") }
}
func TestBloomFilter_FalsePositiveRate(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)
	for i := 0; i < 1000; i++ {
		bf.Add([]byte(string(rune(i))))
	}
	if !bf.MightContain([]byte(string(rune(500)))) { t.Fatalf("Should contain 500") }
	falsePositives := 0
	for i := 1000; i < 2000; i++ {
		if bf.MightContain([]byte(string(rune(i)))) { falsePositives++ }
	}
	fpr := float64(falsePositives) / 1000.0
	if fpr > 0.05 { t.Fatalf("FPR too high: %v", fpr) }
}
func TestEngine_Compaction(t *testing.T) {
	dir := t.TempDir()
	eng, _ := NewStorageEngine(dir)
	defer func() { eng.activeWAL.file.Close(); for _, s := range eng.sstables { s.file.Close() } }()
	eng.Put([]byte("k1"), []byte("v1"), 1, nil)
	eng.triggerFlush()
	time.Sleep(10*time.Millisecond)
	eng.Put([]byte("k2"), []byte("v2"), 2, nil)
	eng.triggerFlush()
	time.Sleep(10*time.Millisecond)
	eng.Put([]byte("k3"), []byte("v3"), 3, nil)
	eng.triggerFlush()
	time.Sleep(10*time.Millisecond)
	eng.Put([]byte("k4"), []byte("v4"), 4, nil)
	eng.triggerFlush()
	time.Sleep(50*time.Millisecond) // Let compaction finish
	val, _, _, found, _ := eng.Get([]byte("k1"))
	if !found || string(val) != "v1" { t.Fatalf("Compaction lost data") }
}
