package interfaces

import (
	"io"
	"sync"
)

type LogStorage struct {
	Buffer MyBuf
	id     string
}

func NewLogStorage(ID string) *LogStorage {
	return &LogStorage{
		Buffer: NewMyBuf(),
		id:     ID,
	}
}

func (l *LogStorage) ID() string {
	return l.id
}

func (l *LogStorage) GetReader() io.Reader {
	return l.Buffer.NewReader()
}

type MyBuf interface {
	io.Writer
	NewReader() io.Reader
}

type mybuf struct {
	data [][]byte
	sync.RWMutex
}

func (mb *mybuf) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	// Cannot retain p, so we must copy it:
	p2 := make([]byte, len(p))
	copy(p2, p)
	mb.Lock()
	mb.data = append(mb.data, p2)
	mb.Unlock()
	return len(p), nil
}

type mybufReader struct {
	mb   *mybuf // buffer we read from
	i    int    // next slice index
	data []byte // current data slice to serve
}

func (mbr *mybufReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	// Do we have data to send?
	if len(mbr.data) == 0 {
		mb := mbr.mb
		mb.RLock()
		if mbr.i < len(mb.data) {
			mbr.data = mb.data[mbr.i]
			mbr.i++
		}
		mb.RUnlock()
	}
	if len(mbr.data) == 0 {
		return 0, io.EOF
	}

	n = copy(p, mbr.data)
	mbr.data = mbr.data[n:]
	return n, nil
}

func (mb *mybuf) NewReader() io.Reader {
	return &mybufReader{mb: mb}
}

func NewMyBuf() MyBuf {
	return &mybuf{}
}
