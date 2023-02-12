package delta_copying

import (
	"fmt"
	"testing"
)

func TestFileCreate(t *testing.T) {
    var rf RemoteFile
    var sf SourceFile
    var ex RsyncExchange

    t.Run("Create Remote File", func(t *testing.T) {
        const default_path string = "res/t8.shakespeare.modif.txt"
        rf = CreateRemoteFile(default_path)
        fmt.Printf("%v", rf)
    })

    t.Run("Create Source File", func(t *testing.T) {
        const default_path string = "res/t8.shakespeare.txt"
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
    })
}
