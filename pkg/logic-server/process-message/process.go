package process_message

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
	"message-center/pkg/configuration"
	ec "message-center/pkg/email-client"
	"message-center/pkg/logic-server/push"
	"message-center/pkg/mongodb"
	"message-center/pkg/mq"
	"message-center/utils"
	"time"
)

type ProcessMessage interface {
	Run()
	Close()
}

type ProcessMessageImpl struct {
	dbController *mongodb.MongoDBController
	mqController *mq.MqController
	emailClient  *ec.EmailClientImpl
	stopChan     chan byte
}

func (p *ProcessMessageImpl) Initialize(dbController *mongodb.MongoDBController, mqController *mq.MqController, emailClient *ec.EmailClientImpl) {
	p.dbController = dbController
	p.mqController = mqController
	p.emailClient = emailClient
	p.stopChan = make(chan byte)
}

func (p *ProcessMessageImpl) Run() {
	t := time.NewTicker(60 * time.Second)
	go func(t *time.Ticker) {
		defer t.Stop()
		for {
			select {
			case <-t.C:
				logrus.Info(fmt.Sprintf("run businsess %s", time.Now().Format("2006-01-02 15:04:05")))
				err := p.processBusiness()
				if err != nil {
					logrus.Info(fmt.Sprintf("process businsess error %s", err.Error()))
				}
			case <-p.stopChan:
				logrus.Info("停止业务处理任务")
				return
			}
		}
	}(t)
}

func (p *ProcessMessageImpl) Close() {
	close(p.stopChan)
}

func (p *ProcessMessageImpl) processBusiness() error {
	session := p.dbController.NewSession()
	defer session.Close()
	c := session.DB(configuration.DB).C("message_center")
	query := bson.M{"known": "unread"}
	query["processed"] = bson.M{"$ne": "processed"}
	mcs := []*MessageCenter{}
	err := c.Find(query).Sort("-create_time").All(&mcs)
	if err != nil {
		return err
	}

	scg := map[string][]*socketMessage{}

	for _, ms := range mcs {
		ms.ProcessedTime = time.Now()
		ms.Processed = "processed"
		if utils.Contains(ms.Channel, "email") {
			logrus.Info(fmt.Sprintf("发送邮件：%s", ms.Subject))
			message := fmt.Sprintf("结果：%s \r\n 链接: %s", ms.Result, ms.Link)
			for _, m := range ms.Emails {
				receive := ec.Receive{
					Ccer:       []string{},
					Recipients: []string{m},
				}
				err := p.emailClient.SendEmail(m, ms.Subject, message, receive)
				if err != nil {
					ms.ProcessedResult += ";" + err.Error()
				}
			}
		}
		if utils.Contains(ms.Channel, "mq") {
			logrus.Info(fmt.Sprintf("推送mq消息：%s", ms.Subject))
			mqMessage, err := json.Marshal(ms)
			if err != nil {
				ms.ProcessedResult += ";" + err.Error()
			}
			err = p.mqController.SendMessage(configuration.TOPIC, mqMessage)
			if err != nil {
				ms.ProcessedResult += ";" + err.Error()
			}
		}
		// 此处处理逻辑为,根据邮箱将消息发往不通渠道
		// 处理数据，将消息按照邮箱分组，并根据消息来源聚合
		if utils.Contains(ms.Channel, "message") {
			logrus.Info(fmt.Sprintf("推送socket消息: %s", ms.Subject))
			if len(ms.Emails) == 0 {
				continue
			}
			for _, em := range ms.Emails {
				if _, ok := scg[em]; ok {
					for _, s := range scg[em] {
						if ms.Source == s.Name {
							s.Count += 1
							if len(s.Data) >= 2 {
								continue
							} else {
								s.Data = append(s.Data, ms)
							}
						}
					}
				} else {
					scg[em] = append(scg[em], &socketMessage{
						Name:  ms.Source,
						Count: 1,
						Data:  []*MessageCenter{ms},
					})

				}
			}

		}

		err = c.UpdateId(ms.Id, ms)
		if err != nil {
			logrus.Info(fmt.Sprintf("更新数据库消息失败：%s-%s", ms.Subject, ms.ProcessedResult))
		}
	}

	// 推送到不同的websocket渠道
	for k, v := range scg {
		// 不采用http请求，而是直接调用方法发送消息
		// 因此要自己进行序列化数据
		var msgArr []json.RawMessage
		v1, err := json.Marshal(v)
		if err != nil {
			logrus.Info(fmt.Sprintf("序列化json数据失败：%s", err))
			continue
		}
		if err = json.Unmarshal(v1, &msgArr); err != nil {
			logrus.Info(fmt.Sprintf("序列化json数据失败：%s", err))
			continue
		}
		err = push.GlobalConnectManager.PushRoom(k, msgArr)
		if err != nil {
			logrus.Info(fmt.Sprintf("推送socket消息失败：%s", err.Error()))
		}
	}
	return nil
}
