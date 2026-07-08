/*
玩家数据库操作
*/

package engine

import (
	"github.com/garyburd/redigo/redis"

	"combine.com/utils/common"
	"combine.com/utils/frame"
)

/*
=========================================================================================================
登录服务器
=========================================================================================================
*/

/*
获取member信息
*/
func (u *User) GetMemberInfo(args ...string) ([]interface{}, error) {
	return frame.GetMemberInfo(u.getMemberid(), args...)
}

//获取玩家名
func (u *User) getUserName() string {
	username, _ := redis.String(u.GetMemberInfo("username"))
	username = common.UnicodeEmojiDecode(username)
	return username
}

//获取元宝数量
func (u *User) getIngot() int {
	ingot := frame.GetIngot(u.getMemberid())
	return ingot
}

//获取vip
func (u *User) getVIP() int {
	if frame.IsVip(u.getMemberid()) {
		return 1
	}
	return 0
}

/*
=========================================================================================================
游戏服务器
=========================================================================================================
*/

/*
获取玩家信息
*/
func (u *User) GetUserInfo(args ...string) ([]interface{}, error) {
	return frame.GetUserInfo(u.getMemberid(), PLATFORM, args...)
}

//设置玩家信息数据过期
func (u *User) setUserInfoExpire() {
	frame.SetUserInfoExpire(u.getMemberid(), PLATFORM)
}

/*
获取玩家等级
*/
func (u *User) getLevel() int {
	level, _ := redis.Int(u.GetUserInfo("level"))
	return level
}

/*
比赛结束-更新玩家数据
in:积分、MatchResult
*/
func (u *User) endUpdateUserInfo(mapInfo map[string]int) {
	// matchResult := mapInfo["matchResult"]
	// win, flat, lose := 0, 0, 0
	// if matchResult == MatchResult_Win {
	// 	win = 1
	// } else if matchResult == MatchResult_Flat {
	// 	flat = 1
	// } else if matchResult == MatchResult_Lose {
	// 	lose = 1
	// }
	// roomType := u.GetRoom().GetRoomType()
	// _, err := gameDB.Exec("call MatchEnd(?,?,?,?,?)", u.GetMemberid(), roomType, win, flat, lose)
	// logger.CheckFatal(err, "endUpdateUserInfo")
}

/*
比赛结束(话费赛)-更新玩家数据
*/
func (u *User) hfsEndUpdateUserInfo(win int, ingot int) {
	// _, err := gameDB.Exec("call MatchEndHFS(?,?,?)", u.GetMemberid(), win, ingot)
	// logger.CheckFatal(err, "hfsEndUpdateUserInfo")
}
