package fuzzy

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	mrand "math/rand"
	"os"
	"path"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/andreistan26/sync/src/file_level"
)

type Pair [2]int64

type FuzzyMutationOptions struct {
	MutationRange Pair  `json:"mutation_range"` // %
	MaxInsertA    int64 `json:"max_insert_a"`
	HasTypeA      bool  `json:"has_type_a,omittype"`
	HasTypeB      bool  `json:"has_type_b"`
	ChunkSize     int64 `json:"chunk_size"`
}

type FuzzyTestOptions struct {
	TestName      string               `json:"test_name"`
	Iterations    int64                `json:"iterations"`
	FileSizeRange Pair                 `json:"file_size_range"`
	Mutation      FuzzyMutationOptions `json:"mutation"`
}

type FuzzyTestList struct {
	FuzzyTests []FuzzyTestOptions `json:"tests"`
}

type FuzzyTest struct {
	testOptions FuzzyTestOptions
	sourceFile  *os.File
	remoteFile  *os.File

	response file_level.Response
}

const DEFAULT_FUZZY_PATH = "test_data/fuzzy"

func RunFuzzyTest(jsonFilePath string, t testing.TB) (err error) {
	t.Helper()

	var jsonData FuzzyTestList

	fileData, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		t.Errorf("JSON file does not exist at path [%s]", jsonFilePath)
		return err
	}

	var wg sync.WaitGroup

	json.Unmarshal(fileData, &jsonData)
	for _, testSpec := range jsonData.FuzzyTests {
		dirPath := path.Join(DEFAULT_FUZZY_PATH, testSpec.TestName)
		os.MkdirAll(dirPath, os.ModePerm)
		// unsafe but unlikely
		for idx := 0; idx < int(testSpec.Iterations); idx++ {
			wg.Add(1)
			go func(dirPath string, idx int, testSpec FuzzyTestOptions) {
				defer wg.Done()
				baseName := path.Join(dirPath, fmt.Sprintf("%s_%d", testSpec.TestName, idx))
				srcPath := fmt.Sprintf("%s_src.sync", baseName)
				srcFileSize := testSpec.getRandomFileSize()
				remotePath := fmt.Sprintf("%s_rem.sync", baseName)
				resPath := fmt.Sprintf("%s_res.sync", baseName)
				logPath := fmt.Sprintf("%s.log", baseName)
				fuzzyUnit := FuzzyTest{
					testOptions: testSpec,
				}
				fuzzyUnit.CreateRandomDataFile(srcPath, srcFileSize)
				fuzzyUnit.CopyCreateFile(remotePath)

				// mutate source file
				fuzzyUnit.MutateSourceFile()

				fuzzyUnit.sourceFile.Close()
				fuzzyUnit.remoteFile.Close()

				MakeSearchTestHelper(t, fuzzyUnit.sourceFile.Name(), fuzzyUnit.remoteFile.Name(),
					resPath, logPath, &fuzzyUnit.response)
			}(dirPath, idx, testSpec)
		}
		wg.Wait()
	}
	return nil
}

func (test FuzzyTestOptions) getRandomFileSize() int64 {
	randNum, err := rand.Int(rand.Reader, new(big.Int).SetInt64(test.FileSizeRange[1]-test.FileSizeRange[0]))
	CheckErr(err)
	return randNum.Int64() + test.FileSizeRange[0]
}

func (ft *FuzzyTest) CreateRandomDataFile(filePath string, size int64) {
	var err error
	ft.sourceFile, err = os.Create(filePath)
	CheckErr(err)

	_, err = io.CopyN(ft.sourceFile, rand.Reader, size)
	CheckErr(err)
}

func (ft *FuzzyTest) CopyCreateFile(destPath string) {
	var err error

	ft.remoteFile, err = os.Create(destPath)
	CheckErr(err)

	// reseting the file pointer in order to copy the whole thing
	ft.sourceFile.Seek(0, 0)

	_, err = io.Copy(ft.remoteFile, ft.sourceFile)
	CheckErr(err)
}

