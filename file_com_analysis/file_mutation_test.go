package delta_copying

import (
	"encoding/binary"
	"math/rand"
	"os"
	"time"
)

func (test *FuzzyTest) MutateSourceFile() {
	rand.Seed(int64(time.Now().Nanosecond()))

	mutationPerc := rand.Int63n(test.testOptions.Mutation.MutationRange[1]-test.testOptions.Mutation.MutationRange[0]) +
		test.testOptions.Mutation.MutationRange[0]
	fileStats, _ := test.sourceFile.Stat()
	fileSize := fileStats.Size()
	mutationSize := int64(float64(mutationPerc) / 100.0 * float64(fileSize))

	mutatedFile, err := os.Create(test.sourceFile.Name() + ".tmp")
	CheckErr(err)

	// find possible blocks
	var blockTypes []ResponseType
	blockTypes = append(blockTypes, A_BLOCK)

	if test.testOptions.Mutation.HasTypeB && fileSize/test.testOptions.Mutation.ChunkSize != 0 {
		blockTypes = append(blockTypes, B_BLOCK)
	}

	blockTypesLen := len(blockTypes)
	for currSz := 0; int64(currSz) < (mutationSize + fileSize); {
		currBlockType := blockTypes[rand.Int31n(int32(blockTypesLen))]
		test.response = append(test.response, ResponsePacket{
			blockType: currBlockType,
		})
		switch currBlockType {
		case A_BLOCK:
			insertSize := rand.Int63n(test.testOptions.Mutation.MaxInsertA)
			randomBytes := make([]byte, insertSize)
			rand.Read(randomBytes)
			mutatedFile.Write(randomBytes)

			test.response[len(test.response)-1].data = randomBytes
			currSz += int(insertSize)
		case B_BLOCK:
			randomChunkIdx := rand.Int63n(fileSize / test.testOptions.Mutation.ChunkSize)
			test.sourceFile.Seek(randomChunkIdx*test.testOptions.Mutation.ChunkSize, 0)
			chunkBytes := make([]byte, test.testOptions.Mutation.ChunkSize)
			test.sourceFile.Read(chunkBytes)
			mutatedFile.Write(chunkBytes)

			test.response[len(test.response)-1].data = make([]byte, 8)
			binary.LittleEndian.PutUint64(test.response[len(test.response)-1].data, uint64(randomChunkIdx))
			currSz += int(test.testOptions.Mutation.ChunkSize)
		}
	}

	os.Remove(test.sourceFile.Name())
	os.Rename(mutatedFile.Name(), test.sourceFile.Name())
}
