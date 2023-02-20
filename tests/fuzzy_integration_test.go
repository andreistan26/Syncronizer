package sync_test

import (
	"testing"

	"github.com/andreistan26/sync/tests/fuzzy"
)

func TestFuzzySample(t *testing.T) {
	t.Run("Sample test", func(t *testing.T) {
		fuzzy.RunFuzzyTest("test_data/json/sample_fuzzy.json", t)
	})
}
