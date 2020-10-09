
##### server 负责维护socket长连接协议
 - 接收websocket连接，将连接分配到不同的bucket
 - socket连接分别订阅不同的room,可订阅多个room
 - 接收logic发送过来的数据
 - 定时合并消息，推送到指定room
 - 默认socket端口7777用于socket client连接
 - HTTP内部端口7788，用于logic逻辑推送数据 
 
 - 这两个接口为logic-server调用接口发送服务，内部接口外部不要调用
 ```cassandraql
/push/room 向指定房间推送消息
/push/all 向所有房间推送消息 
```
- 启动服务
```cassandraql
message-server run
```

##### websocket客户端创建连接及维持连接
- 创建连接请求 `ws://127.0.0.1:7777/connect`
- 维持连接，每次60s内发送PING内容: `{"type": "PING"}` 服务端响应`{"type": "PONG"}`
- 收到JOIN则加入ROOM: `{"type": "JOIN", "data": {"room": "chrome-plugin"}}`
- 收到LEAVE则离开ROOM: `{"type": "LEAVE", "data": {"room": "chrome-plugin"}}`


 - 启动
   - 环境变量 CONFIG = "config.json所在目录"
 
 ##### logic为业务逻辑层，在此实现业务逻辑，并推送消息到server
 
 - 处理自身业务逻辑，生产消息
 - 调用server提供的HTTP1接口，将消息发送到server
 - HTTP端口7799用于调用Logic端口推送数据
 
 - 这两个接口为外部服务调用接口/push/room发送消息
 ```cassandraql
/push/room 向指定房间推送消息
/push/all 向所有房间推送消息 
```
- 启动业务服务所需环境变量
```cassandraql
CONFIG_SERVER=http://10.202.81.110:30002/
ENV=dev
```
- 启动服务
```cassandraql
logic-server run
```
- 额外特殊处理逻辑：要增加新的逻辑在pkg/logic-server下创建目录并编写处理逻辑如process-message

 
 
 ##### websocket测试
 
 - http://www.easyswoole.com/wstool.html