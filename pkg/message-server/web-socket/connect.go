package web_socket

import (
	"github.com/gorilla/websocket"
	"message-center/cmd/message/config"
	"message-center/pkg/types"
	"message-center/utils"
	"sync"
	"time"
)

type WSConnection struct {
	mutex             sync.Mutex
	connId            uint64                // 每个连接唯一ID
	wsSocket          *websocket.Conn       // socket连接
	inChan            chan *types.WSMessage // 收到的消息
	outChan           chan *types.WSMessage // 发出的消息
	closeChan         chan byte             // 收到消息时断开连接
	isClosed          bool                  // 处于关闭状态时，连接已关闭
	lastHeartbeatTime time.Time             // 最近一次心跳时间
	rooms             map[string]bool       // 加入了哪些房间
}

// 初始化单个socket连接，
func InitWSConnection(connId uint64, wsSocket *websocket.Conn) (wsConnection *WSConnection) {
	wsConnection = &WSConnection{
		wsSocket:          wsSocket,
		connId:            connId,
		inChan:            make(chan *types.WSMessage, config.GlobalServerConfig.WsInChannelSize),
		outChan:           make(chan *types.WSMessage, config.GlobalServerConfig.WsOutChannelSize),
		closeChan:         make(chan byte),
		lastHeartbeatTime: time.Now(),
		rooms:             make(map[string]bool),
	}

	go wsConnection.readLoop()
	go wsConnection.writeLoop()

	return
}

// websocket读循环
func (wsConnection *WSConnection) readLoop() {
	var (
		msgType int
		msgData []byte
		message *types.WSMessage
		err     error
	)
	for {
		if msgType, msgData, err = wsConnection.wsSocket.ReadMessage(); err != nil {
			wsConnection.Close()
			return
		}

		message = types.BuildWSMessage(msgType, msgData)

		select {
		case wsConnection.inChan <- message:
		case <-wsConnection.closeChan:
			return
		}
	}
}

// websocket写循环
func (wsConnection *WSConnection) writeLoop() {
	var (
		message *types.WSMessage
		err     error
	)
	for {
		select {
		case message = <-wsConnection.outChan:
			if err = wsConnection.wsSocket.WriteMessage(message.MessageType, message.MessageData); err != nil {
				wsConnection.Close()
				return
			}
		case <-wsConnection.closeChan:
			return
		}
	}
}

// 发送消息
func (wsConnection *WSConnection) SendMessage(message *types.WSMessage) error {
	var err error
	select {
	case wsConnection.outChan <- message:
	case <-wsConnection.closeChan:
		err = utils.ConnectionLossError
	default: // 写操作不会阻塞, 因为channel已经预留给websocket一定的缓冲空间
		err = utils.SendMessageFull
	}
	return err
}

// 读取消息
func (wsConnection *WSConnection) ReadMessage() (message *types.WSMessage, err error) {
	select {
	case message = <-wsConnection.inChan:
	case <-wsConnection.closeChan:
		err = utils.ConnectionLossError
	}
	return
}

// 关闭连接
func (wsConnection *WSConnection) Close() {
	wsConnection.wsSocket.Close()

	wsConnection.mutex.Lock()
	defer wsConnection.mutex.Unlock()

	if !wsConnection.isClosed {
		wsConnection.isClosed = true
		close(wsConnection.closeChan)
	}
}

// 检查心跳（不需要太频繁）
func (wsConnection *WSConnection) IsAlive() bool {
	var (
		now = time.Now()
	)

	wsConnection.mutex.Lock()
	defer wsConnection.mutex.Unlock()

	// 连接已关闭 或者 太久没有心跳
	if wsConnection.isClosed || now.Sub(wsConnection.lastHeartbeatTime) > time.Duration(config.GlobalServerConfig.WsHeartbeatInterval)*time.Second {
		return false
	}
	return true
}

// 更新心跳
func (WSConnection *WSConnection) KeepAlive() {
	var (
		now = time.Now()
	)

	WSConnection.mutex.Lock()
	defer WSConnection.mutex.Unlock()

	WSConnection.lastHeartbeatTime = now
}
