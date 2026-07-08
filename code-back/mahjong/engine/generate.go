package engine

import (
	"context"
	"fmt"
	"strings"

	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
	"combine.com/utils/types"
)

func (this *Engine) GenerateMatch(ctx context.Context, args types.KVS) (reply types.KVS) {
	defer func() {
		if p := recover(); p != nil {
			err := fmt.Sprintf("[recovery] GenerateMatch err : %v", p)
			reply = types.JsonErr(err)
			logger.Errorf(err)
		}
	}()
	roomid := args.GetString(frame.ROOMID)
	//调用结算服务进行金额的扣除
	reply = matchBegin(roomid)
	if reply.IsErr() {
		return
	}
	//生成一个房间
	room := RoomManage.AddRoomWithRoomID(roomid)
	rds := frame.GetBuildRds()
	info, err := rds.HMGet(ctx, frame.GetRoomInfoKey(roomid), frame.MATCHID, frame.ROOMTYPE, frame.PLAYTYPE, frame.INNINGS, frame.INNING, frame.INTEGRALS, frame.CREATETIME).Result()
	logger.CheckError(err)
	matchid, roomtype, playtype, innings, inning, integrals_, createtime := info[0], info[1], info[2], info[3], info[4], info[5], info[6]
	room.setMatchID(common.Int(matchid))
	room.setRoomType(common.Int(roomtype))
	if playtype != nil {
		room.setRoomType(common.Int(playtype))
	}
	if innings != nil {
		room.setInnings(common.Int(innings))
	}
	if inning != nil {
		room.setInning(common.Int(inning))
	}
	if createtime != nil {
		room.SetCreateTime(createtime.(string))
	}
	integrals := make([]int, frame.GetPCount(PLATFORM))
	if integrals_ != nil {
		integrals = common.StrArrToIntArr(strings.Split(integrals_.(string), ","))
	}
	//获取房间中的玩家列表
	memberids := getRoomUsers(roomid)
	//填充玩家
	fillUsers(room, memberids, integrals)
	//开赛
	room.match_Opening()
	return types.Succ()
}

//获取房间中的玩家列表
func getRoomUsers(roomid string) []int {
	rds := frame.GetBuildRds()
	memberids, err := rds.LRange(ctx, frame.GetRoomSeatsKey(roomid), 0, -1).Result()
	logger.CheckFatal(err)
	return common.StrArrToIntArr(memberids)
}

//填充玩家
func fillUsers(room *Room, memberids []int, integrals []int) {
	for i, memberid := range memberids {
		user := UserManage.AddUser(memberid, getConn(memberid))
		user.setRoom(room)
		user.setIndex(i)
		user.setRoundIntegral(integrals[i])
		room.users = append(room.users, user)
	}
}
