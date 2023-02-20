package file_level

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type RemoteFile struct {
	File     *os.File
	FilePath string

	ChunkList  []Chunk
	ChunkCount uint64
}

// for the strongHash we are using MD5
// rsync is based on MD4

type SourceFile struct {
	File     *os.File
	FileSize uint64
	reader   *bufio.Reader

	slidingWin SlidingWindow
}

func (sf *SourceFile) Next(respType ResponseType) (err error) {
	err = sf.slidingWin.CheckStuck(respType)

	if err == ErrSWSize {
		sf.Read(true)
	}

	switch respType {
	case A_BLOCK:
		err = sf.slidingWin.Roll()
	case B_BLOCK:
		err = sf.slidingWin.RollChunk()
	}
	return err
}

func CreateRemoteFile(filePath string) RemoteFile {
	var rf RemoteFile
	var err error

	rf.FilePath = filePath
	rf.File, err = os.Open(filePath)
	if err != nil {
		panic(err)
	}

	defer rf.File.Close()

	r := bufio.NewReader(rf.File)
	for ; ; rf.ChunkCount++ {
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

		rf.ChunkList = append(rf.ChunkList, Chunk{
			checkSum,
			md5.Sum(buf),
			rf.ChunkCount * CHUNK_SIZE,
			uint64(n),
			rf.ChunkCount,
		})
	}

	return rf
}

func CreateSourceFile(filePath string) SourceFile {
	var sf SourceFile
	var err error

	sf.File, err = os.Open(filePath)
	if err != nil {
		panic(err)
	}

	stats, _ := sf.File.Stat()
	sf.FileSize = uint64(stats.Size())

	sf.reader = bufio.NewReader(sf.File)
	n, _ := sf.reader.Read(sf.slidingWin.buffer[:])
	sf.slidingWin.readBytes = uint64(n)
	sf.slidingWin.cap = uint64(n)

	sf.slidingWin.checkSum, sf.slidingWin.a_sum, sf.slidingWin.b_sum = NewCheckSum(sf.slidingWin.buffer[:CHUNK_SIZE])
	sf.slidingWin.l_idx = CHUNK_SIZE - 1
	return sf
}
func (rf *RemoteFile) WriteSyncedFile(response *Response, filePath string, replace bool) error {
	if rf.FilePath == filePath {
		filePath += ".tmp"
	}

	syncedFile, err := os.Create(filePath)
	CheckErr(err)
	rf.File, err = os.Open(rf.FilePath)
	CheckErr(err)

	defer rf.File.Close()
	defer syncedFile.Close()

	for idx := range *response {
		var responsePack *ResponsePacket = &((*response)[idx])
		switch responsePack.BlockType {
		case A_BLOCK:
			syncedFile.Write(responsePack.Data)
		case B_BLOCK:
			chunkIdx := binary.LittleEndian.Uint64(responsePack.Data)
			chunk := &rf.ChunkList[chunkIdx]
			_, err := rf.File.Seek(int64(chunk.offset), 0)

			CheckErr(err)

			buf := make([]byte, CHUNK_SIZE)
			n, _ := rf.File.Read(buf)
			if n != CHUNK_SIZE {
				panic(n)
			}

			syncedFile.Write(buf)
		}
	}
	if replace {
		os.Remove(rf.FilePath)
		os.Rename(filePath, rf.FilePath)
	}
	return nil
}

func (rf RemoteFile) String() string {
	chunkStr := fmt.Sprintf(
		"chunks   : %v \n ", rf.ChunkCount,
	)

	for idx, el := range rf.ChunkList {
		chunkStr += fmt.Sprintf("\tChunk%v\t\n", idx)
		chunkStr += el.String()
	}

	return chunkStr
}

func (sf SourceFile) String() string {
	return fmt.Sprintf(
		"filesize : %v \n ",
		sf.FileSize,
	)
}

// TODO refactor this

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
