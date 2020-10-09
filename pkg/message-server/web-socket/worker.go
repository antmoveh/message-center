package web_socket

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"message-center/cmd/message/config"
	"message-center/pkg/types"
	"message-center/utils"
	"time"
)

type PushBatch struct {
	items       []*json.RawMessage
	commitTimer *time.Timer
	room        string // 按room合并
}

type PushContext struct {
	msg  *json.RawMessage
	room string // 按room合并
}

type MergeWorker struct {
	mergeType int // 合并类型: 广播, room, uid...

	contextChan chan *PushContext
	timeoutChan chan *PushBatch

	room2Batch map[string]*PushBatch // room合并
	allBatch   *PushBatch            // 广播合并
	stopChan   chan byte
}

func initMergeWorker(mergeType int, stopChan chan byte) (worker *MergeWorker) {
	worker = &MergeWorker{
		mergeType:   mergeType,
		room2Batch:  make(map[string]*PushBatch),
		contextChan: make(chan *PushContext, config.GlobalServerConfig.MergerChannelSize),
		timeoutChan: make(chan *PushBatch, config.GlobalServerConfig.MergerChannelSize),
		stopChan:    stopChan,
	}
	go worker.mergeWorkerMain()
	return
}

func (worker *MergeWorker) mergeWorkerMain() {
	var (
		context      *PushContext
		batch        *PushBatch
		timeoutBatch *PushBatch
		existed      bool
		isCreated    bool
		// err          error
	)
	for {
		select {
		case <-worker.stopChan:
			return
		case context = <-worker.contextChan:
			isCreated = false
			// 按房间合并
			if worker.mergeType == types.PUSH_TYPE_ROOM {
				if batch, existed = worker.room2Batch[context.room]; !existed {
					batch = &PushBatch{room: context.room}
					worker.room2Batch[context.room] = batch
					isCreated = true
				}
			} else if worker.mergeType == types.PUSH_TYPE_ALL { // 广播合并
				batch = worker.allBatch
				if batch == nil {
					batch = &PushBatch{}
					worker.allBatch = batch
					isCreated = true
				}
			}

			// 合并消息
			batch.items = append(batch.items, context.msg)

			// 新建批次, 启动超时自动提交
			if isCreated {
				batch.commitTimer = time.AfterFunc(time.Duration(config.GlobalServerConfig.MaxMergerDelay)*time.Millisecond, worker.autoCommit(batch))
			}

			// 批次未满, 继续等待下次提交
			if len(batch.items) < config.GlobalServerConfig.MaxMergerBatchSize {
				continue
			}

			// 批次已满, 取消超时自动提交
			batch.commitTimer.Stop()
		case timeoutBatch = <-worker.timeoutChan:
			if worker.mergeType == types.PUSH_TYPE_ROOM {
				// 定时器触发时, 批次已被提交
				if batch, existed = worker.room2Batch[timeoutBatch.room]; !existed {
					continue
				}

				// 定时器触发时, 前一个批次已提交, 下一个批次已建立
				if batch != timeoutBatch {
					continue
				}
			} else if worker.mergeType == types.PUSH_TYPE_ALL {
				batch = worker.allBatch
				// 定时器触发时, 批次已被提交
				if timeoutBatch != batch {
					continue
				}
			}
		}
		// 提交批次
		err := worker.commitBatch(batch)
		if err != nil {
			logrus.Warn("提交批次失败")
		}
	}
}

func (worker *MergeWorker) autoCommit(batch *PushBatch) func() {
	return func() {
		worker.timeoutChan <- batch
	}
}

func (worker *MergeWorker) commitBatch(batch *PushBatch) (err error) {
	var (
		bizPushData *types.BizPushData
		bizMessage  *types.BizMessage
		buf         []byte
	)

	bizPushData = &types.BizPushData{
		Items: batch.items,
	}
	if buf, err = json.Marshal(*bizPushData); err != nil {
		return
	}

	bizMessage = &types.BizMessage{
		Type: "PUSH",
		Data: json.RawMessage(buf),
	}

	// 打包发送
	if worker.mergeType == types.PUSH_TYPE_ROOM {
		delete(worker.room2Batch, batch.room)
		err = GlobalSocketConnectionManager.PushRoom(batch.room, bizMessage)
	} else if worker.mergeType == types.PUSH_TYPE_ALL {
		worker.allBatch = nil
		err = GlobalSocketConnectionManager.PushAll(bizMessage)
	}
	return
}

func (worker *MergeWorker) pushRoom(room string, msg *json.RawMessage) (err error) {
	var (
		context *PushContext
	)
	context = &PushContext{
		room: room,
		msg:  msg,
	}
	select {
	case worker.contextChan <- context:

	default:
		err = utils.MergeChannelFull
	}
	return
}

func (worker *MergeWorker) pushAll(msg *json.RawMessage) (err error) {
	var (
		context *PushContext
	)
	context = &PushContext{
		msg: msg,
	}
	select {
	case worker.contextChan <- context:

	default:
		err = utils.MergeChannelFull
	}
	return
}
