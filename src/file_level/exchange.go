package file_level

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"os"
)

type RsyncExchange struct {
	sourceFile *SourceFile
	ChunkList  []Chunk
	HashMap    HashMap
}

type HashMap map[CheckSum][]*Chunk

// TODO add error handle
func CreateRsyncExchange(sf *SourceFile, remoteChunks []Chunk) (RsyncExchange, error) {
	ex := RsyncExchange{
		sourceFile: sf,
		ChunkList:  remoteChunks,
		HashMap:    make(HashMap),
	}

	for idx := range remoteChunks {
		ex.HashMap[ex.ChunkList[idx].CheckSum] = append(ex.HashMap[ex.ChunkList[idx].CheckSum], &ex.ChunkList[idx])
	}

	return ex, nil
}

// copy pasted from https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
// apparently one of the only safe ways to do this smh
// TODO remove this pls
func RemoveIndex(s []*Chunk, index int) []*Chunk {
	ret := make([]*Chunk, len(s)-1)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

func (ex *RsyncExchange) Search() (response Response) {
	var packetAData []byte
	var err error = nil
SearchLoop:
	for err == nil {

		// check if current checksum is entry in the hashmap
		if res := ex.HashMap[ex.sourceFile.slidingWin.checkSum]; len(res) > 0 {
			// linear search hashmap value at found key
			for idx, chunk := range res {

				// check if candidate has the same strong hash as the window
				if chunk.StrongHash == md5.Sum(ex.sourceFile.slidingWin.GetBuffer()) {
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
					binary.LittleEndian.PutUint64(idxBytes, chunk.Index)
					response = append(response, ResponsePacket{
						B_BLOCK,
						idxBytes,
					})

					// remove chunk from hashmap value list
					// TODO optimize this ugly thing

					ex.HashMap[ex.sourceFile.slidingWin.checkSum] = RemoveIndex(res, idx)
					err = ex.sourceFile.Next(B_BLOCK)
					continue SearchLoop
				}
			}
			fmt.Fprintf(os.Stderr, "Checksum matched but strongHash didn't, %v vals\n", ex.HashMap[ex.sourceFile.slidingWin.checkSum])
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
