package interfaces

type IModel interface {
	GetID() string
}

type ICollection[T IModel] interface {
	Load(ID string) (item T, ok bool)
	Range(f func(item T) bool)
	Store(item T)
	Delete(ID string)
}
