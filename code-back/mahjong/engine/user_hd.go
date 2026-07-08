/*
游戏服务器操作-玩家
*/

package engine

import (
	"context"
	"fmt"
	"strings"

	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/types"
)

/*
获取换场信息
in:{roomtype:房间类型}
out:新房间类型,元宝限制,时间限制
des:从低房间升到高房间时“时间限制”有作用，但也可能是个空
*/
func (this *Engine) GetTransitionInfo(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	roomType := args.GetInt(frame.ROOMTYPE)
	room := &Room{}
	room.setRoomType(roomType)
	msg := ""
	//玩家元宝太多房间限制的判断
	if ok, ingotlimit, ingotlimittime := userIngotTroppoRoomLimit(user, room); ok {
		msg = fmt.Sprintf("%d,%d,%s", roomType+1, ingotlimit, ingotlimittime)
	} else if ok, playingot := userIngotTooLittleRoomLimit(user, room); ok {
		for roomType > 0 {
			roomType = roomType - 1
			room.setRoomType(roomType)
			ok, _ := userIngotTooLittleRoomLimit(user, room)
			if !ok {
				break
			}
		}
		msg = fmt.Sprintf("%d,%d,", roomType, playingot)
	} else {
		msg = fmt.Sprintf("%d,%d,", roomType, 0)
	}
	reply = types.Json(msg)
	return
}

/*
获取加番牌
in:
out:CardID|番数
*/
func (this *Engine) GetMultipleCard(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	room := user.GetRoom()
	res := fmt.Sprintf("%d|%d", room.getMultipleCardID(), addMultipleCardMultiple)
	reply = types.Json(res)
	return
}

/*
获取所有人听牌状态
in:
out:userid|状态$userid|状态
des:状态(>0已听牌)
*/
func (this *Engine) GetUsersTingStatus(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	room := user.GetRoom()
	info := []string{}
	for _, u := range room.GetUsers() {
		info = append(info, fmt.Sprintf("%d|%d", u.GetMemberid(), u.getTingStatus()))
	}
	res := strings.Join(info, "$")
	reply = types.Json(res)
	return
}

/*
获取房间号
in:
out:-1不在房间中
	房间号
*/
func (this *Engine) GetRoomID(ctx context.Context, args types.KVS) (reply types.KVS) {
	res := "-1"
	user := GetUser(args)
	room := user.GetRoom()
	if room != nil {
		reply = types.Json(room.GetRoomID())
	} else {
		reply = types.Json(res)
	}
	return
}

/*
设置玩家托管状态
in:{state:状态}
push:玩家托管状态
des:状态(0不托1托)
*/
func (this *Engine) SetUserTG(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	user.setTrusteeship(args.GetIntBool("state"))
	return types.DONE
}

/*
获取吃牌组合列表
in:
out:牌|牌$牌|牌#吃的牌
*/
func (this *Engine) GetChiGroups(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	room := user.GetRoom()
	currentCard := room.getCurrentCard()
	groups := GetChiAllGroups(currentCard)
	arr := []string{}
	for _, group := range groups {
		cardid1, cardid2 := group[0], group[1]
		mapCount := map[int]bool{}
		for _, card := range user.getCards() {
			if card.ID == cardid1 {
				mapCount[cardid1] = true
			} else if card.ID == cardid2 {
				mapCount[cardid2] = true
			}
			if len(mapCount) >= 2 {
				arr = append(arr, fmt.Sprintf("%d|%d", cardid1, cardid2))
				break
			}
		}
	}
	res := strings.Join(arr, "$")
	res = fmt.Sprintf("%s#%d", res, currentCard.ID)
	reply = types.Json(res)
	return
}

/*
获取听牌组合列表
in:
out:打牌|打牌@听牌|数量|番数$听牌|数量|番数#听牌|数量|番数$听牌|数量|番数
*/
func (this *Engine) GetTingGroups(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	reply = types.Json(user.getTingGroupsInfo())
	return
}

/*
获取当前听牌组合
in:
out:听牌|数量|番数$听牌|数量|番数
*/
func (this *Engine) GetCurrentTingGroup(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	reply = types.Json(user.getTingGroupInfo())
	return
}

/*
获取陈列区
in:{target:目标memberid}
out:memberid#CardID|CardID|CardID$CardID|CardID|CardID
des:CardID(暗杠的情况下,其它玩家是”“值)
*/
func (this *Engine) GetDisplayArea(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetInt("target")
	user := UserManage.GetUser(memberid)
	myMemberid := args.GetMemberID()
	myUser := UserManage.GetUser(myMemberid)
	displayArea := user.getDisplayArea()
	if displayArea == "" {
		reply = types.Json(fmt.Sprintf("%d#", memberid))
		return
	}
	groupsInfo := strings.Split(displayArea, "$")
	groups := []string{}
	for _, groupInfo := range groupsInfo {
		group := strings.Split(groupInfo, "#")
		if len(group) > 1 { //是杠牌
			if common.ParseInt(group[1]) == GANG_MING { //明杠
				groups = append(groups, group[0])
			} else { //暗杠
				if user == myUser || myUser.getIsTingCard() {
					groups = append(groups, fmt.Sprintf("%s|||", strings.Split(group[0], "|")[0]))
				} else {
					groups = append(groups, "|||")
				}
			}
		} else {
			groups = append(groups, groupInfo)
		}
	}
	reply = types.Json(fmt.Sprintf("%d#%s", memberid, strings.Join(groups, "$")))
	return
}

/*
获取废弃区
in:{target:目标memberid}
out:memberid#CardID|CardID|CardID
*/
func (this *Engine) GetDiscardArea(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetInt("target")
	user := UserManage.GetUser(memberid)
	reply = types.Json(fmt.Sprintf("%d#%s", memberid, user.getDiscardArea()))
	return
}

/*
获取补花区
in:{target:目标memberid}
out:memberid#CardID|CardID|CardID
*/
func (this *Engine) GetRepairFlowerArea(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetInt("target")
	user := UserManage.GetUser(memberid)
	reply = types.Json(fmt.Sprintf("%d#%s", memberid, user.getRepairFlowerArea()))
	return
}

/*
获取对手手牌
in:{target:目标memberid}
out:memberid#CardID|CardID|CardID
des:看不见时CardID列表是“”
*/
func (this *Engine) GetRivalCards(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetInt("target")
	user := UserManage.GetUser(memberid)
	myMemberid := args.GetMemberID()
	myUser := UserManage.GetUser(myMemberid)
	cardsid := ""
	if myUser == user || myUser.getTingStatus() > 0 {
		cardsid = *user.getCardsID("|")
	} else {
		cardsid = strings.Repeat("|", len(user.getCards())-1)
	}
	reply = types.Json(fmt.Sprintf("%d#%s", memberid, cardsid))
	return
}

/*
获取听牌信息
in:
out:[memberid,memberid]
*/
func (this *Engine) GetTingInfo(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := UserManage.GetUser(args.GetMemberID())
	memberids := []int{}
	room := user.GetRoom()
	for _, u := range room.getUsers() {
		if u != nil {
			if u.isTingCard {
				memberids = append(memberids, u.GetMemberid())
			}
		}
	}
	reply = types.Json(memberids)
	return
}
