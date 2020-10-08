package process_message

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

// 消息中心结构体
type MessageCenter struct {
	Id              *bson.ObjectId `bson:"_id" json:"id"`
	TenantId        *bson.ObjectId `json:"tenant_id" bson:"tenant_id"`
	Subject         string         `json:"subject" bson:"subject"`                   // 消息主题
	Result          string         `json:"result" bson:"result"`                     // 结果 success/failed/pass/reject/complete
	Link            string         `json:"link" bson:"link"`                         // 链接
	Known           string         `json:"known" bson:"known"`                       // 是否已读 read/unread
	Emails          []string       `json:"emails" bson:"emails"`                     // 接收消息列表
	Channel         []string       `json:"channel" bson:"channel"`                   // 推送消息渠道，可填写多个 email/mq/message
	Source          string         `json:"source" bson:"source"`                     // 消息来源pipeline/alarm/tenant/domain/service/openapi
	SourceCH        string         `json:"source_ch" bson:"source_ch"`               // 消息来源中文名称
	Type            string         `json:"type" bson:"type"`                         // 消息类型，流水线pipeline/告警alarm/订阅subscribe/申请apply/审核audit/通知inform
	CreateTime      time.Time      `json:"create_time" bson:"create_time"`           // 消息创建时间
	Processed       string         `json:"processed" bson:"processed"`               // 该消息是否已处理
	ProcessedResult string         `json:"processed_result" bson:"processed_result"` // 处理结果
	ProcessedTime   time.Time      `json:"processed_time" bson:"processed_time"`     // 处理时间
}

type socketMessage struct {
	Name  string           `json:"name"`
	Count int              `json:"count"`
	Data  []*MessageCenter `json:"data"`
}
