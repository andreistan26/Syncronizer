package delta_copying

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
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
    index       uint64
}

type SourceFile struct {
    file        *os.File
    fileSize    uint64
    reader      *bufio.Reader

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

    k_idx       uint64
    l_idx       uint64
    a_sum       uint64
    b_sum       uint64
}

type RsyncExchange struct {
    sourceFile  *SourceFile
    chunkList   []Chunk
    hashMap     HashMap
}

type ResponseType int

const (
    B_BLOCK ResponseType = iota
    A_BLOCK
)

type ResponsePacket struct {
    blockType   ResponseType
    data        []byte
}

type Response []ResponsePacket

type HashMap map[CheckSum] []*Chunk

type CheckSum uint64

// Testable Parameters
const (
	CHUNK_SIZE          = 4096
	SW_BUFFER_SIZE      = CHUNK_SIZE * 4
    MOD2_16             = 1 << 16
)

// Errors
var( 
    ErrNoFile       = errors.New("No file bound to the type!")
    ErrSWStuck      = errors.New("No more bytes to slide!")
    ErrSWSize       = errors.New("Sliding window size is not equal to CHUNK_SIZE")
)


// adler-32 hash
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

			if err == io.EOF {
				break
			}

			if err != nil {
				panic(err)
			}

		}

		rf.chunkList = append(rf.chunkList, Chunk{
			NewCheckSum(buf),
			md5.Sum(buf),
			rf.chunkCount * CHUNK_SIZE,
			uint64(n),
            rf.chunkCount,
		})
	}

	return rf
}

// TODO add error handle
func CreateRsyncExchange (sf *SourceFile, remoteChunks []Chunk) (RsyncExchange, error) {
    ex := RsyncExchange{
        sourceFile: sf,
        chunkList: remoteChunks,
        hashMap: make(HashMap),
    }

    for _, chunk := range remoteChunks {
        if _, ok := ex.hashMap[chunk.checkSum]; !ok {
            ex.hashMap[chunk.checkSum] = make([]*Chunk, 0)
        }
        ex.hashMap[chunk.checkSum] = append(ex.hashMap[chunk.checkSum], &chunk)
    }

    return ex, nil
}

func (sf *SourceFile) GetCurrentSW() []byte {
    if (sf.slidingWin.l_idx - sf.slidingWin.k_idx + 1) != CHUNK_SIZE {
        panic(ErrSWSize)
    }
    return sf.slidingWin.buffer[sf.slidingWin.k_idx : sf.slidingWin.l_idx]
}

// copy pasted from https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
// apparently one of the only safe ways to do this smh
// TODO remove this pls
func RemoveIndex(s []*Chunk, index int) []*Chunk {
    ret := make([]*Chunk, 0)
    ret = append(ret, s[:index]...)
    return append(ret, s[index+1:]...) 
}

func (ex RsyncExchange) Search() (response Response) {
    var packetAData []byte
SlideLoop:
    for ;; {

        // check if current checksum is entry in the hashmap
        if res, ok := ex.hashMap[ex.sourceFile.slidingWin.checkSum]; ok{
            
            // linear search hashmap value at found key
            for idx, chunk := range res {

                // check if candidate has the same strong hash as the window
                if chunk.strongHash == md5.Sum(ex.sourceFile.GetCurrentSW()) {
                  
                    // empty the type A buffer into a packet and 
                    // append it to reonstruction header
                    if len(packetAData) > 0 {
                        response = append(response, ResponsePacket{
                            A_BLOCK,
                            packetAData,
                        })
                        packetAData = packetAData[:0]
                    }
                    
                    // construct the type B packet
                    idxBytes := make([]byte, 8)
                    binary.LittleEndian.PutUint64(idxBytes, chunk.index)
                    response = append(response, ResponsePacket{
                        B_BLOCK,
                        idxBytes,
                    })
                                       
                    // remove chunk from hashmap value list
                    // TODO optimize this ugly thing
                    ex.hashMap[ex.sourceFile.slidingWin.checkSum] = RemoveIndex(res, idx)
                    continue SlideLoop
                }
            }

            // tracking the weak checksum collisions
            fmt.Fprintf(os.Stderr, "Checksum matched entry but no hash!")
        }

        // TODO optimize this, appending every byte..
        packetAData = append(packetAData, ex.sourceFile.slidingWin.buffer[ex.sourceFile.slidingWin.k_idx])

        // construction of the type A packet
        // also checking the buffer size in order to manage memory usage
        if len(packetAData) == CHUNK_SIZE {
            response = append(response, ResponsePacket{
                A_BLOCK,
                packetAData,
            })
            packetAData = packetAData[:0]
        }
        
        // rolling the window
        ex.sourceFile.slidingWin.roll()
    }
    return response
}

// TODO refactor this
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

	reader := bufio.NewReader(sf.file)
    n, err := reader.Read(sf.slidingWin.buffer[:])
    sf.slidingWin.readBytes = uint64(n)

    sf.slidingWin.checkSum = NewCheckSum(sf.slidingWin.buffer[:CHUNK_SIZE])
    sf.reader = reader
    sf.slidingWin.l_idx = CHUNK_SIZE - 1
    return sf
} 

// returns io.EOF if i cannot read anymore 
func (sf *SourceFile) Read() error {
    
    return nil
}

func (sw *SlidingWindow) roll() (err error) {
    sw.k_idx++;
    sw.l_idx++;
    
    if sw.readBytes == sw.k_idx {
        return ErrSWStuck
    }

    sw.a_sum = (sw.a_sum - uint64(sw.buffer[sw.k_idx]) +
                uint64(sw.buffer[sw.l_idx + 1])) % MOD2_16
    sw.b_sum = (sw.b_sum - (sw.l_idx - sw.k_idx + 1) * uint64(sw.buffer[sw.k_idx]) +
                sw.a_sum) % MOD2_16
    sw.checkSum = CheckSum(sw.a_sum) + MOD2_16 * CheckSum(sw.b_sum)

    return nil
} 

func (chunk Chunk) String() string {
    chunkStr := fmt.Sprintf(
		"checksum : %v \n " +
		"md5 hash : %v \n " +
		"offset   : %v \n " +
		"size     : %v \n " ,
		chunk.checkSum, chunk.strongHash,
		chunk.offset, chunk.size,
	)
    return chunkStr
}

func (rf RemoteFile) String() string {
    chunkStr := fmt.Sprintf(
        "size     : %v \n " +
        "chunks   : %v \n " , 
        rf.fileSize, rf.chunkCount,
    )


    for idx, el := range rf.chunkList {
        chunkStr += fmt.Sprintf("\tChunk%v\t\n", idx)
        chunkStr += el.String()
    }

    return chunkStr
}


