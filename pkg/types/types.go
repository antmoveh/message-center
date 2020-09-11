package types

import (
	"encoding/json"
	"github.com/gorilla/websocket"
)

// 推送类型
const (
	PUSH_TYPE_ROOM = 1 // 推送房间
	PUSH_TYPE_ALL  = 2 // 推送在线
)

// websocket Message对象
type WSMessage struct {
	MessageType int
	MessageData []byte
}

// 业务消息的固定格式
type BizMessage struct {
	Type string          `json:"type"` // type类型： PING PONG JOIN LEAVE PUSH
	Data json.RawMessage `json:"data"` // 消息内容
}

// PUSH
type BizPushData struct {
	Items []*json.RawMessage
}

// PING
type BizPingData struct {
}

// PONG
type BizPongData struct {
}

// JOIN
type BizJoinData struct {
	Room string `json:"room"`
}

// LEAVE
type BizLeaveData struct {
	Room string `json:"room"`
}

func BuildWSMessage(messageType int, MessageData []byte) *WSMessage {
	return &WSMessage{
		MessageType: messageType,
		MessageData: MessageData,
	}
}

// 将业务消息，编码为推送消息
func EncodeWSMessage(bizMessage *BizMessage) (*WSMessage, error) {
	var (
		buf []byte
		err error
	)
	if buf, err = json.Marshal(*bizMessage); err != nil {
		return nil, err
	}
	wsMessage := &WSMessage{
		MessageType: websocket.TextMessage,
		MessageData: buf,
	}
	return wsMessage, nil
}

// 将推送消息，解码为业务消息
func DecodeBizMessage(buf []byte) (*BizMessage, error) {

	bizMessage := BizMessage{}
	if err := json.Unmarshal(buf, &bizMessage); err != nil {
		return nil, err
	}
	return &bizMessage, nil
}
