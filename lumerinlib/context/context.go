package contextlib

import (
	"context"
	"fmt"
	"net"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
)

type ContextValue string

const ContextKey ContextValue = "ContextKey"

const (
	LevelPanic log.Level = log.LevelPanic
	LevelFatal log.Level = log.LevelFatal
	LevelError log.Level = log.LevelError
	LevelWarn  log.Level = log.LevelWarn
	LevelInfo  log.Level = log.LevelInfo
	LevelDebug log.Level = log.LevelDebug
	LevelTrace log.Level = log.LevelTrace
)

var levelMap = map[log.Level]string{
	LevelPanic: "PANIC",
	LevelFatal: "FATAL",
	LevelError: "ERROR",
	LevelWarn:  " WARN",
	LevelInfo:  " INFO",
	LevelDebug: "DEBUG",
	LevelTrace: "TRACE",
}

//
// ContextStruct is used to pass variables though the context, and have it pass down from the top
// to all of the sub-system and go routines.  Important values are for logging, msgbus, etc
//
type ContextStruct struct {
	MsgBus *msgbus.PubSub
	Log    *log.Logger
	Src    net.Addr
	Dest   *msgbus.Dest
}

func NewContextStruct(proto interface{}, msgbus *msgbus.PubSub, log *log.Logger, src net.Addr, dst net.Addr) (s *ContextStruct) {
	return &ContextStruct{
		MsgBus: msgbus,
		Log:    log,
		Src:    src,
		Dest:   nil,
	}
}

//
//
//
func (s *ContextStruct) SetMsgBus(x *msgbus.PubSub) {
	s.MsgBus = x
}

//
//
//
func (s *ContextStruct) SetSrc(x net.Addr) {

	// Src validation here

	s.Src = x
}

//
//
//
func (s *ContextStruct) SetDest(x *msgbus.Dest) {

	// Dst validation here

	s.Dest = x
}

//
//
//
func (s *ContextStruct) SetLog(x *log.Logger) {
	s.Log = x
}

//
//
//
func (s *ContextStruct) GetMsgBus() *msgbus.PubSub {
	return s.MsgBus
}

//
//
//
func (s *ContextStruct) GetSrc() net.Addr {
	return s.Src
}

//
//
//
func (s *ContextStruct) GetDest() *msgbus.Dest {
	return s.Dest
}

//
//
//
func (s *ContextStruct) GetLog() *log.Logger {
	return s.Log
}

//
//
//
func (s *ContextStruct) Logf(level log.Level, format string, args ...interface{}) {
	if s.Log != nil {
		s.Log.Logf(level, format, args...)
	} else {
		fmt.Printf(levelMap[level]+":"+format+"\n", args...)
	}
}

//
//
//
func GetContextStruct(ctx context.Context) (s *ContextStruct) {
	val := ctx.Value(ContextKey)
	if val == nil {
		panic("Unable to retrieve ContextKey from Context")
	}
	s, ok := val.(*ContextStruct)
	if !ok {
		panic(fmt.Sprintf("Unable to retrieve Context Value with ContextKey Val:%t", val))
	}
	return s
}

//
//
//
func SetContextStruct(ctx context.Context, cs *ContextStruct) (newctx context.Context) {
	return context.WithValue(ctx, ContextKey, cs)
}

func Logf(ctx context.Context, level log.Level, format string, args ...interface{}) {
	v := ctx.Value(ContextKey)

	val, ok := v.(*ContextStruct)
	if ok && val.Log != nil {
		GetContextStruct(ctx).Logf(level, format, args...)
		return
	}

	str := fmt.Sprintf(levelMap[level]+":"+format+"\n", args...)

	if level == log.LevelPanic {
		panic(str)
	}

	fmt.Print(str)
}

//
//
//
func GetMsgBus(ctx context.Context) (x *msgbus.PubSub) {
	return GetContextStruct(ctx).MsgBus
}

//
//
//
func GetSrc(ctx context.Context) (x net.Addr) {
	return GetContextStruct(ctx).Src
}

//
//
//
func GetDest(ctx context.Context) (x *msgbus.Dest) {
	return GetContextStruct(ctx).Dest
}

//
//
//
func CreateNewContext(ctx context.Context) (newctx context.Context, cancel func()) {
	cs := ctx.Value(ContextKey)
	cs, ok := cs.(*ContextStruct)
	if !ok {
		cs = &ContextStruct{}
	}
	newctx, cancel = context.WithCancel(ctx)
	newctx = context.WithValue(newctx, ContextKey, cs)
	return newctx, cancel
}
