package interfaces

type ISocketConfig interface {
	LoadSystemConfig()
	SetSysctl(key string, value string)
	GetSysctl(key string) (string)
	SetRlimit(limit uint64)	
}