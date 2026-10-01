package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

const FooterSize = 36
const MagicNumber = uint32(0xDEADC0DE)
const IndexInterval = 4096

type SSTable struct {
	file         *os.File
	bloom        *BloomFilter
	sparseIndex  []IndexEntry
	minTimestamp int64
	maxTimestamp int64
}

type IndexEntry struct {
	Key    []byte
	Offset int64
}

type SSTableFooter struct {
	BloomOffset  int64
	IndexOffset  int64
	MinTimestamp int64
	MaxTimestamp int64
	Magic        uint32
}

func FlushMemtableToSSTable(mem *Memtable, dir string, id uint64) (*SSTable, error) {
	path := filepath.Join(dir, fmt.Sprintf("%d.sst", id))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	it := mem.Iterator()
	bf := NewBloomFilter(int(mem.Size()/100)+1000, 0.01)
	
	var sparseIndex []IndexEntry
	var bytesWritten int64
	lastIndexOffset := int64(-1)

	minTs := int64(math.MaxInt64)
	maxTs := int64(math.MinInt64)

	for it.HasNext() {
		k, v, ts, tb := it.Next()
		bf.Add(k)

		if ts < minTs { minTs = ts }
		if ts > maxTs { maxTs = ts }

		if bytesWritten-lastIndexOffset >= IndexInterval || lastIndexOffset == -1 {
			sparseIndex = append(sparseIndex, IndexEntry{Key: append([]byte(nil), k...), Offset: bytesWritten})
			lastIndexOffset = bytesWritten
		}

		kLen := uint32(len(k))
		vLen := uint32(len(v))
		tByte := byte(0)
		if tb {
			tByte = 1
		}

		meta := make([]byte, 17)
		binary.LittleEndian.PutUint64(meta[0:8], uint64(ts))
		meta[8] = tByte
		binary.LittleEndian.PutUint32(meta[9:13], kLen)
		binary.LittleEndian.PutUint32(meta[13:17], vLen)

		f.Write(meta)
		f.Write(k)
		f.Write(v)
		bytesWritten += int64(17 + kLen + vLen)
	}

	bloomOffset := bytesWritten
	binary.Write(f, binary.LittleEndian, uint32(len(bf.bitset)))
	binary.Write(f, binary.LittleEndian, bf.m)
	binary.Write(f, binary.LittleEndian, bf.k)
	f.Write(bf.bitset)
	bytesWritten += int64(12 + len(bf.bitset))

	indexOffset := bytesWritten
	binary.Write(f, binary.LittleEndian, uint32(len(sparseIndex)))
	for _, entry := range sparseIndex {
		binary.Write(f, binary.LittleEndian, uint32(len(entry.Key)))
		f.Write(entry.Key)
		binary.Write(f, binary.LittleEndian, entry.Offset)
	}

	// CRITICAL PATCH: Footer with Min/Max TS
	footer := make([]byte, FooterSize)
	binary.LittleEndian.PutUint64(footer[0:8], uint64(bloomOffset))
	binary.LittleEndian.PutUint64(footer[8:16], uint64(indexOffset))
	binary.LittleEndian.PutUint64(footer[16:24], uint64(minTs))
	binary.LittleEndian.PutUint64(footer[24:32], uint64(maxTs))
	binary.LittleEndian.PutUint32(footer[32:36], MagicNumber)
	f.Write(footer)

	f.Sync()
	f.Seek(0, io.SeekStart)

	return &SSTable{
		file:         f,
		bloom:        bf,
		sparseIndex:  sparseIndex,
		minTimestamp: minTs,
		maxTimestamp: maxTs,
	}, nil
}

