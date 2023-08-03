package interfaces
type ISocketConfig interface {
	LoadSystemConfig()

	SetSysctl(key string, value string)
	GetSysctl(key string) (string)

	SetRlimit(limit uint64)

	GetLocalPortRange() string
	SetLocalPortRange(value string)

	GetTcpMaxSynBacklog() string
	SetTcpMaxSynBacklog(value string)

	GetSomaxconn() string 
	SetSomaxconn(value string)

	GetNetdevMaxBacklog() string
	SetNetdevMaxBacklog(value string)

	GetRlimitSoft() uint64
	SetRlimitSoft(value uint64)

	GetRlimitHard() uint64
	SetRlimitHard(value uint64)
}
