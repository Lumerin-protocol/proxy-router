package mining

type MiningMessage struct {
	rawMessage     string
	messageMethods []*MiningMethodCallPayload
}

type MiningMethodCallPayload struct {
	id         string
	methodType string
	params     []map[string]interface{}
}
