package room

import (
	"net/http"
	"strings"
	"tesg/pkg/server"
	"tesg/pkg/utils"
)

const (
	JoinRoom       = "joinRoom"       // 加入房间
	Offer          = "offer"          // Offer消息
	Answer         = "answer"         // Answer消息
	Candidate      = "candidate"      // Cadidate消息
	HangUp         = "hangUp"         // 挂断
	LeaveRoom      = "leaveRoom"      // 离开房间
	IsFull         = "isFull"         // 离开房间
	UpdateUserList = "updateUserList" // 更新房间用户列表
)

type RoomMananger struct {
	rooms map[string]*Room
}

func NewRoomManager() *RoomMananger {
	return &RoomMananger{
		rooms: make(map[string]*Room),
	}
}

type Room struct {
	users    map[string]User
	sessions map[string]Session
	Id       string
}

func NewRoom(id string) *Room {
	return &Room{
		users:    map[string]User{},
		sessions: map[string]Session{},
		Id:       id,
	}
}

func (rm *RoomMananger) GetRoom(id string) *Room {
	return rm.rooms[id]
}

func (rm *RoomMananger) createRoom(id string) *Room {
	room := rm.rooms[id]
	room = NewRoom(id)
	return room
}

func (rm *RoomMananger) deleteRoom(id string) {
	delete(rm.rooms, id)
}

func (rm *RoomMananger) InterHandleWebSocket(conn *server.WebSocketConn, request *http.Request) {
	utils.InfoF("有新用户链接成功")
	conn.On("message", func(message []byte) {
		jsonValue, err := utils.JsonStrToStruct(string(message))
		if err != nil {
			utils.ErrorF("解析JSOn出错 %v", err)
			return
		}
		var data map[string]interface{}
		temp, ok := jsonValue["data"]
		if !ok {
			utils.ErrorF("没有找到数据 %v", data)
			return
		}
		data = temp.(map[string]interface{})

		var roomId string
		if val, ok := data["roomId"]; ok {
			if roomId, ok = val.(string); !ok {
				utils.ErrorF("房间Id类型错误：%v", val)
				return
			}
		}
		// 根据房间id查找房间
		room := rm.GetRoom(roomId)
		if room == nil {
			room = rm.createRoom(roomId)
			rm.rooms[roomId] = room
			utils.WarnF("房间[%v]不存在，已创建房间", roomId)
		} else {
			//if len(room.users) >= 2 {
			//	utils.WarnF("房间[%v]已满人", roomId)
			//	data := map[string]interface{}{
			//		"type": IsFull,
			//		"data": map[string]interface{}{},
			//	}
			//	conn.Send(utils.MapToJson(data))
			//	return
			//} else {
			utils.InfoF("房间[%v]已存在，可供加入", roomId)
			//}
		}
		utils.InfoF("房间列表[%v]", rm.rooms)
		switch jsonValue["type"] {
		case JoinRoom:
			onJoinRoom(conn, data, room, rm)
			break
		case Offer:
			fallthrough
		case Answer:
			fallthrough
		case Candidate:
			onCandidate(conn, data, room, jsonValue)
		case HangUp:
			onHangUP(conn, data, room, jsonValue)
		default:
			utils.WarnF("遇到了服务器未知的请求 %v", jsonValue)
		}
	})

	conn.On("close", func(code int, text string) {
		onClose(conn, rm)
	})
}

func onHangUP(conn *server.WebSocketConn, data map[string]interface{}, room *Room, value map[string]interface{}) {
	sessionId := data["sessionId"].(string)
	ids := strings.Split(sessionId, "-")
	if user, ok := room.users[ids[0]]; !ok {
		utils.WarnF("用户 【'%v'】没有找到", ids[0])
		return
	} else {
		hangupData := map[string]interface{}{
			"type": HangUp,
			"data": map[string]interface{}{
				"to":        ids[0],
				"sessionId": sessionId,
			},
		}
		user.conn.Send(utils.MapToJson(hangupData))
	}

	if user, ok := room.users[ids[1]]; !ok {
		utils.WarnF("用户 【'%v'】没有找到", ids[1])
		return
	} else {
		hangupData := map[string]interface{}{
			"type": HangUp,
			"data": map[string]interface{}{
				"to":        ids[1],
				"sessionId": sessionId,
			},
		}
		user.conn.Send(utils.MapToJson(hangupData))
	}
}

// offer answer candidate公用一个，目的只有转发，基本上不做数据处理
func onCandidate(conn *server.WebSocketConn, data map[string]interface{}, room *Room, value map[string]interface{}) {
	utils.InfoF("接收到的内容：%v", data)
	to := data["to"].(string)
	user, ok := room.users[to]
	if !ok {
		utils.ErrorF("目标用户不存在 id:[%v]", to)
		return
	}
	user.conn.Send(utils.MapToJson(data))
}

func (rm *RoomMananger) notifyUsersUpdate(conn *server.WebSocketConn, users map[string]User) {
	var userInfos []UserInfo
	for _, clientUser := range users {
		userInfos = append(userInfos, clientUser.info)
	}
	sendData := map[string]interface{}{}
	sendData["type"] = UpdateUserList
	sendData["data"] = userInfos
	for _, user := range users {
		// 通知当前房间的所有用户，人数发生变化
		user.conn.Send(utils.MapToJson(sendData))
	}
}

func onJoinRoom(conn *server.WebSocketConn, data map[string]interface{}, room *Room, rm *RoomMananger) {
	user := User{
		conn: conn,
		info: UserInfo{
			ID:   data["id"].(string),
			Name: data["name"].(string),
		},
	}
	room.users[user.info.ID] = user
	utils.InfoF("📢【%v】", room.users)
	utils.InfoF("📢通知：用户【%v】将加入房间【%v】", user.info.Name, room.Id)
	rm.notifyUsersUpdate(conn, room.users)
}

func onClose(conn *server.WebSocketConn, rm *RoomMananger) {
	utils.InfoF("链接关闭 %v", conn)

	var userId, roomId = "", ""
	for _, room := range rm.rooms {
		for _, user := range room.users {
			if user.conn == conn {
				userId = user.info.ID
				roomId = room.Id
			}
		}
	}
	if userId == "" {
		utils.WarnF("没有找到退出房间的用户和房间")
		return
	}
	for _, user := range rm.GetRoom(roomId).users {
		if user.conn != conn {

		}
	}
	delete(rm.GetRoom(roomId).users, userId)
	//if len(rm.GetRoom(roomId).users) <= 1 {
	//	delete(rm.rooms, roomId)
	//}
	rm.notifyUsersUpdate(conn, rm.GetRoom(roomId).users)
}
