package message_server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"message-center/cmd/message/config"
	"message-center/pkg/message-server/web-socket"
	"message-center/utils"
	"net"
	"net/http"
	"strconv"
	"time"
)

type HttpService struct {
	server *http.Server
}

var GlobalHttpServer *HttpService

func InitHttpService() error {
	var (
		mux      *http.ServeMux
		server   *http.Server
		listener net.Listener
		err      error
	)

	// 路由
	mux = http.NewServeMux()
	mux.HandleFunc("/push/all", handlePushAll)
	mux.HandleFunc("/push/room", handlePushRoom)

	// HTTP/2 TLS服务
	server = &http.Server{
		ReadTimeout:  time.Duration(config.GlobalServerConfig.ServiceReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(config.GlobalServerConfig.ServiceWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	// 监听端口
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(config.GlobalServerConfig.ServicePort)); err != nil {
		return err
	}

	// 赋值全局变量
	GlobalHttpServer = &HttpService{
		server: server,
	}
	if config.GlobalServerConfig.ServerPem != "" && config.GlobalServerConfig.ServerKey != "" {
		// 验证tls证书是否合法
		if _, err := tls.LoadX509KeyPair(config.GlobalServerConfig.ServerPem, config.GlobalServerConfig.ServerKey); err != nil {
			return utils.CertInvalid
		}
		go server.ServeTLS(listener, config.GlobalServerConfig.ServerPem, config.GlobalServerConfig.ServerKey)
		return nil
	}

	go server.Serve(listener)

	return nil
}

// 全量推送POST msg={}
func handlePushAll(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		items  string
		msgArr []json.RawMessage
		msgIdx int
	)
	if err = req.ParseForm(); err != nil {
		return
	}

	items = req.PostForm.Get("items")
	if err = json.Unmarshal([]byte(items), &msgArr); err != nil {
		return
	}

	for msgIdx, _ = range msgArr {
		_ = web_socket.GlobalMessageMergeServer.PushAll(&msgArr[msgIdx])
	}
}

// 房间推送POST room=xxx&msg
func handlePushRoom(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		room   string
		items  string
		msgArr []json.RawMessage
		msgIdx int
	)
	if err = req.ParseForm(); err != nil {
		return
	}

	room = req.PostForm.Get("room")
	items = req.PostForm.Get("items")

	if err = json.Unmarshal([]byte(items), &msgArr); err != nil {
		return
	}

	for msgIdx, _ = range msgArr {
		_ = web_socket.GlobalMessageMergeServer.PushRoom(room, &msgArr[msgIdx])
	}
}

func HttpServerClose() {
	_ = GlobalHttpServer.server.Shutdown(context.TODO())
}
