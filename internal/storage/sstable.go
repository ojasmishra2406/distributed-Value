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
	dataEnd      int64
}

type IndexEntry struct {
	Key    []byte
	Offset int64
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

	f.Seek(bOff, io.SeekStart)
	bloomData := make([]byte, iOff-bOff)
	io.ReadFull(f, bloomData)
	bloom := &BloomFilter{}
	bloom.bitset = make([]byte, binary.LittleEndian.Uint32(bloomData[0:4]))
	bloom.m = binary.LittleEndian.Uint32(bloomData[4:8])
	bloom.k = binary.LittleEndian.Uint32(bloomData[8:12])
	copy(bloom.bitset, bloomData[12:])

	f.Seek(iOff, io.SeekStart)
	indexData := make([]byte, stat.Size()-FooterSize-iOff)
	io.ReadFull(f, indexData)

	var sparseIndex []IndexEntry
	buf := bytes.NewReader(indexData)
	for buf.Len() > 0 {
		var kLen uint32
		binary.Read(buf, binary.LittleEndian, &kLen)
		k := make([]byte, kLen)
		buf.Read(k)
		var offset int64
		binary.Read(buf, binary.LittleEndian, &offset)
		sparseIndex = append(sparseIndex, IndexEntry{Key: k, Offset: offset})
	}

	return &SSTable{
		file:         f,
		bloom:        bloom,
		sparseIndex:  sparseIndex,
		minTimestamp: minTs,
		maxTimestamp: maxTs,
		dataEnd:      bOff,
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
		curr, _ := s.file.Seek(0, io.SeekCurrent)
		if curr >= s.dataEnd { break }
		
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
			break
		} else if cmp == 0 {
			v := make([]byte, vLen)
			io.ReadFull(s.file, v)
			ts := int64(binary.LittleEndian.Uint64(meta[0:8]))
			tomb := meta[8] == 1
			return v, tomb, ts, true, nil
		}

		s.file.Seek(int64(vLen), io.SeekCurrent)
	}

	return nil, false, 0, false, nil
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

	indexOffset := bytesWritten + int64(12+len(bf.bitset))
	for _, entry := range sparseIndex {
		binary.Write(f, binary.LittleEndian, uint32(len(entry.Key)))
		f.Write(entry.Key)
		binary.Write(f, binary.LittleEndian, entry.Offset)
	}

	footer := make([]byte, FooterSize)
	binary.LittleEndian.PutUint64(footer[0:8], uint64(bloomOffset))
	binary.LittleEndian.PutUint64(footer[8:16], uint64(indexOffset))
	binary.LittleEndian.PutUint64(footer[16:24], uint64(minTs))
	binary.LittleEndian.PutUint64(footer[24:32], uint64(maxTs))
	binary.LittleEndian.PutUint32(footer[32:36], MagicNumber)

	f.Write(footer)
	f.Sync()

	return &SSTable{
		file:         f,
		bloom:        bf,
		sparseIndex:  sparseIndex,
		minTimestamp: minTs,
		maxTimestamp: maxTs,
		dataEnd:      bloomOffset,
	}, nil
}

func (s *SSTable) ScanSince(sinceTs int64) ([][]byte, error) {
	if s.maxTimestamp < sinceTs {
		return nil, nil // completely skip this SSTable
	}

	var results [][]byte
	s.file.Seek(0, io.SeekStart)
	meta := make([]byte, 17)

	for {
		curr, _ := s.file.Seek(0, io.SeekCurrent)
		if curr >= s.dataEnd { break }
		
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
