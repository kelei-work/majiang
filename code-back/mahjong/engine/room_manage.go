/*
房间管理
*/

package engine

import (
	"strconv"
	"sync"
	"time"

	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/types"
)

type roomManage struct {
	rooms map[string]*Room //房间列表
	Lock  sync.Mutex
}

func ctor() *roomManage {
	RoomManage := roomManage{}
	RoomManage.rooms = make(map[string]*Room)
	return &RoomManage
}

var (
	RoomManage = *ctor()
	_          = RoomManage.Init()
)

//初始化
func (r *roomManage) Init() int {
	return 1
}

//获取所有的房间
func (r *roomManage) GetRooms() map[string]*Room {
	return r.rooms
}

//创建一个房间
func (r *roomManage) createRoom(roomid string) *Room {
	room := Room{}
	if roomid != "" {
		room.id = roomid
	} else {
		cur := time.Now()
		timestamp := cur.UnixNano() / 1000000 //毫秒ranking
		room.id = strconv.FormatInt(timestamp, 10)
	}
	room.createTime = time.Now().Format(common.FormatTime)
	room.currentCard = nil
	room.idleusers = []*User{}
	room.users = []*User{}
	room.mapDeck = map[int]int{}
	room.setBaseScore(10)
	room.config()
	room.ConfigLog()
	room.roomDaemonThreadStart()
	return &room
}

//添加一个房间
func (r *roomManage) AddRoom() *Room {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	room := r.createRoom("")
	r.rooms[room.id] = room
	return room
}

//添加一个房间
func (r *roomManage) AddRoomWithRoomID(roomid string) *Room {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	room := r.createRoom(roomid)
	r.rooms[room.id] = room
	return room
}

//删除一个房间
func (r *roomManage) RemoveRoom(room *Room) {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	delete(r.rooms, room.id)
}

//获取房间
func (r *roomManage) GetRoom(roomid string) *Room {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	return r.rooms[roomid]
}

//获取房间数量
func (r *roomManage) GetRoomCount() int {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	return len(r.GetRooms())
}

/*
释放房间
*/
func (r *roomManage) ReleaseRoom(roomid string) bool {
	room := r.GetRoom(roomid)
	if room != nil {
		for _, user := range room.GetUsers() {
			if user != nil {
				user.close_countDown_playCard()
				UserManage.RemoveUser(user)
				mapUserConn.Delete(user.GetMemberid())
			}
		}
		room.close()
		return true
	}
	return false
}

//清空所有房间
func (r *roomManage) ClearALLRoom() {
	RoomManage.Lock.Lock()
	rooms := RoomManage.GetRooms()
	allRooms := []*Room{}
	for _, room := range rooms {
		if room != nil {
			allRooms = append(allRooms, room)
		}
	}
	RoomManage.Lock.Unlock()
	for _, room := range allRooms {
		//释放房间
		r.ReleaseRoom(room.GetRoomID())
		//解散房间
		matchEndBuild(types.KVS{frame.ROOMID: room.GetRoomID(), frame.ENDMODE: frame.ENDMODE_DISSOLVE})
	}
}
