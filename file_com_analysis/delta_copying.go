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
	file     *os.File
	filePath string

	chunkList  []Chunk
	chunkCount uint64
}

// for the strongHash we are using MD5
// rsync is based on MD4
type Chunk struct {
	checkSum   CheckSum
	strongHash [16]byte

	offset uint64
	size   uint64
	index  uint64
}

type SourceFile struct {
	file     *os.File
	fileSize uint64
	reader   *bufio.Reader

	slidingWin SlidingWindow
}

type SlidingWindow struct {
	checkSum CheckSum

	buffer    [CHUNK_SIZE * 4]byte
	readBytes uint64
	cap       uint64

	k_idx uint64
	l_idx uint64
	a_sum uint64
	b_sum uint64
}

type RsyncExchange struct {
	sourceFile *SourceFile
	chunkList  []Chunk
	hashMap    HashMap
}

type ResponseType int

const (
	B_BLOCK ResponseType = iota
	A_BLOCK
)

type ResponsePacket struct {
	blockType ResponseType
	data      []byte
}

type Response []ResponsePacket

type HashMap map[CheckSum][]*Chunk

type CheckSum uint64

// Testable Parameters
const (
	CHUNK_SIZE     = 4096
	SW_BUFFER_SIZE = CHUNK_SIZE * 4
	MOD2_16        = 1 << 16
)

// Errors
var (
	ErrNoFile       = errors.New("No file bound to the type!")
	ErrSWStuck      = errors.New("No more bytes to slide!")
	ErrSWSize       = errors.New("Sliding window size is not equal to CHUNK_SIZE!")
	ErrSWSizeRem    = errors.New("Sliding window is stuck but still has data to be read!")
	ErrSWSizeNoData = io.EOF
)

// adler-32 hash
// Source : https://rsync.samba.org/tech_report/node3.html
// x represents the value of the byte from the window
// a(k, l) = (sum i=k->l : x_i) % MOD2_16
// b(k, l) = (sum i=k->l : (l-i+1) * x_i ) % MOD2_16
// checkSum = a(k, l) + MOD2_16 * b(k, l)
func NewCheckSum(bytes []byte) (sum CheckSum, a_sum, b_sum uint64) {

	chunk_len := len(bytes)

	for idx, el := range bytes {
		a_sum += uint64(el)
		b_sum += uint64(chunk_len-idx) * uint64(el)
	}

	a_sum %= MOD2_16
	b_sum %= MOD2_16
	sum = CheckSum(a_sum) + CheckSum(b_sum)*MOD2_16

	return sum, a_sum, b_sum
}

func CreateRemoteFile(filePath string) RemoteFile {
	var rf RemoteFile
	var err error

	rf.filePath = filePath
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

		} else if n < CHUNK_SIZE {
			break
		}

		checkSum, _, _ := NewCheckSum(buf)

		rf.chunkList = append(rf.chunkList, Chunk{
			checkSum,
			md5.Sum(buf),
			rf.chunkCount * CHUNK_SIZE,
			uint64(n),
			rf.chunkCount,
		})
	}

	return rf
}

// TODO add error handle
func CreateRsyncExchange(sf *SourceFile, remoteChunks []Chunk) (RsyncExchange, error) {
	ex := RsyncExchange{
		sourceFile: sf,
		chunkList:  remoteChunks,
		hashMap:    make(HashMap),
	}

	for idx, chunk := range remoteChunks {
		if _, ok := ex.hashMap[chunk.checkSum]; !ok {
			ex.hashMap[chunk.checkSum] = make([]*Chunk, 0)
		}
		ex.hashMap[chunk.checkSum] = append(ex.hashMap[chunk.checkSum], &ex.chunkList[idx])
	}

	return ex, nil
}

