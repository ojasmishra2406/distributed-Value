package storage

import (
	"hash/fnv"
	"math"
)

type BloomFilter struct {
	bitset []byte
	m      uint32
	k      uint32
}

func NewBloomFilter(expectedElements int, falsePositiveRate float64) *BloomFilter {
	m := uint32(math.Ceil(float64(expectedElements) * math.Log(falsePositiveRate) / math.Log(1.0/math.Pow(2.0, math.Ln2))))
	k := uint32(math.Round(float64(m) / float64(expectedElements) * math.Ln2))
	if k == 0 {
		k = 1
	}
	return &BloomFilter{
		bitset: make([]byte, (m+7)/8),
		m:      m,
		k:      k,
	}
}

func (bf *BloomFilter) Add(data []byte) {
	h1, h2 := hash(data)
	for i := uint32(0); i < bf.k; i++ {
		idx := (h1 + i*h2) % bf.m
		bf.bitset[idx/8] |= (1 << (idx % 8))
	}
}

func (bf *BloomFilter) MightContain(data []byte) bool {
	h1, h2 := hash(data)
	for i := uint32(0); i < bf.k; i++ {
		idx := (h1 + i*h2) % bf.m
		if bf.bitset[idx/8]&(1<<(idx%8)) == 0 {
			return false
		}
	}
	return true
}

func hash(data []byte) (uint32, uint32) {
	h := fnv.New64a()
	h.Write(data)
	sum := h.Sum64()
	return uint32(sum >> 32), uint32(sum)
}
