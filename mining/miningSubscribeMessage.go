package mining

type MiningSubscribeMethodCallPayload struct {
	MiningMessageBase
	workerNameParam   string
	workerNumberParam string
}

func (m *MiningSubscribeMethodCallPayload) UnmarshalJSON(buf []byte) (err error) {

	miningMessageBase, err := unMarshalEmbedded(buf)

	if err != nil {
		return err
	}

	m.MiningMessageBase = *miningMessageBase

	m.workerNameParam = m.Params[0].(string)
	m.workerNumberParam = m.Params[1].(string)

	return err
}
