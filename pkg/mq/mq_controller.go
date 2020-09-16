package mq

import (
	"errors"
	"fmt"
	"github.com/go-stomp/stomp"
	"github.com/sirupsen/logrus"
	"message-center/pkg/configuration"
)

type MqControllerInterface interface {
	// active 重试连接
	ActiveReConnect() error
	// topic active mq send message
	SendMessage(topic string, message []byte) error
}

type MqController struct {
	connectionStr   string
	activeUsername  string
	activePassword  string
	activeStompConn *stomp.Conn // activemq
}

func (mc *MqController) Initialize(dcl configuration.ConfigurationLoader) {
	MqConnectionStr := dcl.GetField("DevOps.Mgmt.API", "MQ_ConnectionStr")
	MqUsername := dcl.GetField("DevOps.Mgmt.API", "MQ_Username")
	MqPassword := dcl.GetField("DevOps.Mgmt.API", "MQ_Password")
	mc.InitializeByConfig(MqConnectionStr, MqUsername, MqPassword)
	//mc.InitializeByConfig("10.200.50.67:61613")
}

func (mc *MqController) InitializeByConfig(connectionStr, username, password string) {
	mc.connectionStr = connectionStr
	mc.activeUsername = username
	mc.activePassword = password
	_ = mc.ActiveReConnect()
}

func (mc *MqController) ActiveReConnect() error {
	var err error
	mc.activeStompConn, err = stomp.Dial("tcp", mc.connectionStr,
		stomp.ConnOpt.Login(mc.activeUsername, mc.activePassword),
		stomp.ConnOpt.AcceptVersion(stomp.V11),
		stomp.ConnOpt.AcceptVersion(stomp.V12),
		// stomp.ConnOpt.Host("dragon"),
		// stomp.ConnOpt.Header("nonce", "B256B26D320A")
	)
	return err
}

func (mc *MqController) SendMessage(topic string, message []byte) error {
	err := mc.ActiveReConnect()
	if mc.activeStompConn == nil || err != nil {
		logrus.Errorf(fmt.Sprintf("连接activemq失败：%s-%s-%s", mc.connectionStr, mc.activeUsername, mc.activePassword))
		return errors.New("连接activemq失败")
	}
	defer mc.activeStompConn.Disconnect()
	err = mc.activeStompConn.Send(
		"/topic/"+topic,
		"text/plain",
		message)
	if err != nil {
		return err
	}
	return nil
}
