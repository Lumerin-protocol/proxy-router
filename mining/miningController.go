package mining

import "gitlab.com/TitanInd/hashrouter/interfaces"

type MiningController struct {
	interfaces.IMiningRequestProcessor
	request  string
	response string
}
