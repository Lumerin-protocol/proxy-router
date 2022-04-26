package msgbus

import (
	"fmt"
	"time"

	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

type ConnectionState string
type ConnectionID IDString

const (
	ConnNewState          ConnectionState = "NewState"
	ConnSrcSubscribeState ConnectionState = "SrcSubscribeState"
	ConnAuthState         ConnectionState = "AuthState"
	ConnVerifyState       ConnectionState = "VerifyState"
	ConnRoutingState      ConnectionState = "RoutingState"
	ConnConnectingState   ConnectionState = "ConnectingState"
	ConnConnectedState    ConnectionState = "ConnectedState"
	ConnConnectErrState   ConnectionState = "ConnectErrState"
	ConnMsgErrState       ConnectionState = "MsgErrState"
	ConnRouteChangeState  ConnectionState = "RouteChangeState"
	ConnDstCloseState     ConnectionState = "DstCloseState"
	ConnSrcCloseState     ConnectionState = "SrcCloseState"
	ConnShutdownState     ConnectionState = "ShutdownState"
	ConnErrorState        ConnectionState = "ErrorState"
	ConnClosedState       ConnectionState = "ClosedState"
)

//
// Created & Updated by Connection Manager
//
type Connection struct {
	ID        ConnectionID
	Miner     MinerID
	Dest      DestID
	State     ConnectionState
	TotalHash int
	StartDate time.Time
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) ConnPubWait(conn Connection) (c Connection, err error) {

	if conn.ID == "" {
		conn.ID = ConnectionID(GetRandomIDString())
	}

	event, err := ps.PubWait(ConnectionMsg, IDString(conn.ID), conn)
	if err != nil || event.Err != nil {
		panic(fmt.Sprintf(lumerinlib.Funcname()+"Unable to add Record %s, %s\n", err, event.Err))
	}

	c = event.Data.(Connection)
	if err != nil || event.Err != nil {
		fmt.Printf(lumerinlib.Funcname()+" PubWait returned err: %s, %s\n", err, event.Err)
	}

	return c, err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) ConnGetWait(id ConnectionID) (conn *Connection, err error) {

	if id == "" {
		panic(fmt.Sprintf(lumerinlib.Funcname() + " ID not provided\n"))
	}

	event, err := ps.GetWait(ConnectionMsg, IDString(id))
	if err != nil || event.Err != nil {
		fmt.Printf(lumerinlib.Funcname()+" ID not found %s, %s\n", err, event.Err)
	}

	if event.Data == nil {
		conn = nil
	} else {
		c := event.Data.(Connection)
		conn = &c
	}
	return conn, err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) ConnSetWait(conn Connection) (err error) {

	if conn.ID == "" {
		panic(fmt.Sprintf(lumerinlib.Funcname() + " ID not provided\n"))
	}

	_, err = ps.ConnGetWait(conn.ID)
	if err != nil {
		return err
	}

	e, err := ps.SetWait(ConnectionMsg, IDString(conn.ID), conn)
	if err != nil {
		return err
	}

	if e.Err != nil {
		return e.Err
	}

	return nil

}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) ConnUpdateStateWait(id ConnectionID, state ConnectionState) (err error) {

	if id == "" {
		panic(fmt.Sprintf(lumerinlib.Funcname() + " ID not provided\n"))
	}

	conn, err := ps.ConnGetWait(id)
	if err != nil {
		fmt.Printf(lumerinlib.Funcname()+" not found %s\n", err)
		return err
	}

	if state != conn.State {
		conn.State = state
		err = ps.ConnSetWait(*conn)
		if err != nil {
			fmt.Printf(lumerinlib.Funcname()+" ConnSetWait failed %s\n", err)
			return err
		}

	}

	return nil
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) ConnExistsWait(id ConnectionID) bool {

	dest, _ := ps.ConnGetWait(id)

	return dest != nil
}
