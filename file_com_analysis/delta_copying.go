package delta_copying

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
)

type RemoteFile struct {
	file        *os.File
	fileSize    uint64

	chunkList   []Chunk
	chunkCount  uint64
}

// for the strongHash we are using MD5
// rsync is based on MD4
type Chunk struct {
	checkSum    CheckSum
	strongHash  [16]byte

	offset      uint64
	size        uint64
}

type SourceFile struct {
    file        *os.File
    fileSize    uint64
    slidingWin  SlidingWindow
}

// Source : https://rsync.samba.org/tech_report/node3.html
// x represents the value of the byte from the window
// a(k, l) = (sum i=k->l : x_i) % MOD2_16
// b(k, l) = (sum i=k->l : (l-i+1) * x_i ) % MOD2_16
// checkSum = a(k, l) + MOD2_16 * b(k, l)
type SlidingWindow struct {
    checkSum    CheckSum

    buffer      [CHUNK_SIZE * 4]byte
    readBytes   uint64

    k_idx       uint32
    l_idx       uint32
    a_sum       uint64
    b_sum       uint64
}

type RsyncExchange struct {
    sourceFile  *SourceFile
    chunkList   []Chunk
    hashMap     HashMap
}

type HashMap map[CheckSumHash] []*Chunk

type CheckSum uint64

type CheckSumHash uint32

// Testable Parameters
const (
	CHUNK_SIZE          = 4096
	SW_BUFFER_SIZE      = CHUNK_SIZE * 4
    MOD2_16             = 1 << 16
)

// Errors
var( 
    ErrNoFile   = errors.New("No file bound to the type!")
    ErrSWStuck  = errors.New("No more bytes to slide!")
)

func NewCheckSum(bytes []byte) (sum CheckSum) {
	b_sum := 0

	chunk_len := len(bytes)

	for idx, el := range bytes {
		sum += CheckSum(el)
		b_sum += (chunk_len - idx) * int(el)
	}

	sum %= MOD2_16
	sum += CheckSum((b_sum % MOD2_16)) * MOD2_16

	return sum
}

func CreateRemoteFile(filePath string) (RemoteFile) {
	var rf RemoteFile
    var err error

	rf.file, err = os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer rf.file.Close()

	r := bufio.NewReader(rf.file)
	for ; ; rf.chunkCount++ {
		buf := make([]byte, CHUNK_SIZE)

		n, err := r.Read(buf)

		if n == 0 {

			if err != nil {
				panic(err)
			}

			if err == io.EOF {
				break
			}
		}

		rf.chunkList = append(rf.chunkList, Chunk{
			NewCheckSum(buf),
			md5.Sum(buf),
			rf.chunkCount * CHUNK_SIZE,
			uint64(n),
		})
	}

	return rf
}

func CreateSourceFile(filePath string) (SourceFile) {
	var sf SourceFile
	var err error

	sf.file, err = os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer sf.file.Close()

    stats, _ := sf.file.Stat()
    sf.fileSize = uint64(stats.Size())

	r := bufio.NewReader(sf.file)
    n, err := r.Read(sf.slidingWin.buffer[:])
    sf.slidingWin.readBytes = uint64(n)

    sf.slidingWin.checkSum = NewCheckSum(sf.slidingWin.buffer[:CHUNK_SIZE])
    sf.slidingWin.l_idx = CHUNK_SIZE - 1
    return sf
} 

func (sw *SlidingWindow) roll() (err error) {
    sw.k_idx++;
    sw.l_idx++;
    
    if sw.readBytes == uint64(sw.k_idx) {
        return ErrSWStuck
    }

    sw.a_sum = (sw.a_sum - uint64(sw.buffer[sw.k_idx]) +
                uint64(sw.buffer[sw.l_idx + 1])) % MOD2_16
    sw.b_sum = (sw.b_sum - uint64(sw.l_idx - sw.k_idx + 1) * uint64(sw.buffer[sw.k_idx]) +
                sw.a_sum) % MOD2_16
    sw.checkSum = CheckSum(sw.a_sum) + MOD2_16 * CheckSum(sw.b_sum)

    return nil
} 


func (chunk Chunk) String() string {
	return fmt.Sprintf(
		"checksum : %v\n" +
		"md5 hash : %v\n" +
		"offset   : %v\n" +
		"size     : %v\n" ,
		chunk.checkSum, chunk.strongHash,
		chunk.offset, chunk.size,
	)
}

func (rf RemoteFile) String() string {
    chunkStr := fmt.Sprintf(
        "size     : %v\n" +
        "chunks   : %v\n" , 
        rf.fileSize, rf.chunkCount,
    )

    for idx, el := range rf.chunkList {
        chunkStr += fmt.Sprintf("\tChunk%v\t\n", idx)
        chunkStr += el.String()
    }

    return chunkStr
}