func OpenSSTable(path string) (*SSTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	stat, _ := f.Stat()
	f.Seek(stat.Size()-FooterSize, io.SeekStart)
	footerData := make([]byte, FooterSize)
	io.ReadFull(f, footerData)

	if binary.LittleEndian.Uint32(footerData[32:36]) != MagicNumber {
		return nil, errors.New("corrupt sstable: bad magic number")
	}

	bOff := int64(binary.LittleEndian.Uint64(footerData[0:8]))
	iOff := int64(binary.LittleEndian.Uint64(footerData[8:16]))
	minTs := int64(binary.LittleEndian.Uint64(footerData[16:24]))
	maxTs := int64(binary.LittleEndian.Uint64(footerData[24:32]))

	// Load Bloom
	f.Seek(bOff, io.SeekStart)
	var bitsetLen, m, k uint32
	binary.Read(f, binary.LittleEndian, &bitsetLen)
	binary.Read(f, binary.LittleEndian, &m)
	binary.Read(f, binary.LittleEndian, &k)
	bitset := make([]byte, bitsetLen)
	io.ReadFull(f, bitset)
	bf := &BloomFilter{bitset: bitset, m: m, k: k}

	// Load Index
	f.Seek(iOff, io.SeekStart)
	var numEntries uint32
	binary.Read(f, binary.LittleEndian, &numEntries)
	sparseIndex := make([]IndexEntry, numEntries)
	for i := uint32(0); i < numEntries; i++ {
		var kLen uint32
		binary.Read(f, binary.LittleEndian, &kLen)
		key := make([]byte, kLen)
		io.ReadFull(f, key)
		var offset int64
		binary.Read(f, binary.LittleEndian, &offset)
		sparseIndex[i] = IndexEntry{Key: key, Offset: offset}
	}

	return &SSTable{
		file:         f,
		bloom:        bf,
		sparseIndex:  sparseIndex,
		minTimestamp: minTs,
		maxTimestamp: maxTs,
	}, nil
}

func (s *SSTable) Get(key []byte) ([]byte, bool, int64, bool, error) {
	if !s.bloom.MightContain(key) {
		return nil, false, 0, false, nil
	}

	var startOffset int64 = 0
	for i := len(s.sparseIndex) - 1; i >= 0; i-- {
		if bytes.Compare(s.sparseIndex[i].Key, key) <= 0 {
			startOffset = s.sparseIndex[i].Offset
			break
		}
	}

	s.file.Seek(startOffset, io.SeekStart)
	meta := make([]byte, 17)

	for {
		if _, err := io.ReadFull(s.file, meta); err != nil {
			if err == io.EOF { break }
			return nil, false, 0, false, err
		}

		kLen := binary.LittleEndian.Uint32(meta[9:13])
		vLen := binary.LittleEndian.Uint32(meta[13:17])

		k := make([]byte, kLen)
		io.ReadFull(s.file, k)

		cmp := bytes.Compare(k, key)
		if cmp > 0 {
			break // Passed the key
		}

		if cmp == 0 {
			v := make([]byte, vLen)
			io.ReadFull(s.file, v)
			ts := int64(binary.LittleEndian.Uint64(meta[0:8]))
			tb := meta[8] == 1
			return v, tb, ts, true, nil
		}
		s.file.Seek(int64(vLen), io.SeekCurrent)
	}

	return nil, false, 0, false, nil
}

// ScanSince uses the min/max timestamp footers to optimize scanning.
func (s *SSTable) ScanSince(sinceTs int64) ([][]byte, error) {
	if s.maxTimestamp < sinceTs {
		return nil, nil // completely skip this SSTable
	}

	var results [][]byte
	s.file.Seek(0, io.SeekStart)
	meta := make([]byte, 17)

	for {
		if _, err := io.ReadFull(s.file, meta); err != nil {
			if err == io.EOF { break }
			return nil, err
		}

		ts := int64(binary.LittleEndian.Uint64(meta[0:8]))
		kLen := binary.LittleEndian.Uint32(meta[9:13])
		vLen := binary.LittleEndian.Uint32(meta[13:17])

		if ts >= sinceTs {
			k := make([]byte, kLen)
			io.ReadFull(s.file, k)
			results = append(results, append([]byte(nil), k...))
			s.file.Seek(int64(vLen), io.SeekCurrent)
		} else {
			s.file.Seek(int64(kLen+vLen), io.SeekCurrent)
		}
	}
	return results, nil
}
