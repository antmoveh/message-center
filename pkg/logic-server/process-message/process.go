package process_message

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
	"message-center/pkg/configuration"
	ec "message-center/pkg/email-client"
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
	t := time.NewTicker(10 * time.Second)
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
	query := bson.M{"Known": "unread"}
	query["processed"] = bson.M{"$ne": "processed"}
	mcs := []*MessageCenter{}
	err := c.Find(query).Sort("-create_time").All(&mcs)
	if err != nil {
		return err
	}

	sm := []*socketMessage{}

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
		if utils.Contains(ms.Channel, "message") {
			logrus.Info(fmt.Sprintf("推送socket消息: %s", ms.Subject))
			for _, s := range sm {
				if ms.Source == s.Name {
					s.Count += 1
					if len(s.Data) >= 2 {
						continue
					} else {
						s.Data = append(s.Data, ms)
					}
				}
			}
		}

		err = c.UpdateId(ms.Id, ms)
		if err != nil {
			logrus.Info(fmt.Sprintf("更新数据库消息失败：%s-%s", ms.Subject, ms.ProcessedResult))
		}
	}

	// 推送websocket消息
	if len(sm) > 0 {

	}

	return nil
}
