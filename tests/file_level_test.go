package sync_test

import (
	"fmt"
	"testing"

	"github.com/andreistan26/sync/src/file_level"
)

func TestFileCreate(t *testing.T) {
	var rf file_level.RemoteFile
	var sf file_level.SourceFile
	var ex file_level.RsyncExchange

	t.Run("Create Remote File", func(t *testing.T) {
		const default_path string = "test_data/file1_rem.sync"
		rf = file_level.CreateRemoteFile(default_path)
	})

	t.Run("Create Source File", func(t *testing.T) {
		const default_path string = "test_data/file1_src.sync"
		sf = file_level.CreateSourceFile(default_path)
	})

	t.Run("Create Rsync Exchange", func(t *testing.T) {
		var err error
		ex, err = file_level.CreateRsyncExchange(&sf, rf.ChunkList)

		if err != nil {
			fmt.Println(err.Error())
		}
	})

	t.Run("Perform a search", func(t *testing.T) {
		resp := ex.Search()

		AssertPackageTypeByCount(t, resp, map[file_level.ResponseType]int{
			file_level.A_BLOCK: 1,
			file_level.B_BLOCK: 3,
		})
	})
}

func TestPerfectMatch(t *testing.T) {
	// TODO replace this with other global scoped function
	assert_response_type_B := func(t testing.TB, resp file_level.Response, respSizeRef int) {
		t.Helper()

		if len(resp) != respSizeRef {
			t.Errorf(
				"Resp does not have the right size",
			)
		}

		for idx, el := range resp {
			if el.BlockType != file_level.B_BLOCK {
				t.Errorf(
					"Found block of type A{ idx : %d} when i wanted type B!",
					idx,
				)
			}
		}
	}

	assert_correct_hashmap := func(t testing.TB, ex file_level.RsyncExchange) {
		for _, chunk := range ex.ChunkList {
			if _, ok := ex.HashMap[chunk.CheckSum]; !ok {
				t.Errorf("chunk was not found in the hashMap")
			}
		}
	}

	t.Run("1 Block", func(t *testing.T) {
		const default_path_src = "test_data/typeB/1_block_src.sync"
		const default_path_rem = "test_data/typeB/1_block_rem.sync"

		rf := file_level.CreateRemoteFile(default_path_rem)
		sf := file_level.CreateSourceFile(default_path_src)
		fmt.Printf("%v", sf)
		ex, _ := file_level.CreateRsyncExchange(&sf, rf.ChunkList)

		resp := ex.Search()

		assert_response_type_B(t, resp, 1)

	})

	t.Run("2 Blocks", func(t *testing.T) {
		const default_path_src = "test_data/typeB/2_block_src.sync"
		const default_path_rem = "test_data/typeB/2_block_rem.sync"

		rf := file_level.CreateRemoteFile(default_path_rem)
		sf := file_level.CreateSourceFile(default_path_src)
		ex, _ := file_level.CreateRsyncExchange(&sf, rf.ChunkList)

		resp := ex.Search()

		assert_response_type_B(t, resp, 2)

	})

	t.Run("5 Blocks", func(t *testing.T) {
		const default_path_src = "test_data/typeB/5_block_src.sync"
		const default_path_rem = "test_data/typeB/5_block_rem.sync"

		rf := file_level.CreateRemoteFile(default_path_rem)
		sf := file_level.CreateSourceFile(default_path_src)
		ex, _ := file_level.CreateRsyncExchange(&sf, rf.ChunkList)

		resp := ex.Search()

		assert_response_type_B(t, resp, 5)
		assert_correct_hashmap(t, ex)

	})

	t.Run("128 Blocks", func(t *testing.T) {
		const default_path_src = "test_data/typeB/128_block_src.sync"
		const default_path_rem = "test_data/typeB/128_block_rem.sync"

		rf := file_level.CreateRemoteFile(default_path_rem)
		sf := file_level.CreateSourceFile(default_path_src)
		ex, _ := file_level.CreateRsyncExchange(&sf, rf.ChunkList)

		resp := ex.Search()

		assert_response_type_B(t, resp, 128)
	})
}

func TestReminder(t *testing.T) {
	t.Run("1 Block + 128 bytes", func(t *testing.T) {
		const default_path_src = "test_data/remTest/1_chunk_128_rem_src.sync"
		const default_path_rem = "test_data/remTest/1_chunk_128_rem_rem.sync"

		rf := file_level.CreateRemoteFile(default_path_rem)
		sf := file_level.CreateSourceFile(default_path_src)
		ex, _ := file_level.CreateRsyncExchange(&sf, rf.ChunkList)

		resp := ex.Search()

		AssertPackageTypeByCount(t, resp, map[file_level.ResponseType]int{
			file_level.A_BLOCK: 1,
			file_level.B_BLOCK: 1,
		})
	})
}

func TestWriteFile(t *testing.T) {
	t.Run("2 Chunk + 128 bytes", func(t *testing.T) {
		const hostPath = "test_data/writeFile/2_chunk_128_src.sync"
		const remPath = "test_data/writeFile/2_chunk_128_rem.sync"

		rf := file_level.CreateRemoteFile(remPath)
		sf := file_level.CreateSourceFile(hostPath)
		ex, _ := file_level.CreateRsyncExchange(&sf, rf.ChunkList)
		resp := ex.Search()

		AssertPackageTypeByCount(t, resp, map[file_level.ResponseType]int{
			file_level.A_BLOCK: 1,
			file_level.B_BLOCK: 2,
		})

		rf.WriteSyncedFile(&resp, "test_data/writeFile/2_chunk_128_res.sync", false)
	})
}

func AssertPackageTypeByCount(t testing.TB, resp file_level.Response, wantMap map[file_level.ResponseType]int) {
	t.Helper()
	gotMap := map[file_level.ResponseType]int{
		file_level.A_BLOCK: 0,
		file_level.B_BLOCK: 0,
	}

	for _, pack := range resp {
		gotMap[pack.BlockType]++
	}

	for key, val := range wantMap {
		if val != gotMap[key] {
			t.Errorf(
				"Search failed,\n"+
					"received A : %d, B : %d\n"+
					"wanted   A : %d, B : %d\n",
				gotMap[file_level.A_BLOCK], gotMap[file_level.B_BLOCK],
				wantMap[file_level.A_BLOCK], wantMap[file_level.B_BLOCK],
			)
		}
	}
}
