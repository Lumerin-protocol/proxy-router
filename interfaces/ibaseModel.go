package interfaces

type IBaseModel interface {
	GetID() string
	SetID(string) IBaseModel
}
