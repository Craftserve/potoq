package utils

// Ogolnie da sie to samo zaimplementowac uzywajac math/big.Int, ale nie wiem ktore bedzie szybsze
// a na pewno ta implementacja jest znacznie prostsza - przydalby sie jakis benchmark

type Bitarray []uint64

func NewBitarray(bits int) Bitarray {
	if bits%64 != 0 {
		panic("bits % 64 != 0")
	}
	return Bitarray(make([]uint64, bits/64))
}

func (b Bitarray) Get(i int) bool {
	if i/64 > len(b) {
		panic(i)
	}
	return b[i/64]&(1<<(uint64(i)%64)) != 0
}

func (b Bitarray) Set(i int, v bool) {
	if i/64 > len(b) {
		panic(i)
	}
	var bitSetVar uint64
	if v {
		bitSetVar = 1
	}
	b[i/64] ^= (-bitSetVar ^ b[i/64]) & (1 << uint64(i%64))
}
