package delta_copying

import (
	"fmt"
	"testing"
)

func TestRemoteFile(t *testing.T) {
    t.Run("Create Remote File", func(t *testing.T) {
        const default_path string = "res/t8.shakespeare.modif.txt"
        rf := CreateRemoteFile(default_path)
        fmt.Printf("%v", rf)
    })   
}