func (sw *SlidingWindow) GetBuffer() []byte {
	if (sw.l_idx - sw.k_idx + 1) != CHUNK_SIZE {
		panic(ErrSWSize)
	}
	return sw.buffer[sw.k_idx : sw.l_idx+1]
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
	var err error = nil
SearchLoop:
	for err == nil {

		// check if current checksum is entry in the hashmap
		if res, ok := ex.hashMap[ex.sourceFile.slidingWin.checkSum]; ok {
			// linear search hashmap value at found key
			for idx, chunk := range res {

				// check if candidate has the same strong hash as the window
				if chunk.strongHash == md5.Sum(ex.sourceFile.slidingWin.GetBuffer()) {
					// empty the type A buffer into a packet and
					// append it to reonstruction header
					if len(packetAData) > 0 {
						response = append(response, ResponsePacket{
							A_BLOCK,
							packetAData,
						})
						packetAData = nil
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
					err = ex.sourceFile.Next(B_BLOCK)
					continue SearchLoop
				}
				fmt.Fprintf(os.Stderr, "Checksum matched but strongHash didn't")
				fmt.Fprintf(os.Stderr, "%v", chunk)
			}
			// TODO replace with acutal log, bad way to keep track of things
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
			packetAData = nil
		}
		err = ex.sourceFile.Next(A_BLOCK)
	}

	if err == ErrSWSizeRem {
		packetAData = append(packetAData, ex.sourceFile.slidingWin.buffer[ex.sourceFile.slidingWin.l_idx+1:ex.sourceFile.slidingWin.cap]...)
		for len(packetAData) > 0 {
			var dim int

			if len(packetAData) < CHUNK_SIZE {
				dim = len(packetAData)
			} else {
				dim = CHUNK_SIZE
			}

			response = append(response, ResponsePacket{
				A_BLOCK,
				packetAData[:dim],
			})
			packetAData = packetAData[dim:]
		}
	}

	// TODO continue to search and append remaining bytes like [Chunk][Chunk][rem]
	// rem will not be added from the loop above
	return response
}

// TODO refactor this
func CreateSourceFile(filePath string) SourceFile {
	var sf SourceFile
	var err error

	sf.file, err = os.Open(filePath)
	if err != nil {
		panic(err)
	}

	stats, _ := sf.file.Stat()
	sf.fileSize = uint64(stats.Size())

	sf.reader = bufio.NewReader(sf.file)
	n, err := sf.reader.Read(sf.slidingWin.buffer[:])
	sf.slidingWin.readBytes = uint64(n)
	sf.slidingWin.cap = uint64(n)

	sf.slidingWin.checkSum, sf.slidingWin.a_sum, sf.slidingWin.b_sum = NewCheckSum(sf.slidingWin.buffer[:CHUNK_SIZE])
	sf.slidingWin.l_idx = CHUNK_SIZE - 1
	return sf
}

// returns io.EOF if i cannot read anymore
// my buffer is 4 * CHUNK_SIZE so i need to read 3 * CHUNK_SIZE
// reads next 3 * CHUNK_SIZE bytes from file and resets k and l
func (sf *SourceFile) Read(resetWindowBounds bool) (int, error) {
	var newBuf [4 * CHUNK_SIZE]byte
	copy(newBuf[:], sf.slidingWin.buffer[sf.slidingWin.k_idx:sf.slidingWin.cap])
	dif := sf.slidingWin.cap - sf.slidingWin.k_idx
	n, err := sf.reader.Read(newBuf[dif:])
	sf.slidingWin.buffer = newBuf
	sf.slidingWin.readBytes += uint64(n)
	sf.slidingWin.cap = uint64(n) + dif
	if resetWindowBounds {
		sf.slidingWin.l_idx = CHUNK_SIZE - 1
		sf.slidingWin.k_idx = 0
	}
	return n, err
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

func (sf *SourceFile) Next(respType ResponseType) (err error) {
	err = sf.slidingWin.checkStuck(respType)

	if err == ErrSWSize {
		sf.Read(true)
	}

	switch respType {
	case A_BLOCK:
		err = sf.slidingWin.roll()
	case B_BLOCK:
		err = sf.slidingWin.rollChunk()
	}
	return err
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
	sw.k_idx++
	sw.l_idx++

	sw.a_sum = (sw.a_sum - uint64(sw.buffer[sw.k_idx-1]) +
		uint64(sw.buffer[sw.l_idx])) % MOD2_16
	sw.b_sum = (sw.b_sum - CHUNK_SIZE*uint64(sw.buffer[sw.k_idx-1]) +
		sw.a_sum) % MOD2_16
	sw.checkSum = CheckSum(sw.a_sum) + MOD2_16*CheckSum(sw.b_sum)

	return nil
}

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}

func (rf *RemoteFile) WriteSyncedFile(response *Response, filePath string) error {
	syncedFile, err := os.Create(filePath)
	CheckErr(err)
	rf.file, err = os.Open(rf.filePath)
	CheckErr(err)

	defer rf.file.Close()
	defer syncedFile.Close()

	for idx, _ := range *response {
		var responsePack *ResponsePacket = &((*response)[idx])
		switch responsePack.blockType {
		case A_BLOCK:
			syncedFile.Write(responsePack.data)
		case B_BLOCK:
			chunkIdx := binary.LittleEndian.Uint64(responsePack.data)
			chunk := &rf.chunkList[chunkIdx]
			_, err := rf.file.Seek(int64(chunk.offset), 0)

			CheckErr(err)

			buf := make([]byte, CHUNK_SIZE)
			n, err := rf.file.Read(buf)
			if n != CHUNK_SIZE {
				panic(n)
			}

			syncedFile.Write(buf)
		}
	}
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

func (rf RemoteFile) String() string {
	chunkStr := fmt.Sprintf(
		"chunks   : %v \n ", rf.chunkCount,
	)

	for idx, el := range rf.chunkList {
		chunkStr += fmt.Sprintf("\tChunk%v\t\n", idx)
		chunkStr += el.String()
	}

	return chunkStr
}

func (sf SourceFile) String() string {
	return fmt.Sprintf(
		"filesize : %v \n ",
		sf.fileSize,
	)
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

func (resp ResponseType) String() string {
	switch resp {
	case A_BLOCK:
		return "A_BLOCK"
	default:
		return "B_BLOCK"
	}
}

func (packet ResponsePacket) String() string {
	return fmt.Sprintf(
		"Block Type : %v \n "+
			"Data       : %v \n "+
			"Size       : %v \n ",
		packet.blockType, packet.data, len(packet.data),
	)
}

func (response Response) String() string {
	var responseStr string
	for idx, el := range response {
		responseStr += fmt.Sprintf(
			"\tPacket %v \n"+
				"%v\n",
			idx, el,
		)
	}
	return responseStr
}
