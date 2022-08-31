package interfaces

type IBaseModel interface {
	GetID() string
	SetID(ID string) IBaseModel
}
