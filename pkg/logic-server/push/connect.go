package push

import (
	"crypto/tls"
	"github.com/prometheus/common/log"
	"golang.org/x/net/http2"
	"message-center/cmd/logic/config"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type pushInterface interface {
	PushAll(itemsJson []byte) error
	PushRoom(room string, itemsJson []byte) error
}

// 与消息服之间的通讯
type ServerConn struct {
	schema string
	client *http.Client // 内置长连接+并发连接数
}

func InitMessageServerConn(gatewayConfig *config.MessageServerConfig) (serverConn *ServerConn, err error) {
	var (
		transport *http.Transport
	)

	serverConn = &ServerConn{
		schema: "http://" + gatewayConfig.Hostname + ":" + strconv.Itoa(gatewayConfig.Port),
	}

	transport = &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // 不校验服务端证书
		MaxIdleConns:        config.GlobalLogicConfig.MessageServerMaxConnection,
		MaxIdleConnsPerHost: config.GlobalLogicConfig.MessageServerMaxConnection,
		IdleConnTimeout:     time.Duration(config.GlobalLogicConfig.MessageServerIdleTimeout) * time.Second, // 连接空闲超时
	}
	// 启动HTTP/2协议
	http2.ConfigureTransport(transport)

	// HTTP/2 客户端
	serverConn.client = &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.GlobalLogicConfig.MessageServerTimeout) * time.Millisecond, // 请求超时
	}
	return
}

// 出于性能考虑, 消息数组在此前已经编码成json
func (serverConn *ServerConn) PushAll(itemsJson []byte) (err error) {
	var (
		apiUrl string
		form   url.Values
		resp   *http.Response
		retry  int
	)

	apiUrl = serverConn.schema + "/push/all"

	form = url.Values{}
	form.Set("items", string(itemsJson))

	for retry = 0; retry < config.GlobalLogicConfig.MessageServerPushRetry; retry++ {
		if resp, err = serverConn.client.PostForm(apiUrl, form); err != nil {
			log.Warn("向message server发送消息失败：" + err.Error())
			continue
		}
		resp.Body.Close()
		break
	}
	return
}

// 出于性能考虑, 消息数组在此前已经编码成json
func (serverConn *ServerConn) PushRoom(room string, itemsJson []byte) (err error) {
	var (
		apiUrl string
		form   url.Values
		resp   *http.Response
		retry  int
	)

	apiUrl = serverConn.schema + "/push/room"

	form = url.Values{}
	form.Set("room", room)
	form.Set("items", string(itemsJson))

	for retry = 0; retry < config.GlobalLogicConfig.MessageServerPushRetry; retry++ {
		if resp, err = serverConn.client.PostForm(apiUrl, form); err != nil {
			log.Warn("向message server发送消息失败：" + err.Error())
			continue
		}
		resp.Body.Close()
		break
	}
	return
}
