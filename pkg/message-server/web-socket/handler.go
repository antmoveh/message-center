package web_socket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"message-center/cmd/message/config"
	"message-center/pkg/types"
	"message-center/utils"
	"time"
)

// websocket处理器，主要负责任务如下
// 首先将连接添加到连接管理器
// 每隔60进行健康检查，健康检查的维持需要从client发起PING请求，响应PONG
// 处理加入房间和退出房间业务，客户端发起JOIN类型请求则将连接加入ROOM，客户端发起LEAVE类型请求则离开ROOM
func (wsConnection *WSConnection) WSHandle() {
	var (
		message *types.WSMessage
		bizReq  *types.BizMessage
		bizResp *types.BizMessage
		err     error
		buf     []byte
	)

	// 连接加入管理器, 可以推送端查找到
	GlobalSocketConnectionManager.AddConn(wsConnection)

	// 心跳检测线程
	go wsConnection.heartbeatChecker()

	defer func() {
		// 确保连接关闭
		wsConnection.Close()
		// 离开所有房间
		wsConnection.leaveAll()
		// 从连接池中移除
		GlobalSocketConnectionManager.DelConn(wsConnection)
	}()

	// 请求处理协程
	for {
		if message, err = wsConnection.ReadMessage(); err != nil {
			return
		}

		// 只处理文本消息
		if message.MessageType != websocket.TextMessage {
			continue
		}

		// 解析消息体
		if bizReq, err = types.DecodeBizMessage(message.MessageData); err != nil {
			return
		}

		bizResp = nil

		// 1,收到PING则响应PONG: {"type": "PING"}, {"type": "PONG"}
		// 2,收到JOIN则加入ROOM: {"type": "JOIN", "data": {"room": "chrome-plugin"}}
		// 3,收到LEAVE则离开ROOM: {"type": "LEAVE", "data": {"room": "chrome-plugin"}}

		// 请求串行处理
		switch bizReq.Type {
		case "PING":
			if bizResp, err = wsConnection.handlePing(bizReq); err != nil {
				return
			}
		case "JOIN":
			if bizResp, err = wsConnection.handleJoin(bizReq); err != nil {
				return
			}
		case "LEAVE":
			if bizResp, err = wsConnection.handleLeave(bizReq); err != nil {
				return
			}
		}

		if bizResp != nil {
			if buf, err = json.Marshal(*bizResp); err != nil {
				return
			}
			// socket缓冲区写满不是致命错误
			if err = wsConnection.SendMessage(&types.WSMessage{websocket.TextMessage, buf}); err != nil {
				if err != utils.SendMessageFull {
					return
				} else {
					err = nil
				}
			}
		}
	}

}

// 每隔1秒, 检查一次连接是否健康
func (wsConnection *WSConnection) heartbeatChecker() {
	var (
		timer *time.Timer
	)
	timer = time.NewTimer(time.Duration(config.GlobalServerConfig.WsHeartbeatInterval) * time.Second)
	for {
		select {
		case <-timer.C:
			if !wsConnection.IsAlive() {
				wsConnection.Close()
				return
			}
			timer.Reset(time.Duration(config.GlobalServerConfig.WsHeartbeatInterval) * time.Second)
		case <-wsConnection.closeChan:
			timer.Stop()
			return
		}
	}
}

// 处理PING请求
func (wsConnection *WSConnection) handlePing(bizReq *types.BizMessage) (bizResp *types.BizMessage, err error) {
	var (
		buf []byte
	)

	wsConnection.KeepAlive()

	if buf, err = json.Marshal(types.BizPongData{}); err != nil {
		return
	}
	bizResp = &types.BizMessage{
		Type: "PONG",
		Data: json.RawMessage(buf),
	}
	return
}

// 处理JOIN请求
func (wsConnection *WSConnection) handleJoin(bizReq *types.BizMessage) (bizResp *types.BizMessage, err error) {
	var (
		existed bool
	)
	bizJoinData := &types.BizJoinData{}
	if err = json.Unmarshal(bizReq.Data, bizJoinData); err != nil {
		return
	}
	if len(bizJoinData.Room) == 0 {
		err = utils.RoomIdInvalid
		return
	}
	if len(wsConnection.rooms) >= config.GlobalServerConfig.MaxJoinRoom {
		// 超过了房间数量限制, 忽略这个请求
		return
	}
	// 已加入过
	if _, existed = wsConnection.rooms[bizJoinData.Room]; existed {
		// 忽略掉这个请求
		return
	}
	// 建立房间 -> 连接的关系
	if err = GlobalSocketConnectionManager.JoinRoom(bizJoinData.Room, wsConnection); err != nil {
		return
	}
	// 建立连接 -> 房间的关系
	wsConnection.rooms[bizJoinData.Room] = true
	return
}

// 处理LEAVE请求
func (wsConnection *WSConnection) handleLeave(bizReq *types.BizMessage) (bizResp *types.BizMessage, err error) {
	var (
		bizLeaveData *types.BizLeaveData
		existed      bool
	)
	bizLeaveData = &types.BizLeaveData{}
	if err = json.Unmarshal(bizReq.Data, bizLeaveData); err != nil {
		return
	}
	if len(bizLeaveData.Room) == 0 {
		err = utils.RoomIdInvalid
		return
	}
	// 未加入过
	if _, existed = wsConnection.rooms[bizLeaveData.Room]; !existed {
		// 忽略掉这个请求
		return
	}
	// 删除房间 -> 连接的关系
	if err = GlobalSocketConnectionManager.LeaveRoom(bizLeaveData.Room, wsConnection); err != nil {
		return
	}
	// 删除连接 -> 房间的关系
	delete(wsConnection.rooms, bizLeaveData.Room)
	return
}

func (wsConnection *WSConnection) leaveAll() {
	var (
		roomId string
	)
	// 从所有房间中退出
	for roomId, _ = range wsConnection.rooms {
		GlobalSocketConnectionManager.LeaveRoom(roomId, wsConnection)
		delete(wsConnection.rooms, roomId)
	}
}
