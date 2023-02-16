package delta_copying

import (
	"fmt"
	"testing"
    "os"
)


func TestFileCreate(t *testing.T) {
    var rf RemoteFile
    var sf SourceFile
    var ex RsyncExchange

    t.Run("Create Remote File", func(t *testing.T) {
        const default_path string = "res/file1_rem.sync"
        rf = CreateRemoteFile(default_path)
        fmt.Printf("%v", rf)
    })

    t.Run("Create Source File", func(t *testing.T) {
        const default_path string = "res/file1_src.sync"
        sf = CreateSourceFile(default_path)
        fmt.Printf("%v", sf)
    })

    t.Run("Create Rsync Exchange", func(t *testing.T) {
        var err error
        ex, err = CreateRsyncExchange(&sf, rf.chunkList)
        
        if err != nil {
            fmt.Printf(err.Error())
        }
    })
    
    t.Run("Perform a search", func(t *testing.T) {
        resp := ex.Search()

        fmt.Printf("%v", resp)

        gotMap := map[ResponseType]int{
            A_BLOCK : 0,
            B_BLOCK : 0,
        }

        wantMap := map[ResponseType]int{
            A_BLOCK : 1,
            B_BLOCK : 3,
        }

        AssertPackageTypeByCount(t, resp, gotMap, wantMap);
    })
}

func TestPerfectMatch(t *testing.T) {
    // TODO replace this with other global scoped function
    assert_response_type_B := func (t testing.TB, resp Response, respSizeRef int) {
        t.Helper()

        if len(resp) != respSizeRef {
            t.Errorf(
                "Resp does not have the right size",
            )
        }

        for idx, el := range resp {
            if el.blockType != B_BLOCK {
                t.Errorf(
                    "Found block of type A{ idx : %d} when i wanted type B!",
                    idx,
            )
            }
        }
    }

    assert_correct_hashmap := func (t testing.TB, ex RsyncExchange) {
        for _, chunk := range ex.chunkList {
            if _, ok := ex.hashMap[chunk.checkSum]; !ok {
                t.Errorf("chunk was not found in the hashMap")
            }
        }
    }

    t.Run("1 Block", func(t *testing.T) {
        const default_path_src = "res/typeB/1_block_src.sync"
        const default_path_rem = "res/typeB/1_block_rem.sync"


        rf := CreateRemoteFile(default_path_rem)
        sf := CreateSourceFile(default_path_src)
        fmt.Printf("%v", sf)
        ex, _ := CreateRsyncExchange(&sf, rf.chunkList)

        resp := ex.Search()

        fmt.Printf("%v", resp)

        assert_response_type_B(t, resp, 1)
        
    })

    t.Run("2 Blocks", func(t *testing.T) {
        const default_path_src = "res/typeB/2_block_src.sync"
        const default_path_rem = "res/typeB/2_block_rem.sync"


        rf := CreateRemoteFile(default_path_rem)
        sf := CreateSourceFile(default_path_src)
        ex, _ := CreateRsyncExchange(&sf, rf.chunkList)

        resp := ex.Search()

        assert_response_type_B(t, resp, 2)
        
    })

    t.Run("5 Blocks", func(t *testing.T) {
        const default_path_src = "res/typeB/5_block_src.sync"
        const default_path_rem = "res/typeB/5_block_rem.sync"


        rf := CreateRemoteFile(default_path_rem)
        sf := CreateSourceFile(default_path_src)
        ex, _ := CreateRsyncExchange(&sf, rf.chunkList)

        resp := ex.Search()

        assert_response_type_B(t, resp, 5)
        assert_correct_hashmap(t, ex)
        
    })
    
    t.Run("128 Blocks", func(t *testing.T) {
        const default_path_src = "res/typeB/128_block_src.sync"
        const default_path_rem = "res/typeB/128_block_rem.sync"


        rf := CreateRemoteFile(default_path_rem)
        sf := CreateSourceFile(default_path_src)
        ex, _ := CreateRsyncExchange(&sf, rf.chunkList)

        resp := ex.Search()

        assert_response_type_B(t, resp, 128)
    })
}

func TestReminder(t *testing.T) {
    t.Run("1 Block + 128 bytes", func(t *testing.T) {
        const default_path_src = "res/remTest/1_chunk_128_rem_src.sync"
        const default_path_rem = "res/remTest/1_chunk_128_rem_rem.sync"
        
        rf := CreateRemoteFile(default_path_rem)
        sf := CreateSourceFile(default_path_src)
        ex, _ := CreateRsyncExchange(&sf, rf.chunkList)

        ex.Search()
    })
}

func TestWriteFile(t *testing.T) {
    t.Run("2 Chunks + 128 bytes", func(t *testing.T) {
        const hostPath = "res/writeFile/2_chunk_128_src.sync"
        const remPath = "res/writeFile/2_chunk_128_rem.sync"

        rf := CreateRemoteFile(remPath)
        sf := CreateSourceFile(hostPath)
        ex, _ := CreateRsyncExchange(&sf, rf.chunkList)
        
        resp := ex.Search()

        rf.WriteSyncedFile(&resp, "res/writeFile/2_chunk_128_res.sync")
    })
}


func AssertPackageTypeByCount(t testing.TB, resp Response, gotMap, wantMap map[ResponseType]int) {
    t.Helper()

    for _, pack := range resp {
        gotMap[pack.blockType]++
    }

    for key, val := range wantMap {
        if val != gotMap[key] {
            t.Errorf(
                "Search failed," +  
                "received A : %d, B : %d" + 
                "wanted   A : %d, B : %d",
                wantMap[A_BLOCK], wantMap[B_BLOCK],
                gotMap[A_BLOCK], wantMap[B_BLOCK],
            )
        }
    }
}

func MakeSearchTestHelper(t testing.TB, sourceFilePath, remoteFilePath, resFilePath, logFilePath string) (sf SourceFile, rf RemoteFile, resp Response) {
    t.Helper()

    sf = CreateSourceFile(sourceFilePath)
    rf = CreateRemoteFile(remoteFilePath)
    ex, _ := CreateRsyncExchange(&sf, rf.chunkList)
    defer sf.file.Close()
    resp = ex.Search()

    if logFilePath != "" {
        logFile, err := os.Create(logFilePath)
        if err == nil {
            t.Fatal(err)
        }
        logFile.WriteString(sf.String())
        logFile.WriteString(rf.String())
        logFile.WriteString(resp.String())
    }

    return sf, rf, resp
} 
