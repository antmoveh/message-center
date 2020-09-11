package push

import (
	"context"
	"encoding/json"
	"message-center/cmd/logic/config"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Service struct {
	server *http.Server
}

func InitHttpService() (err error) {
	var (
		mux      *http.ServeMux
		server   *http.Server
		listener net.Listener
	)

	// 路由
	mux = http.NewServeMux()
	mux.HandleFunc("/push/all", handlePushAll)
	mux.HandleFunc("/push/room", handlePushRoom)
	// mux.HandleFunc("/stats", handleStats)

	// HTTP/1服务
	server = &http.Server{
		ReadTimeout:  time.Duration(config.GlobalLogicConfig.ServiceReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(config.GlobalLogicConfig.ServiceWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	// 监听端口
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(config.GlobalLogicConfig.ServicePort)); err != nil {
		return
	}
	GlobalHttpServer = &Service{
		server: server,
	}

	// 拉起服务
	go server.Serve(listener)

	return
}

// 全量推送POST msg={}
func handlePushAll(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		items  string
		msgArr []json.RawMessage
	)
	if err = req.ParseForm(); err != nil {
		return
	}

	items = req.PostForm.Get("items")
	if err = json.Unmarshal([]byte(items), &msgArr); err != nil {
		return
	}

	_ = GlobalConnectManager.PushAll(msgArr)
}

// 房间推送POST room=xxx&msg
func handlePushRoom(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		room   string
		items  string
		msgArr []json.RawMessage
	)
	if err = req.ParseForm(); err != nil {
		return
	}

	room = req.PostForm.Get("room")
	items = req.PostForm.Get("items")

	if err = json.Unmarshal([]byte(items), &msgArr); err != nil {
		return
	}

	_ = GlobalConnectManager.PushRoom(room, msgArr)
}

func HttpServerClose() {
	_ = GlobalHttpServer.server.Shutdown(context.TODO())
}
