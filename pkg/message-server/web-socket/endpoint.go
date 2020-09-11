package web_socket

import (
	"context"
	"github.com/gorilla/websocket"
	"message-center/cmd/message/config"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// 	WebSocket服务端
type SocketEndpoint struct {
	server    *http.Server
	curConnId uint64
}

var (
	wsUpgrader = websocket.Upgrader{
		// 允许所有CORS跨域请求
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// 创建连接，客户端通过发起ws://0.0.0.0：7777/connect请求创建一个新的socket连接
func InitSocketEndpoint() error {
	var (
		mux      *http.ServeMux
		server   *http.Server
		listener net.Listener
		err      error
	)

	// 路由
	mux = http.NewServeMux()
	mux.HandleFunc("/connect", handleConnect)

	// HTTP服务
	server = &http.Server{
		ReadTimeout:  time.Duration(config.GlobalServerConfig.WsReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(config.GlobalServerConfig.WsWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	// 监听端口
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(config.GlobalServerConfig.WsPort)); err != nil {
		return err
	}

	// 赋值全局变量
	ws := &SocketEndpoint{
		server:    server,
		curConnId: uint64(time.Now().Unix()),
	}

	GlobalSocketEndpoint = ws

	// 启动socket连接服务
	go server.Serve(listener)

	return nil
}

func handleConnect(resp http.ResponseWriter, req *http.Request) {
	var (
		err      error
		wsSocket *websocket.Conn
		connId   uint64
		wsConn   *WSConnection
	)

	// WebSocket握手
	if wsSocket, err = wsUpgrader.Upgrade(resp, req, nil); err != nil {
		return
	}

	// 为每个连接创建唯一ID标识
	connId = atomic.AddUint64(&GlobalSocketEndpoint.curConnId, 1)

	// 初始化WebSocket的读写协程
	wsConn = InitWSConnection(connId, wsSocket)

	// 开始处理websocket消息
	wsConn.WSHandle()
}

func SocketConnectClose() {
	_ = GlobalSocketEndpoint.server.Shutdown(context.TODO())
}
