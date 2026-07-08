package cmds

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"combine.com/mahjong/engine"
	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
	"combine.com/utils/types"
)

var (
	eng *engine.Engine
)

func Inject(eng_ *engine.Engine) {
	eng = eng_
}

func GetCmds() []frame.Cmd {
	var cmds = []frame.Cmd{
		frame.NewCmd("usercount", "玩家数量", usercount),
		frame.NewCmd("userimage", "玩家镜像", userimage),
		frame.NewCmd("baduser", "坏玩家", baduser),
		frame.NewCmd("clearbaduser", "清除坏玩家", clearbaduser),
		frame.NewCmd("forcedeleteuser", "强制删除玩家", forcedeleteuser),
		frame.NewCmd("roomcount", "房间数量", roomcount),
		frame.NewCmd("roominfo", "房间信息", roominfo),
		frame.NewCmd("roomimage", "房间镜像", roomimage),
		frame.NewCmd("releaseroom", "释放房间", releaseroom),
		frame.NewCmd("badroom", "坏房间", badroom),
	}
	return cmds
}

func usercount(cmdVal string) {
	logger.Infof("玩家数量:%d", engine.UserManage.GetUserCount())
}

func userimage(commVal string) {
	user := engine.UserManage.GetUser(common.ParseInt(commVal))
	if user != nil {
		logger.Infof(user.GetUserImage())
	}
}

//强制删除一个玩家
func forcedeleteuser(commVal string) {
	engine.UserManage.ForceDeleteUser(common.ParseInt(commVal))
}

func baduser(commVal string) {
	users := engine.UserManage.GetAllUsers()
	buff := bytes.Buffer{}
	buff.WriteString("{\n")
	for _, user := range users {
		room := user.GetRoom()
		if room != nil && engine.RoomManage.GetRoom(room.GetRoomID()) == nil {
			buff.WriteString(fmt.Sprintf("%d,", user.GetMemberid()))
		}
	}
	buff.WriteString("\n}")
	logger.Infof(buff.String())
}

func clearbaduser(commVal string) {
	users := engine.UserManage.GetAllUsers()
	buff := bytes.Buffer{}
	buff.WriteString("{\n")
	for _, user := range users {
		room := user.GetRoom()
		if room != nil && engine.RoomManage.GetRoom(room.GetRoomID()) == nil {
			forcedeleteuser(strconv.Itoa(user.GetMemberid()))
			buff.WriteString(fmt.Sprintf("%d,", user.GetMemberid()))
		}
	}
	buff.WriteString("\n}")
	logger.Infof(buff.String())
}

func roomcount(cmdVal string) {
	logger.Infof("房间数量:%d", len(engine.RoomManage.GetRooms()))
}

func roominfo(commVal string) {
	engine.RoomManage.Lock.Lock()
	rooms := engine.RoomManage.GetRooms()
	buff := bytes.Buffer{}
	buff.WriteString("{\n")
	roomcount := 0
	if commVal == "" {
		for _, room := range rooms {
			roomcount++
			buff.WriteString(fmt.Sprintf("     id:%s,创建时间:[%s]\n", room.GetRoomID(), room.GetCreateTime()))
		}
	} else {
		for _, room := range rooms {
			roomcount++
			buff.WriteString(fmt.Sprintf("     id:%s,创建时间:[%s],玩家信息:%s\n", room.GetRoomID(), room.GetCreateTime(), room.GetAllUsersInfo()))
		}
	}
	buff.WriteString("}")
	logger.Infof(buff.String())
	engine.RoomManage.Lock.Unlock()
	logger.Infof("房间数量:%d", roomcount)
}

func releaseroom(commVal string) {
	roomid := commVal
	if engine.RoomManage.ReleaseRoom(roomid) {
		//解散房间
		engine.MatchEndBuild(types.KVS{frame.ROOMID: roomid, frame.ENDMODE: frame.ENDMODE_DISSOLVE})
	}
}

func roomimage(commVal string) {
	room := engine.RoomManage.GetRoom(commVal)
	if room != nil {
		logger.Infof(room.GetRoomImage())
	}
}

//查询坏房
func badroom(commVal string) {
	engine.RoomManage.Lock.Lock()
	rooms := engine.RoomManage.GetRooms()
	buff := bytes.Buffer{}
	buff.WriteString("{\n")
	for _, room := range rooms {
		offlineCount := 0
		for _, u := range room.GetUsers() {
			if u != nil && !u.GetOnline() {
				offlineCount++
			}
		}
		createTime, _ := common.ParseTime(room.GetCreateTime())
		if offlineCount == frame.GetPCount(engine.PLATFORM) || time.Now().Sub(createTime) > time.Hour*24 {
			buff.WriteString(fmt.Sprintf("     id:%s,创建时间:[%s],玩家信息:%s\n", room.GetRoomID(), room.GetCreateTime(), room.GetAllUsersInfo()))
		}
	}
	buff.WriteString("}")
	logger.Infof(buff.String())
	engine.RoomManage.Lock.Unlock()
}
