package server

import (
	"errors"
	"github.com/chuckpreslar/emission"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"tesg/pkg/utils"
	"time"
)

type WebSocketConn struct {
	emission.Emitter
	socket *websocket.Conn
	mutex  *sync.Mutex
	closed bool
}

func (conn *WebSocketConn) ReadMessage() {
	in := make(chan []byte)
	stop := make(chan struct{})
	pingTicker := time.NewTicker(pingPeriod)

	c := conn.socket
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				utils.WarnF("[ReadMessage]获取到错误：%v", err)
				var c *websocket.CloseError
				k := errors.As(err, &c)
				if k {
					conn.Emit("close", c.Code, c.Text)
				} else {
					var c *net.OpError
					k := errors.As(err, &c)
					if k {
						conn.Emit("close", 1001, c.Error())
					}
				}
				close(stop)
				break
			}
			in <- message
		}
	}()

	for {
		select {
		case _ = <-pingTicker.C:
			//utils.InfoF("发送心跳包...")
			//heartPackage := map[string]interface{}{
			//	"type": "heartPackage",
			//	"data": " ",
			//}
			//str := utils.MapToJson(heartPackage)
			//err := conn.Send(str)
			//if err != nil {
			//	utils.ErrorF("发送心跳包时遇到错误")
			//	pingTicker.Stop()
			//	return
			//}
		case message := <-in:
			{
				utils.InfoF("接收到数据：%s", message)
				conn.Emit("message", []byte(message))
			}
		case <-stop:
			return
		}
	}
}

func (conn *WebSocketConn) Send(message string) error {
	if !strings.HasSuffix(message, "heartPackage\"}") {
		utils.InfoF("发送数据: %s", message)
	}
	//连接加锁
	conn.mutex.Lock()
	//延迟执行连接解锁
	defer conn.mutex.Unlock()
	//判断连接是否关闭
	if conn.closed {
		return errors.New("websocket: write closed")
	}
	//发送消息
	return conn.socket.WriteMessage(websocket.TextMessage, []byte(message))
}

// 发送心跳包的间隔时间
const pingPeriod = 5 * time.Second

func NewWebSocketConn(socket *websocket.Conn) *WebSocketConn {
	var conn WebSocketConn
	conn.Emitter = *emission.NewEmitter()
	conn.socket = socket
	conn.mutex = new(sync.Mutex)
	conn.closed = false
	// socket关闭回调
	conn.socket.SetCloseHandler(func(code int, text string) error {
		utils.WarnF("%s [%d]", text, code)
		conn.Emit("close", code, text)
		conn.closed = true
		return nil
	})
	return &conn
}

// 服务配置
type P2PServerConfig struct {
	//IP
	Host string
	//端口
	Port int
	//Cert文件
	CertFile string
	//Key文件
	KeyFile string
	//WebSocket路径
	WebSocketPath string
}

func GetDefaultConfig() P2PServerConfig {
	return P2PServerConfig{
		Host:          "0.0.0.0",
		Port:          8080,
		WebSocketPath: "/ws",
	}
}

// P2P服务
type P2PServer struct {
	//WebSocket绑定函数,由信令服务处理
	handleWebSocket func(ws *WebSocketConn, request *http.Request)
	//Websocket升级为长连接
	upgrader websocket.Upgrader
}

func NewP2PServer(wsHandler func(ws *WebSocketConn, request *http.Request)) *P2PServer {
	server := &P2PServer{
		handleWebSocket: wsHandler,
	}
	server.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return server
}

func (server *P2PServer) handlerWebSocketRequest(writer http.ResponseWriter, request *http.Request) {
	utils.InfoF("in ws")
	conn, err := server.upgrader.Upgrade(writer, request, http.Header{
		"connect": []string{"test"},
	})
	if err != nil {
		utils.PanicF("%v", err)
		return
	}
	wsTransport := NewWebSocketConn(conn)
	server.handleWebSocket(wsTransport, request)
	wsTransport.ReadMessage()
}

func (server *P2PServer) Bind(config P2PServerConfig) {
	http.HandleFunc(config.WebSocketPath, server.handlerWebSocketRequest)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test peer 2 peer server"))
	})
	utils.InfoF("P2P server listening on :%s:%d", config.Host, config.Port)
	//e := http.ListenAndServe(":8080", nil)
	//if e != nil {
	//	return
	//}
	panic(http.ListenAndServeTLS(":"+strconv.Itoa(config.Port), config.CertFile, config.KeyFile, nil))
}
