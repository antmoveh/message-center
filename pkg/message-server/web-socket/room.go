package web_socket

import (
	"message-center/pkg/types"
	"message-center/utils"
	"sync"
)

type RoomInterface interface {
	// 加入房间
	Join(connection *WSConnection) error
	// 离开房间
	Leave(connection *WSConnection) error
	// 房间内连接个数
	Count() int
	// 推送消息
	Push(message *types.WSMessage)
}

// 房间
type Room struct {
	rwMutex sync.RWMutex
	roomId  string
	id2Conn map[uint64]*WSConnection
}

func InitRoom(roomId string) (room *Room) {
	room = &Room{
		roomId:  roomId,
		id2Conn: make(map[uint64]*WSConnection),
	}
	return
}

func (room *Room) Join(wsConn *WSConnection) (err error) {
	var (
		existed bool
	)

	room.rwMutex.Lock()
	defer room.rwMutex.Unlock()

	if _, existed = room.id2Conn[wsConn.connId]; existed {
		err = utils.JoinRoomTwice
		return
	}

	room.id2Conn[wsConn.connId] = wsConn
	return
}

func (room *Room) Leave(wsConn *WSConnection) (err error) {
	var (
		existed bool
	)

	room.rwMutex.Lock()
	defer room.rwMutex.Unlock()

	if _, existed = room.id2Conn[wsConn.connId]; !existed {
		err = utils.NotInRoom
		return
	}

	delete(room.id2Conn, wsConn.connId)
	return
}

func (room *Room) Count() int {
	room.rwMutex.RLock()
	defer room.rwMutex.RUnlock()

	return len(room.id2Conn)
}

func (room *Room) Push(wsMsg *types.WSMessage) {
	var (
		wsConn *WSConnection
	)
	room.rwMutex.RLock()
	defer room.rwMutex.RUnlock()

	for _, wsConn = range room.id2Conn {
		wsConn.SendMessage(wsMsg)
	}
}
