package delta_copying

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
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

	response Response
}

const DEFAULT_FUZZY_PATH = "res/fuzzy"

func TestParser(t *testing.T) {
	t.Run("One object parse", func(t *testing.T) {
		const fileName = "res/json/one_object_parse.json"
		fileData, _ := ioutil.ReadFile(fileName)
		var jsonData FuzzyTestOptions
		json.Unmarshal(fileData, &jsonData)

		want := FuzzyTestOptions{
			TestName:      "sample_test",
			Iterations:    50,
			FileSizeRange: Pair{128, 100000},
			Mutation: FuzzyMutationOptions{
				MutationRange: Pair{30, 60},
				HasTypeA:      false,
				HasTypeB:      true,
				ChunkSize:     4096,
				MaxInsertA:    128,
			},
		}

		if !cmp.Equal(want, jsonData) {
			t.Errorf(
				"Got %v, want %v!",
				jsonData, want,
			)
		}
	})

	t.Run("Array parse", func(t *testing.T) {
		const fileName = "res/json/array_parse.json"
		fileData, _ := ioutil.ReadFile(fileName)
		var jsonData FuzzyTestList
		json.Unmarshal(fileData, &jsonData)

		want := FuzzyTestOptions{
			TestName:      "second_test",
			Iterations:    101,
			FileSizeRange: Pair{101230, 10000},
			Mutation: FuzzyMutationOptions{
				MutationRange: Pair{30, 60},
				HasTypeA:      false,
				HasTypeB:      true,
				ChunkSize:     4096,
				MaxInsertA:    128,
			},
		}

		if !cmp.Equal(want, jsonData.FuzzyTests[1]) {
			t.Errorf(
				"Got %v, want %v!",
				jsonData.FuzzyTests[1], want,
			)
		}

	})
}

func TestFuzzySample(t *testing.T) {
	t.Run("Sample test", func(t *testing.T) {
		RunFuzzyTest("res/json/sample_fuzzy.json", t)
	})
}

func RunFuzzyTest(jsonFilePath string, t testing.TB) {
	t.Helper()

	var jsonData FuzzyTestList

	fileData, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		t.Fatal(err)
	}

	json.Unmarshal(fileData, &jsonData)
	for _, testSpec := range jsonData.FuzzyTests {
		dirPath := path.Join(DEFAULT_FUZZY_PATH, testSpec.TestName)
		os.MkdirAll(dirPath, os.ModePerm)
		// unsafe but unlikely
		for idx := 0; idx < int(testSpec.Iterations); idx++ {
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
				resPath, logPath, fuzzyUnit.response)
		}
	}
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
}

func (ft *FuzzyTest) CopyCreateFile(destPath string) {
	var err error

	ft.remoteFile, err = os.Create(destPath)
	CheckErr(err)

	// reseting the file pointer in order to copy the whole thing
	ft.sourceFile.Seek(0, 0)

	_, err = io.Copy(ft.remoteFile, ft.sourceFile)
}
