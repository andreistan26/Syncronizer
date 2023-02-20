package checksum

import (
	"errors"
	"fmt"
)

type CheckSum uint32

const (
	CHUNK_SIZE     = 4096
	SW_BUFFER_SIZE = CHUNK_SIZE * 4
	MOD2_16        = 1 << 16
)

var (
	ErrSWStuck   = errors.New("No more bytes to slide!")
	ErrSWSize    = errors.New("Sliding window size is not equal to CHUNK_SIZE!")
	ErrSWSizeRem = errors.New("Sliding window is stuck but still has data to be read!")
)

type Chunk struct {
	checkSum   CheckSum
	strongHash [16]byte

	offset uint64
	size   uint64
	index  uint64
}

type SlidingWindow struct {
	checkSum CheckSum

	buffer    [CHUNK_SIZE * 4]byte
	readBytes uint64
	cap       uint64

	k_idx uint64
	l_idx uint64
	a_sum uint32
	b_sum uint32
}

func (sw *SlidingWindow) GetBuffer() []byte {
	if (sw.l_idx - sw.k_idx + 1) != CHUNK_SIZE {
		panic(ErrSWSize)
	}
	return sw.buffer[sw.k_idx : sw.l_idx+1]
}

// adler-32 hash
// Source : https://rsync.samba.org/tech_report/node3.html
// x represents the value of the byte from the window
// a(k, l) = (sum i=k->l : x_i) % MOD2_16
// b(k, l) = (sum i=k->l : (l-i+1) * x_i ) % MOD2_16
// checkSum = a(k, l) + MOD2_16 * b(k, l)
func NewCheckSum(bytes []byte) (sum CheckSum, a_sum, b_sum uint32) {
	signExtend := func(b byte) uint32 {
		return uint32(int32(uint32(b)<<24) >> 24)
	}

	chunk_len := len(bytes)

	for idx, el := range bytes {
		a_sum += signExtend(el)
		b_sum += uint32(chunk_len-idx) * signExtend(el)
	}

	//a_sum %= MOD2_16
	//b_sum %= MOD2_16

	sum = CheckSum(a_sum)&0xffff | CheckSum(b_sum)<<16

	return sum, a_sum, b_sum
}

func (sw *SlidingWindow) checkStuck(respType ResponseType) (err error) {
	offset_map := map[ResponseType]uint64{
		A_BLOCK: 1,
		B_BLOCK: CHUNK_SIZE,
	}

	if sw.cap <= (sw.l_idx + offset_map[respType]) {
		return ErrSWSize
	}
	return nil
}

func (sw *SlidingWindow) rollChunk() error {
	if sw.checkStuck(B_BLOCK) == ErrSWSize {
		return ErrSWSizeRem
	}
	sw.k_idx += CHUNK_SIZE
	sw.l_idx += CHUNK_SIZE
	sw.checkSum, sw.a_sum, sw.b_sum = NewCheckSum(sw.GetBuffer())
	return nil
}

func (sw *SlidingWindow) roll() error {
	if sw.checkStuck(A_BLOCK) == ErrSWSize {
		sw.l_idx = sw.k_idx
		return ErrSWSizeRem
	}

	signExtend := func(b byte) uint32 {
		return uint32(int32(uint32(b)<<24) >> 24)
	}

	sw.k_idx++
	sw.l_idx++

	sw.a_sum = (sw.a_sum - signExtend(sw.buffer[sw.k_idx-1]) +
		signExtend(sw.buffer[sw.l_idx])) //% MOD2_16
	sw.b_sum = (sw.b_sum - CHUNK_SIZE*signExtend(sw.buffer[sw.k_idx-1]) +
		sw.a_sum) //% MOD2_16
	sw.checkSum = CheckSum(sw.a_sum)&0xffff | CheckSum(sw.b_sum)<<16

	return nil
}

func (chunk Chunk) String() string {
	chunkStr := fmt.Sprintf(
		"checksum : %v \n "+
			"md5 hash : %v \n "+
			"offset   : %v \n "+
			"size     : %v \n ",
		chunk.checkSum, chunk.strongHash,
		chunk.offset, chunk.size,
	)
	return chunkStr
}

func (sw SlidingWindow) String() string {
	return fmt.Sprintf(
		"checksum : %v \n "+
			"buffer   : %v \n "+
			"readB    : %v \n "+
			"k_idx    : %v \n "+
			"l_idx    : %v \n "+
			"a_sum    : %v \n "+
			"b_sum    : %v \n ",
		sw.checkSum, sw.buffer, sw.readBytes,
		sw.k_idx, sw.l_idx, sw.a_sum, sw.b_sum,
	)
}
