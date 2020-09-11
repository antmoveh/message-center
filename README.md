

##### server 负责维护socket长连接协议
 - 接收websocket连接，将连接分配到不同的bucket
 - socket连接分别订阅不同的room,可订阅多个room
 - 接收logic发送过来的数据
 - 定时合并消息，推送到指定room
 - 默认socket端口7777用于socket client连接
 - HTTP内部端口7788，用于logic逻辑推送数据  

 - 启动
   - 环境变量 CONFIG = "config.json所在目录"
 
 ##### logic为业务逻辑层，在此实现业务逻辑，并推送消息到server
 
 - 处理自身业务逻辑，生产消息
 - 调用server提供的HTTP1接口，将消息发送到server
 - HTTP端口7799用于调用Logic端口推送数据
 
 ##### websocket测试
 
 - http://www.easyswoole.com/wstool.html