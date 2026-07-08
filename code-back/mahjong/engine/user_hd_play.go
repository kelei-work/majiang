/*
玩家-操作-出牌
*/

package engine

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"combine.com/utils/frame"
	"combine.com/utils/logger"
	"combine.com/utils/types"
)

/*
玩家出牌
in:{CardID:牌ID,FrontCardCount:前台牌数量}
out:1成功 -2手中没有这张牌 -4玩家不可出牌 -5花牌不可打出 -6前后台数据不一致 -7赛前操作(发牌、补花)或者等待天听时不可出牌 -103玩家没有操作权限
*/
func (this *Engine) PlayCard(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	cardid := args.GetInt("CardID")
	frontCardCount := args.GetInt("FrontCardCount")
	if frontCardCount != len(user.getCards()) {
		reply = types.Json(-6)
		return
	}
	//玩家听牌不可以自己出牌
	if user.userCanPlayCard() == 0 {
		reply = types.Json(-4)
		return
	}
	playIndex := user.getCardIDIndex(cardid)
	if playIndex < 0 {
		reply = types.Json(-2)
		return
	}
	reply = types.Json(PlayCardWithUser(user, playIndex))
	return
}

//同上
func PlayCardWithUser(user *User, playIndex int) (res int) {
	defer func() {
		if p := recover(); p != nil {
			logger.Warnf("PlayCardWithUser:%v,%d", p, playIndex)
		}
	}()
	user.lockHandle.Lock()
	defer user.lockHandle.Unlock()
	res = frame.HANDLE_SUCCESS
	//玩家没有操作权限
	if !user.getHandlePerm() {
		return frame.HANDLE_NOPERM
	}
	room := user.GetRoom()
	userCards := user.getCards()
	playCard := userCards[playIndex]
	//花牌不可打出
	if playCard.Type == CardType_Flower {
		return -5
	}
	//赛前操作(发牌、补花)或者等待天听时不可出牌
	if room.GetRoomState() != frame.ROOMSTATE_MATCH || len(room.getTianTingStatusUsers()) > 0 {
		return -7
	}
	//关闭玩家操作倒计时
	user.close_countDown_playCard()
	//更新玩家手中的牌
	user.updateUserCards(playIndex)
	//将牌打出
	user.play(playCard)
	//不能删除
	time.Sleep(time.Millisecond * 20)
	//将牌打出之后,后续的操作
	user.playAfter(playCard)
	return
}

/*
检查牌值是否有效
1.王可以和(任意一套牌)组合
*/
func isValid(cards []Card, room *Room) bool {
	return true
}

/*
打出
*/
func (u *User) play(playCard *Card) {
	defer func() {
		if p := recover(); p != nil {
			logger.Fatalf("[recovery] play err:%v", p)
		}
	}()
	room := u.GetRoom()
	cardid := playCard.ID
	room.setCurrentCard(playCard)
	room.setCurrentCardUser(u)
	//记录出牌信息
	u.recordPlayCard(playCard)
	//向所有人推送出牌的信息
	message := fmt.Sprintf("%d,%d,%d", u.GetMemberid(), cardid, PlayType_Normal)
	/*
		玩家出牌
		push:Play,userid,cardid,出牌类型
		des:出牌类型(0正常出牌 1残局进来的出牌)
	*/
	room.pushMessageToUsers("Play", []string{message}, room.getAllUsers())
}

//记录出牌信息
func (u *User) recordPlayCard(playCard *Card) {
	room := u.GetRoom()
	//记录出牌的信息（debug）
	playTimes := room.updatePlayTimes()
	bf := bytes.Buffer{}
	bf.WriteString(strconv.Itoa(playCard.ID))
	bf1 := bytes.Buffer{}
	for _, card := range u.getCards() {
		bf1.WriteString(strconv.Itoa(card.ID))
		bf1.WriteString("|")
	}
	//记录牌
	logger.Debugf("玩家[%d],第（%d）次出牌:%s,剩余牌:%s", u.GetMemberid(), playTimes, bf.String(), bf1.String())
}

//打出之后
func (u *User) playAfter(playCard *Card) {
	defer func() {
		if p := recover(); p != nil {
			logger.Fatalf("[recovery] playAfter err:%v", p)
		}
	}()
	room := u.GetRoom()
	if room == nil {
		return
	}
	//比赛结束
	if room.GetRoomState() == frame.ROOMSTATE_OVER {
		return
	}
	room.addPlayCardCount()
	room.setCanHandleUserList()
	room.triggerUserHandle()
}

//将出的牌的index列表转化成对应的牌列表
func indexsToCards(indexs []int, userCards []Card) []Card {
	cards := make([]Card, len(indexs))
	for i, index := range indexs {
		cards[i] = userCards[index]
	}
	return cards
}

//根据cardid获取牌index
func (u *User) getCardIDIndex(cardid int) int {
	cards := u.getCards()
	for i, card := range cards {
		if card.ID == cardid {
			return i
		}
	}
	return -1
}
