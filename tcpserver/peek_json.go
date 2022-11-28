package tcpserver

import "gitlab.com/TitanInd/hashrouter/lib"

// PeekNewLine overcomes limitation of bufio.Peek method, waits until buffer reaches newline char
// and returns buffer without advancing the reader
func PeekNewLine(b bufferedConn) ([]byte, error) {
	var (
		peeked []byte
		err    error
	)

	for i := 1; true; i++ {
		peeked, err = b.r.Peek(i)
		if err != nil {
			return nil, err
		}
		char := peeked[len(peeked)-1]
		if char == lib.CharNewLine {
			break
		}
	}

	return peeked, nil
}

// PeekJSON overcomes limitation of bufio.Peek method, waits until buffer contains full JSON message
// and returns it without advancing the reader. If the message is invalid error is returned
func PeekJSON(b bufferedConn) ([]byte, error) {
	counter := 0
	var CurlyOpen = byte('{')
	var CurlyClosed = byte('}')
	var SquareOpen = byte('[')
	var SquareClosed = byte(']')

	var (
		peeked []byte
		err    error
	)

	for i := 1; true; i++ {
		peeked, err = b.r.Peek(i)
		if err != nil {
			return nil, err
		}
		char := peeked[len(peeked)-1]
		if char == CurlyOpen || char == SquareOpen {
			counter++
		}
		if char == CurlyClosed || char == SquareClosed {
			counter--
		}
		if counter == 0 {
			break
		}
	}
	return peeked, nil
}
