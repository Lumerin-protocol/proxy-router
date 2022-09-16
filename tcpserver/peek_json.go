package tcpserver

// PeekJSON overcomes limitation of bufio.Peek method, allowing to peek json message of unknown length
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
