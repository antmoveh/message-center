package push

import (
	"encoding/json"
	"github.com/prometheus/common/log"
	"message-center/cmd/logic/config"
	"message-center/pkg/types"
	"message-center/utils"
)

type managerInterface interface {
	PushAll(items []json.RawMessage) error
	PushRoom(roomId string, items []json.RawMessage) error
	MessageConnectClose()
}

type PushJob struct {
	pushType int               // 推送类型
	roomId   string            // 房间ID
	items    []json.RawMessage // 要推送的消息数组
}

type MessageConnectManager struct {
	ServerConns  []*ServerConn // 到所有Message Server的连接数组
	pendingChan  []chan byte   // message server的并发请求控制
	dispatchChan chan *PushJob // 待分发的推送
	stopChan     chan byte     // 关闭连接
}

func InitConnManager() error {
	var (
		serverIdx         int
		dispatchWorkerIdx int
		serverConfig      config.MessageServerConfig
		serverConnMgr     *MessageConnectManager
		err               error
	)

	serverConnMgr = &MessageConnectManager{

		ServerConns:  make([]*ServerConn, len(config.GlobalLogicConfig.MessageServerList)),
		pendingChan:  make([]chan byte, len(config.GlobalLogicConfig.MessageServerList)),
		dispatchChan: make(chan *PushJob, config.GlobalLogicConfig.MessageServerDispatchChannelSize),
		stopChan:     make(chan byte, 1),
	}

	for serverIdx, serverConfig = range config.GlobalLogicConfig.MessageServerList {
		if serverConnMgr.ServerConns[serverIdx], err = InitMessageServerConn(&serverConfig); err != nil {
			return err
		}
		serverConnMgr.pendingChan[serverIdx] = make(chan byte, config.GlobalLogicConfig.MessageServerMaxPendingCount)
	}

	for dispatchWorkerIdx = 0; dispatchWorkerIdx < config.GlobalLogicConfig.MessageServerDispatchWorkerCount; dispatchWorkerIdx++ {
		go serverConnMgr.dispatchWorkerMain(dispatchWorkerIdx)
	}

	GlobalConnectManager = serverConnMgr

	return nil
}

func (serverConnMgr *MessageConnectManager) PushAll(items []json.RawMessage) (err error) {
	var (
		pushJob *PushJob
	)

	pushJob = &PushJob{
		pushType: types.PUSH_TYPE_ALL,
		items:    items,
	}

	select {
	case serverConnMgr.dispatchChan <- pushJob:
	default:
		err = utils.LogicDisPatchChannelFull
	}
	return
}

func (serverConnMgr *MessageConnectManager) PushRoom(roomId string, items []json.RawMessage) (err error) {
	var (
		pushJob *PushJob
	)

	pushJob = &PushJob{
		pushType: types.PUSH_TYPE_ROOM,
		roomId:   roomId,
		items:    items,
	}

	select {
	case serverConnMgr.dispatchChan <- pushJob:
	default:
		err = utils.LogicDisPatchChannelFull
	}
	return
}

// 推送给一个message server
func (serverConnMgr *MessageConnectManager) doPush(gatewayIdx int, pushJob *PushJob, itemsJson []byte) {
	if pushJob.pushType == types.PUSH_TYPE_ALL {
		serverConnMgr.ServerConns[gatewayIdx].PushAll(itemsJson)
	} else if pushJob.pushType == types.PUSH_TYPE_ROOM {
		serverConnMgr.ServerConns[gatewayIdx].PushRoom(pushJob.roomId, itemsJson)
	}

	// 释放名额
	<-serverConnMgr.pendingChan[gatewayIdx]
}

// 消息分发协程
func (serverConnMgr *MessageConnectManager) dispatchWorkerMain(dispatchWorkerIdx int) {
	var (
		pushJob   *PushJob
		serverIdx int
		itemsJson []byte
		err       error
	)
	for {
		select {
		case <-serverConnMgr.stopChan:
			log.Info("worker 终止")
			return
		case pushJob = <-serverConnMgr.dispatchChan:
			// 序列化
			if itemsJson, err = json.Marshal(pushJob.items); err != nil {
				continue
			}
			// 分发到所有message server
			for serverIdx = 0; serverIdx < len(serverConnMgr.ServerConns); serverIdx++ {
				select {
				case serverConnMgr.pendingChan[serverIdx] <- 1: // 并发控制
					go serverConnMgr.doPush(serverIdx, pushJob, itemsJson)
				default: // 并发已满, 直接丢弃
				}
			}
		}
	}
}

func (serverConnMgr *MessageConnectManager) MessageConnectClose() {
	serverConnMgr.ServerConns = nil
	close(serverConnMgr.stopChan)
}
