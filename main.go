package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"gopkg.in/ini.v1"
	"net/http"
	"os"
	"strings"
	"tesg/pkg/room"
	"tesg/pkg/server"
	"tesg/pkg/utils"
)

var clientMap = make(map[string]*websocket.Conn)

var upgrade = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Header)

	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	userId, ok := r.Header["User-Agent"]
	if ok {
		clientMap[strings.Join(userId, "")] = conn
	}
	fmt.Println(clientMap)

	defer func(conn *websocket.Conn) {
		fmt.Println("断开ws链接")
		err := conn.Close()
		if err != nil {
		}
		delete(clientMap, strings.Join(userId, ""))
		fmt.Println(clientMap)
	}(conn)

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("received message :%s \n", p)
		if err := conn.WriteMessage(messageType, []byte("res from server")); err != nil {
			fmt.Println("WriteMessage error:")
			fmt.Println(err)
			return
		}
	}
}

func main() {
	cfg, err := ini.Load("configs/config.ini")
	if err != nil {
		utils.ErrorF("读取配置文件失败，请检查configs/config.ini内容：%v", err)
		os.Exit(1)
	}
	//bind := cfg.Section("general").Key("bind").String()
	//port := cfg.Section("general").Key("port").String()
	sslCert := cfg.Section("general").Key("cert").String()
	//读取证书Key配置
	sslKey := cfg.Section("general").Key("key").String()
	utils.InfoF(sslCert)
	utils.InfoF(sslKey)

	//http.Handle("/ws", http.HandlerFunc(handleWebSocket))
	//http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	fmt.Fprintf(w, "Hello, World!")
	//})
	rm := room.NewRoomManager()
	wsServer := server.NewP2PServer(rm.InterHandleWebSocket)
	config := server.GetDefaultConfig()
	config.KeyFile = sslKey
	config.CertFile = sslCert
	wsServer.Bind(config)
	//e := http.ListenAndServe(":8080", nil)
	//if e != nil {
	//	return
	//}
}
