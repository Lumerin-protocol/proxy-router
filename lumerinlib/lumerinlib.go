package lumerinlib

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type ConcurrentMap struct {
	sync.RWMutex
	M map[string]interface{}
}

func (r *ConcurrentMap) Get(key string) interface{} {
	r.RLock()
	defer r.RUnlock()
	return r.M[key]
}

func (r *ConcurrentMap) GetAll() (vals []interface{}) {
	r.RLock()
	defer r.RUnlock()
	for _, v := range r.M {
		vals = append(vals, v)
	}
	return vals
}

func (r *ConcurrentMap) GetMap() map[string]interface{} {
	r.RLock()
	defer r.RUnlock()
	return r.M
}

func (r *ConcurrentMap) Set(key string, val interface{}) {
	r.Lock()
	defer r.Unlock()
	r.M[key] = val
}

func (r *ConcurrentMap) Exists(key string) bool {
	r.RLock()
	defer r.RUnlock()
	_, ok := r.M[key]
	return ok
}

func (r *ConcurrentMap) Delete(key string) {
	r.Lock()
	defer r.Unlock()
	delete(r.M, key)
}

func BoilerPlateLibFunc(msg string) string {
	return msg
}

func FileLineFunc(a ...int) string {
	var depth int = 1

	if len(a) != 0 {
		depth = len(a)
		if depth < 1 && depth > 10 {
			panic(FileLine() + " depth out of bounds")
		}
	}

	pc, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "FileLine() failed"
	}

	lineno := strconv.Itoa(line)

	fileArr := strings.Split(file, "/")
	fileName := fileArr[len(fileArr)-1]

	funcPtr := runtime.FuncForPC(pc)

	var funcName string

	if funcPtr == nil {
		funcName = "TheUNKNOWNFunction()"
	} else {
		f := strings.Split(funcPtr.Name(), "/")
		funcName = f[len(f)-1]
		g := strings.Split(funcName, ".")
		funcName = g[len(g)-1]
	}

	gid := fmt.Sprintf("%04d", getGID())

	ret := "[" + gid + "|" + fileName + ":" + lineno + ":" + funcName + "()]:"
	return ret
}

func FileLine() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return "FileLine() failed"
	}

	f := strings.Split(file, "/")

	lineno := strconv.Itoa(line)

	return "[" + f[len(f)-1] + ":" + lineno + "]:"
}

func Funcname() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "TheUnknownFunction()"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "TheUnknownFunction()"
	}

	f := strings.Split(fn.Name(), "/")

	return f[len(f)-1]
}

func Errtrace() string {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return "file?[0]:func?"
	}

	lineno := strconv.Itoa(line)

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return file + "[" + lineno + "]:func?"
	}

	return file + "[" + lineno + "]:" + fn.Name()
}

func PanicHere(text ...string) string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		panic("Well this is unexpected...")
	}

	f := strings.Split(file, "/")

	lineno := strconv.Itoa(line)

	panic(fmt.Sprintf("[%s:%s]:%s", f[len(f)-1], lineno, text[0]))
}

// borrowed from https://blog.sgmansfield.com/2015/12/goroutine-ids/
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

// goCounter()
// Generates a UniqueID (int) and returns via supplied channel
func RunGoCounter(c chan int) {
	go func() {
		counter := 10000
		for {
			c <- counter
			counter += 1
		}
	}()
}
