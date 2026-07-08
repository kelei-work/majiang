/*
定时任务
*/

package engine

import (
	"time"

	"combine.com/utils/delay"
	"combine.com/utils/frame"
)

//引擎初始化
func engine_init() {
	taskSystem := delay.NewTaskSystem()
	key := "matchingRoomCountStatistics"
	task := &delay.Task{
		Key: key,
		Exec: func() {
			MatchingRoomCountStatistics()
		},
		SurplusTime: time.Second * 10,
		CycleMode:   delay.CYCLEMODE_FOREVER,
	}
	taskSystem.AddTask(task)
	taskSystem.StartTask(key)
	key = "clearEmptyRoom"
	task = &delay.Task{
		Key: key,
		Exec: func() {
			clearBadRoom()
		},
		SurplusTime: time.Minute * 15,
		CycleMode:   delay.CYCLEMODE_FOREVER,
	}
	taskSystem.AddTask(task)
	taskSystem.StartTask(key)
}

//比赛中的房间数量统计
func MatchingRoomCountStatistics() {
	RoomManage.Lock.Lock()
	matchingRoomCount := len(RoomManage.GetRooms())
	RoomManage.Lock.Unlock()
	rds := frame.GetMemberRds()
	rds.Set(ctx, frame.GetMatchingRoomCountKey(PLATFORM), matchingRoomCount, time.Minute)
}

//清理坏房间
func clearBadRoom() {
	RoomManage.Lock.Lock()
	rooms := RoomManage.GetRooms()
	badRooms := []*Room{}
	for _, room := range rooms {
		if room != nil {
			//获取不在线人数
			offlineCount := 0
			for _, u := range room.GetUsers() {
				if u != nil && !u.getOnline() {
					offlineCount++
				}
			}
			//房间中的人都不在线 && 房间创建时长>10分钟
			if offlineCount == rule.PCount && room.GetCreateDuration() > 10 {
				badRooms = append(badRooms, room)
			}
		}
	}
	RoomManage.Lock.Unlock()
	for _, room := range badRooms {
		RoomManage.ReleaseRoom(room.GetRoomID())
	}
}
