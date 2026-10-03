package storage
import (
	"fmt"
	"testing"
)
func TestEngine_CompactionDebug2(t *testing.T) {
	dir := t.TempDir()
	eng, _ := NewStorageEngine(dir)
	eng.Put([]byte("k1"), []byte("v1"), 1, nil)
	eng.triggerFlush()
	eng.Put([]byte("k2"), []byte("v2"), 2, nil)
	eng.triggerFlush()
	eng.Put([]byte("k3"), []byte("v3"), 3, nil)
	eng.triggerFlush()
	eng.Put([]byte("k4"), []byte("v4"), 4, nil)
	eng.triggerFlush()
	
	// manually run compaction
	err := eng.Compact()
	fmt.Printf("Compact returned: %v\n", err)
	fmt.Printf("Num SSTables: %d\n", len(eng.sstables))
}
