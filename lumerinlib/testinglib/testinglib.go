package testinglib

import (
	"context"
	"math/rand"

	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

var basePort int = 50000
var localhost string = "127.0.0.1"

//
//
//
func GetRandPort() (port int) {
	port = rand.Intn(10000) + basePort
	return port
}

func GetNewContextWithValueStruct() (ctx context.Context) {

	ctx = context.Background()
	cs := &contextlib.ContextStruct{}
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)
	// cs.SetMsgBus(ps)
	// cs.SetSrc(src)
	// cs.SetDest(dest)

	return ctx
}
