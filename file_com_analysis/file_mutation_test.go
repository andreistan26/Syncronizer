package delta_copying

import (
	"encoding/binary"
	"math/rand"
	"os"
	"time"
)

func (test FuzzyTest) MutateSourceFile() {    
    rand.Seed(int64(time.Now().Nanosecond()))

    mutationPerc := rand.Int63n(test.testOptions.Mutation.MutationRange[1]-test.testOptions.Mutation.MutationRange[0]) +
                        test.testOptions.Mutation.MutationRange[0]
    fileStats, _ := test.sourceFile.Stat()
    fileSize := fileStats.Size()
    mutationSize := (mutationPerc / 100) * fileSize
    
    
    mutatedFile, err := os.Create(test.sourceFile.Name() + ".tmp")
    CheckErr(err)

    // find possible blocks
    var blockTypes []ResponseType
    if test.testOptions.Mutation.HasTypeA {
        blockTypes = append(blockTypes, A_BLOCK)
    }
    
    if test.testOptions.Mutation.HasTypeB {
        blockTypes = append(blockTypes, B_BLOCK)
    }

    for currSz := 0; int64(currSz) < mutationSize; {
        currBlockType := blockTypes[rand.Int31n(2)]
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
            test.sourceFile.Seek(0, int(randomChunkIdx * test.testOptions.Mutation.ChunkSize))
            chunkBytes := make([]byte, test.testOptions.Mutation.ChunkSize)
            test.sourceFile.Read(chunkBytes)
            mutatedFile.Write(chunkBytes)

            binary.LittleEndian.PutUint64(test.response[len(test.response)-1].data, uint64(randomChunkIdx))
            currSz += int(test.testOptions.Mutation.ChunkSize)
        }
    }

    os.Remove(test.sourceFile.Name())
    os.Rename(mutatedFile.Name(), test.sourceFile.Name())
}
