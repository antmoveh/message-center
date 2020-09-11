package web_socket

import (
	"message-center/cmd/message/config"
	"message-center/pkg/types"
	"message-center/utils"
)

type SocketConnectManager interface {
	// 加入socket连接
	AddConn(connection *WSConnection)
	// 删除socket连接
	DelConn(connection *WSConnection)
	// 加入房间
	JoinRoom(roomId string, connection *WSConnection) error
	// 离开房间
	LeaveRoom(roomId string, connection *WSConnection) error
	// 向指定房间推送消息
	PushRoom(roomId string, message *types.BizMessage) error
	// 向所有连接推送消息
	PushAll(message *types.BizMessage) error
	// 获取桶
	GetBucket(connection *WSConnection) *Bucket
	// 关闭
	ConnectManagerClose()
}

// 推送任务
type PushJob struct {
	pushType int               // 推送类型
	roomId   string            // 房间ID
	bizMsg   *types.BizMessage // 未序列化的业务消息
	wsMsg    *types.WSMessage  // 已序列化的业务消息
}

// 建立的socket连接管理器，负责检查连接是否存活
// 根据配置buckets数量，初始化桶
// 根据配置job数量，初始化job
type ConnectionManager struct {
	buckets      []*Bucket
	jobChan      []chan *PushJob // 每个Bucket对应一个Job Queue
	dispatchChan chan *PushJob   // 待分发消息队列
	stopChan     chan byte       // 关闭
}

// 初始化Buckets、job
func InitConnectManager() error {
	var (
		bucketIdx         int
		jobWorkerIdx      int
		dispatchWorkerIdx int
		connMgr           *ConnectionManager
	)

	connMgr = &ConnectionManager{
		buckets:      make([]*Bucket, config.GlobalServerConfig.BucketCount),
		jobChan:      make([]chan *PushJob, config.GlobalServerConfig.BucketCount),
		dispatchChan: make(chan *PushJob, config.GlobalServerConfig.DispatchChannelSize),
		stopChan:     make(chan byte, 1),
	}
	for bucketIdx, _ = range connMgr.buckets {
		connMgr.buckets[bucketIdx] = InitBucket(bucketIdx)                                               // 初始化Bucket
		connMgr.jobChan[bucketIdx] = make(chan *PushJob, config.GlobalServerConfig.BucketJobChannelSize) // Bucket的Job队列
		// 为每个Buckets启动job，负责监听channel，然后推送消息
		for jobWorkerIdx = 0; jobWorkerIdx < config.GlobalServerConfig.BucketJobWorkerCount; jobWorkerIdx++ {
			go connMgr.jobWorkerMain(jobWorkerIdx, bucketIdx)
		}
	}
	GlobalSocketConnectionManager = connMgr
	// 初始化分发协程, 用于将消息扇出给各个Bucket
	for dispatchWorkerIdx = 0; dispatchWorkerIdx < config.GlobalServerConfig.DispatchWorkerCount; dispatchWorkerIdx++ {
		go connMgr.dispatchWorkerMain(dispatchWorkerIdx)
	}
	return nil
}

func (connMgr *ConnectionManager) GetBucket(wsConnection *WSConnection) (bucket *Bucket) {
	bucket = connMgr.buckets[wsConnection.connId%uint64(len(connMgr.buckets))]
	return
}

func (connMgr *ConnectionManager) AddConn(wsConnection *WSConnection) {
	var (
		bucket *Bucket
	)

	bucket = connMgr.GetBucket(wsConnection)
	bucket.AddConn(wsConnection)

}

func (connMgr *ConnectionManager) DelConn(wsConnection *WSConnection) {
	var (
		bucket *Bucket
	)

	bucket = connMgr.GetBucket(wsConnection)
	bucket.DelConn(wsConnection)
}

func (connMgr *ConnectionManager) JoinRoom(roomId string, wsConn *WSConnection) (err error) {
	var (
		bucket *Bucket
	)

	bucket = connMgr.GetBucket(wsConn)
	err = bucket.JoinRoom(roomId, wsConn)
	return
}

func (connMgr *ConnectionManager) LeaveRoom(roomId string, wsConn *WSConnection) (err error) {
	var (
		bucket *Bucket
	)

	bucket = connMgr.GetBucket(wsConn)
	err = bucket.LeaveRoom(roomId, wsConn)
	return
}

// 向所有在线用户发送消息
func (connMgr *ConnectionManager) PushAll(bizMsg *types.BizMessage) (err error) {
	var (
		pushJob *PushJob
	)

	pushJob = &PushJob{
		pushType: types.PUSH_TYPE_ALL,
		bizMsg:   bizMsg,
	}

	select {
	case connMgr.dispatchChan <- pushJob:
	default:
		err = utils.DisPatchChannelFull
	}
	return
}

// 向指定房间发送消息
func (connMgr *ConnectionManager) PushRoom(roomId string, bizMsg *types.BizMessage) (err error) {
	var (
		pushJob *PushJob
	)

	pushJob = &PushJob{
		pushType: types.PUSH_TYPE_ROOM,
		bizMsg:   bizMsg,
		roomId:   roomId,
	}

	select {
	case connMgr.dispatchChan <- pushJob:
	default:
		err = utils.DisPatchChannelFull
	}
	return
}

// 消息分发到Bucket
func (connMgr *ConnectionManager) dispatchWorkerMain(dispatchWorkerIdx int) {
	var (
		bucketIdx int
		pushJob   *PushJob
		err       error
	)
	for {
		select {
		case <-connMgr.stopChan:
			return
		case pushJob = <-connMgr.dispatchChan:

			// 序列化
			if pushJob.wsMsg, err = types.EncodeWSMessage(pushJob.bizMsg); err != nil {
				continue
			}
			// 分发给所有Bucket, 若Bucket拥塞则等待
			for bucketIdx, _ = range connMgr.buckets {
				connMgr.jobChan[bucketIdx] <- pushJob
			}
		}
	}
}

// Job负责消息广播给客户端
func (connMgr *ConnectionManager) jobWorkerMain(jobWorkerIdx int, bucketIdx int) {
	var (
		bucket  = connMgr.buckets[bucketIdx]
		pushJob *PushJob
	)

	for {
		select {
		case <-connMgr.stopChan:
			return
		case pushJob = <-connMgr.jobChan[bucketIdx]: // 从Bucket的job queue取出一个任务
			if pushJob.pushType == types.PUSH_TYPE_ALL {
				bucket.PushAll(pushJob.wsMsg)
			} else if pushJob.pushType == types.PUSH_TYPE_ROOM {
				bucket.PushRoom(pushJob.roomId, pushJob.wsMsg)
			}
		}
	}
}

func (connMgr *ConnectionManager) ConnectManagerClose() {
	connMgr.buckets = nil
	close(connMgr.dispatchChan)
	close(connMgr.stopChan)
}
