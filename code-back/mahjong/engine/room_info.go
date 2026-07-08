/*
房间数据库操作
*/

package engine

import (
	"fmt"
	"time"

	"combine.com/utils/frame"
	"combine.com/utils/redis"
	"combine.com/utils/types"
)

/*
=========================================================================================================
表-RoomData
=========================================================================================================
*/

//获取表数据
func (r *Room) GetRoomData(args ...string) ([]int, error) {
	matchid := r.GetMatchID()
	roomtype := r.GetRoomType()
	condition := types.KVS{frame.PLATFORM: PLATFORM, frame.MATCHID: matchid, frame.ROOMTYPE: roomtype}
	key := fmt.Sprintf("roomdata:%d:%d:%d", PLATFORM, matchid, roomtype)
	return redis.Ints(frame.LoadRow(frame.PLATFORM_MEMBER, "roomdata", condition, key, time.Second, nil, args...))
}
