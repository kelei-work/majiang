package engine

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"combine.com/utils/frame"

	"combine.com/utils/common"
	"combine.com/utils/logger"
	"combine.com/utils/redis"
	"combine.com/utils/types"
)

/*
获取话费赛时间
in:
out:20:00|20:32|
des:
*/
func GetHFSTime(ctx context.Context, args types.KVS) (reply types.KVS) {
	info, _ := redis.String(getGameConfigInfo("hfstime"))
	hfstime := fmt.Sprintf("%s|%s", info[0], common.GetCurrentTime())
	reply = types.Json(hfstime)
	return
}

func getHFSInfoKey(platform int) string {
	return fmt.Sprintf("HFSInfo:%d", platform)
}

/*
获取话费赛排名
in:
out:当前时间|花费元宝#userid|玩家名|元宝数量|头像地址$#我的排名|我的元宝数量#描述
des:
*/
func GetHFSInfo(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetMemberID()
	rds := frame.GetMemberRds()
	key := getHFSInfoKey(PLATFORM)
	info, _ := rds.Get(ctx, key).Result()
	if info == "" {
		timeInfo := func() string {
			roomData, err := frame.GetRoomData(PLATFORM, frame.MATCHID_HFS, frame.ROOMTYPE_NORM, "expendingot")
			logger.CheckFatal(err)
			expendIngot := roomData[0]
			return fmt.Sprintf("%s|%d", common.GetCurrentTime(), expendIngot)
		}
		rankingInfo := func() string {
			memberDB := frame.GetMemberDB()
			rows, err := memberDB.Query("call GetHFSRanking(?)", PLATFORM)
			defer rows.Close()
			logger.CheckFatal(err, "GetHFSRanking")
			buff := bytes.Buffer{}
			userid, ingot := 0, 0
			userName, headUrl := "", ""
			for rows.Next() {
				rows.Scan(&userid, &userName, &ingot, &headUrl)
				userName = common.UnicodeEmojiDecode(userName)
				buff.WriteString(fmt.Sprintf("%d|%s|%d|%s$", userid, userName, ingot, headUrl))
			}
			return *common.RemoveLastChar(buff)
		}
		info = fmt.Sprintf("%s#%s", timeInfo(), rankingInfo())
		rds.Set(ctx, key, info, common.TIME_SHORT)
	}
	data, _ := redis.Strings(getGameConfigInfo("hfsdes"))
	hfsdes := data[0]
	info = fmt.Sprintf("%s#%s#%s", info, *GetHFSMyRankingInfo(memberid), hfsdes)
	reply = types.Json(info)
	return
}

func GetHFSMyRankingInfo(memberid int) *string {
	rds := frame.GetMemberRds()
	key := fmt.Sprintf("HFSRanking:%d", memberid)
	info, _ := rds.Get(ctx, key).Result()
	if info == "" {
		myRankingInfo := func() string {
			memberDB := frame.GetMemberDB()
			ranking, ingot, hfsMatchTimeStatus, inning := 0, 0, 0, -1
			memberDB.QueryRow("call GetHFSMyRanking(?,?)", memberid, PLATFORM).Scan(&ranking, &ingot, &hfsMatchTimeStatus, &inning)
			if inning == 0 {
				ranking = 0
			}
			return fmt.Sprintf("%d|%d", ranking, ingot)
		}
		info = myRankingInfo()
		rds.Set(ctx, key, info, common.TIME_SHORT)
	}
	return &info
}

/*
获取话费赛奖励
in:
out:排名@itemid|count$itemid|count#
*/
func GetHFSAward(ctx context.Context, args types.KVS) (reply types.KVS) {
	rds := frame.GetMemberRds()
	key := "HFSAward"
	info, _ := rds.Get(ctx, key).Result()
	if info == "" {
		memberDB := frame.GetMemberDB()
		count := 0
		memberDB.QueryRow("select count(*) from HFSAward where platform=?;", PLATFORM).Scan(&count)
		rows, err := memberDB.Query("select RankingRange,content from HFSAward where platform=?;", PLATFORM)
		defer rows.Close()
		logger.CheckError(err)
		buff := bytes.Buffer{}
		rankingRange := 0
		content := ""
		prev := 0
		i := 0
		for rows.Next() {
			i++
			rows.Scan(&rankingRange, &content)
			ranking := ""
			if i == count {
				ranking = fmt.Sprintf("%d以上", prev)
			} else {
				if rankingRange-prev > 1 {
					ranking = fmt.Sprintf("%d-%d", prev+1, rankingRange)
				} else {
					ranking = strconv.Itoa(rankingRange)
				}
			}
			prev = rankingRange
			buff.WriteString(fmt.Sprintf("%s@%s#", ranking, content))
		}
		info = *common.RemoveLastChar(buff)
		rds.Set(ctx, key, info, common.TIME_SHORT)
	}
	reply = types.Json(info)
	return
}

//设置过期
func (u *User) setHFSInfoExpire() {
	rds := frame.GetMemberRds()
	key := getHFSInfoKey(PLATFORM)
	rds.Expire(ctx, key, 0)
}
