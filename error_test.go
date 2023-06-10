package rest
import (
	"testing"
)

func  TestError(t *testing.T){
	err1 := Error{Code: 25, Message: "not found"}

    err2 := err1.Error()
    if err2 != "not found" {
        t.Error("ahtung")
    }
}