func (test *FuzzyTest) MutateSourceFile() {
	mrand.Seed(int64(time.Now().Nanosecond()))

	mutationPerc := mrand.Int63n(test.testOptions.Mutation.MutationRange[1]-test.testOptions.Mutation.MutationRange[0]) +
		test.testOptions.Mutation.MutationRange[0]
	fileStats, _ := test.sourceFile.Stat()
	fileSize := fileStats.Size()
	mutationSize := int64(float64(mutationPerc) / 100.0 * float64(fileSize))

	mutatedFile, err := os.Create(test.sourceFile.Name() + ".tmp")
	CheckErr(err)

	// find possible blocks
	var blockTypes []file_level.ResponseType
	blockTypes = append(blockTypes, file_level.A_BLOCK)

	if test.testOptions.Mutation.HasTypeB && fileSize/test.testOptions.Mutation.ChunkSize != 0 {
		blockTypes = append(blockTypes, file_level.B_BLOCK)
	}

	blockTypesLen := len(blockTypes)
	for currSz := 0; int64(currSz) < (mutationSize + fileSize); {
		currBlockType := blockTypes[mrand.Int31n(int32(blockTypesLen))]
		test.response = append(test.response, file_level.ResponsePacket{
			BlockType: currBlockType,
		})
		switch currBlockType {
		case file_level.A_BLOCK:
			insertSize := mrand.Int63n(test.testOptions.Mutation.MaxInsertA)
			randomBytes := make([]byte, insertSize)
			rand.Read(randomBytes)
			mutatedFile.Write(randomBytes)

			test.response[len(test.response)-1].Data = randomBytes
			currSz += int(insertSize)
		case file_level.B_BLOCK:
			randomChunkIdx := mrand.Int63n(fileSize / test.testOptions.Mutation.ChunkSize)
			test.sourceFile.Seek(randomChunkIdx*test.testOptions.Mutation.ChunkSize, 0)
			chunkBytes := make([]byte, test.testOptions.Mutation.ChunkSize)
			test.sourceFile.Read(chunkBytes)
			mutatedFile.Write(chunkBytes)

			test.response[len(test.response)-1].Data = make([]byte, 8)
			binary.LittleEndian.PutUint64(test.response[len(test.response)-1].Data, uint64(randomChunkIdx))
			currSz += int(test.testOptions.Mutation.ChunkSize)
		}
	}

	os.Remove(test.sourceFile.Name())
	os.Rename(mutatedFile.Name(), test.sourceFile.Name())
}

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Refactor so that log file is opened since the creation of the test
func MakeSearchTestHelper(t testing.TB, sourceFilePath, remoteFilePath, resFilePath, logFilePath string,
	creationResp *file_level.Response) {
	t.Helper()

	sf := file_level.CreateSourceFile(sourceFilePath)
	rf := file_level.CreateRemoteFile(remoteFilePath)
	ex, _ := file_level.CreateRsyncExchange(&sf, rf.ChunkList)
	defer sf.File.Close()
	defer rf.File.Close()
	resp := ex.Search()
	rf.WriteSyncedFile(&resp, resFilePath, false)

	if logFilePath != "" && !AssertFileHash(t, sf.File, rf.File) {
		logFile, err := os.Create(logFilePath)
		if err != nil {
			t.Fatal(err)
		}
		logFile.WriteString(sf.String())
		logFile.WriteString(rf.String())
		logFile.WriteString(resp.String())
		logFile.WriteString("\nCreation Response\n")
		logFile.WriteString(creationResp.String())
	} else {
		// os.Remove(sf.File.Name())
		// os.Remove(rf.File.Name())
		// os.Remove(resFilePath)
	}
}

func AssertFileHash(t testing.TB, sourcePath, remotePath *os.File) bool {
	t.Helper()

	assertHashErrCheck := func(file *os.File) []byte {
		hash1, err := calculateFileHash(t, sourcePath)
		if err != nil {
			t.Errorf("file path error when trying to calculate hash")
		}
		return hash1
	}
	hash1 := assertHashErrCheck(sourcePath)
	hash2 := assertHashErrCheck(sourcePath)

	if !reflect.DeepEqual(hash1, hash2) {
		t.Errorf("hash of files is not equal [%s, %s]", sourcePath.Name(), remotePath.Name())
		return false
	}
	return true
}

func calculateFileHash(t testing.TB, file *os.File) ([]byte, error) {
	t.Helper()

	file.Seek(0, 0)
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Errorf("calculating hash error : %s", err.Error())
		return nil, err
	}
	return hash.Sum(nil), nil
}
