package interfaces

type BaseModelTypeConstraint[T any] interface {
	IBaseModel
	*T
}

type IRepository[T any] interface {
	Get(string) (T, error)
	Save(payload T) (T, error)
	Create(payload T) (T, error)
	Update(payload T) (T, error)
	FindOne(T) (T, error)
	Query(T) []T
	Delete(T) (T, error)
}

type IConcreteRepository[T any] interface {
	Get(string) (*T, error)
	Save(payload *T) (*T, error)
	Create(payload *T) (*T, error)
	Update(payload *T) (*T, error)
	FindOne(*T) (*T, error)
	Query(*T) []*T
	Delete(*T) (*T, error)
}
