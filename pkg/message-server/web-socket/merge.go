package web_socket

import (
	"encoding/json"
	"message-center/cmd/message/config"
	"message-center/pkg/types"
)

// 广播消息、房间消息的合并
type MessageMerge struct {
	roomWorkers     []*MergeWorker // 房间合并
	broadcastWorker *MergeWorker   // 广播合并
	stopChan        chan byte      // 关闭
}

func InitMessageMerger() error {
	var (
		workerIdx int
		merger    *MessageMerge
	)

	merger = &MessageMerge{
		roomWorkers: make([]*MergeWorker, config.GlobalServerConfig.MergerWorkerCount),
		stopChan:    make(chan byte, 1),
	}
	for workerIdx = 0; workerIdx < config.GlobalServerConfig.MergerWorkerCount; workerIdx++ {
		merger.roomWorkers[workerIdx] = initMergeWorker(types.PUSH_TYPE_ROOM, merger.stopChan)
	}
	merger.broadcastWorker = initMergeWorker(types.PUSH_TYPE_ALL, merger.stopChan)

	GlobalMessageMergeServer = merger

	return nil
}

// 广播合并推送
func (merger *MessageMerge) PushAll(msg *json.RawMessage) (err error) {
	return merger.broadcastWorker.pushAll(msg)
}

// 房间合并推送
func (merger *MessageMerge) PushRoom(room string, msg *json.RawMessage) (err error) {
	// 计算room hash到某个worker
	var (
		workerIdx uint32 = 0
		ch        byte
	)
	for _, ch = range []byte(room) {
		workerIdx = (workerIdx + uint32(ch)*33) % uint32(config.GlobalServerConfig.MergerWorkerCount)
	}
	return merger.roomWorkers[workerIdx].pushRoom(room, msg)
}

func (merger *MessageMerge) MergeClose() {
	close(merger.stopChan)
}
