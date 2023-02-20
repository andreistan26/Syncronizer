package fuzzy

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParser(t *testing.T) {
	t.Run("One object parse", func(t *testing.T) {
		const fileName = "../test_data/json/one_object_parse.json"
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
		const fileName = "../test_data/json/array_parse.json"
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
