package stratumv1_message

type MiningMessageGeneric interface {
	Serialize() []byte
}

type MiningMessageToPool interface {
	MiningMessageGeneric
	GetID() int
	SetID(int)
}
