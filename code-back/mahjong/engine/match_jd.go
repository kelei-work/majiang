/*
比赛构建-够级英雄
*/

package engine

import (
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"

	"combine.com/utils/common"
	"combine.com/utils/logger"
)

//玩家元宝太多房间限制的判断
func userIngotTroppoRoomLimit(user *User, room *Room) (bool, int, string) {
	userIngot := user.getIngot()
	roomData, err := redis.Strings(room.GetRoomData("ingotlimit", "ingotlimittime"))
	logger.CheckFatal(err, "userIngotTroppoRoomLimit")
	ingotlimit := common.ParseInt(roomData[0])
	ingotlimittime := roomData[1]
	if ingotlimit > 0 {
		if ingotlimittime == "" {
			if userIngot >= ingotlimit {
				return true, ingotlimit, ingotlimittime
			}
		} else {
			times := strings.Split(ingotlimittime, "|")
			timeA, err := common.ParseTime(times[0], common.FormatTimeHMS)
			timeB, err2 := common.ParseTime(times[1], common.FormatTimeHMS)
			if err != nil || err2 != nil {
				if userIngot >= ingotlimit {
					return true, ingotlimit, ingotlimittime
				}
			} else {
				timeNow, _ := common.ParseTime(time.Now().Format(common.FormatTimeHMS), common.FormatTimeHMS)
				if timeNow.After(timeA) && timeNow.Before(timeB) {
					if userIngot >= ingotlimit {
						return true, ingotlimit, ingotlimittime
					}
				}
			}
		}
	}
	return false, ingotlimit, ingotlimittime
}

//玩家元宝太少房间限制的判断
func userIngotTooLittleRoomLimit(user *User, room *Room) (bool, int) {
	userIngot := user.getIngot()
	roomData, err := redis.Strings(room.GetRoomData("playingot"))
	logger.CheckFatal(err, "userIngotTooLittleRoomLimit")
	playingot := common.ParseInt(roomData[0])
	if userIngot < playingot {
		return true, playingot
	}
	return false, playingot
}
