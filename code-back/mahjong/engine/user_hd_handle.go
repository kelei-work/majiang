/*
玩家-操作-过牌
*/

package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"combine.com/utils/types"

	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
)

const (
	GANG_MING = 1
	GANG_AN   = 2
)

//操作类型
const (
	HANDLE_CHI  = iota //吃
	HANDLE_PENG        //碰
	HANDLE_GANG        //杠
	HANDLE_TING        //听
	HANDLE_HU          //胡
	HANDLE_QI          //弃
)

/*
玩家操作
in:{HandleType:操作类型,Content:内容}
out:1成功 -1操作无效 -103玩家没有操作权限
des:内容(杠:cardid 听:打出的CardID 吃:索引)
*/
func (this *Engine) Handle(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	handleType := args.GetInt("HandleType")
	content := args.GetString("Content")
	reply = types.Json(HandleWithUser(user, handleType, content))
	return
}

//同上
func HandleWithUser(user *User, hanleIndex int, content string) (res int) {
	defer func() {
		if p := recover(); p != nil {
			logger.Warnf("HandleWithUser:%v,%d", p, hanleIndex)
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
	//比赛结束了
	if room.GetRoomState() == frame.ROOMSTATE_OVER {
		return frame.HANDLE_NOPERM
	}
	//闲家天听时可以操作听和弃,其它时间需要判断有没有操作权限
	if hanleIndex == HANDLE_TING || hanleIndex == HANDLE_QI {
		if !isPlayerTianTingTimeHandle(user) {
			if !user.getHandlePerm() {
				return frame.HANDLE_NOPERM
			}
		}
	} else {
		if !user.getHandlePerm() {
			return frame.HANDLE_NOPERM
		}
	}
	//关闭玩家操作倒计时
	user.close_countDown_handle()
	room.setBankerFirstHandle(false)
	// fmt.Println("玩家：", *user.GetUserID(), "操作：", hanleIndex, "内容：", content)
	if hanleIndex != HANDLE_HU {
		room.setGangSendCard(false)
	}
	if hanleIndex == HANDLE_QI {
		if qi(user) {
			return
		}
	} else if hanleIndex == HANDLE_CHI {
		chi(user, content)
	} else if hanleIndex == HANDLE_PENG {
		peng(user)
	} else if hanleIndex == HANDLE_GANG {
		gang(user, common.ParseInt(content))
	} else if hanleIndex == HANDLE_TING {
		if ting(user, content) {
			return
		}
	} else if hanleIndex == HANDLE_HU {
		if hu(user) {
			room.checkMatchingOver()
			return
		}
	} else {
		return frame.HANDLE_INVALID
	}
	//触发玩家操作
	room.triggerUserHandle()
	return
}

//吃
func chi(user *User, content string) {
	room := user.GetRoom()
	currentCard := room.getCurrentCard()
	if currentCard == nil {
		qi(user)
		return
	}
	groups_str := (&Engine{}).GetChiGroups(ctx, types.KVS{frame.MEMBERID: user.getMemberid()}).GetMsg()
	if groups_str == "" {
		qi(user)
		return
	}
	groups_str = strings.Split(groups_str, "#")[0]
	currentCardID := currentCard.ID
	groups := strings.Split(groups_str, "$")
	index := common.ParseInt(content)
	group := groups[index]
	cardids := strings.Split(group, "|")
	caridid1 := cardids[0]
	caridid2 := cardids[1]
	//将吃的这套牌放入陈列区
	groupCardIDs := []string{strconv.Itoa(currentCardID), caridid1, caridid2}
	sort.Strings(groupCardIDs)
	//删除手里的牌
	cards := user.getCards()
	newCards := CardList{}
	newCards = append(newCards, cards...)
	mapCount := map[int]bool{}
	for i := len(newCards) - 1; i >= 0; i-- {
		cardid := newCards[i].ID
		if strconv.Itoa(cardid) == caridid2 {
			if mapCount[cardid] == false {
				mapCount[cardid] = true
				newCards = append(newCards[:i], newCards[i+1:]...)
			}
		} else if strconv.Itoa(cardid) == caridid1 {
			if mapCount[cardid] == false {
				mapCount[cardid] = true
				newCards = append(newCards[:i], newCards[i+1:]...)
			}
		}
		if len(mapCount) == 2 {
			break
		}
	}
	if len(mapCount) == 2 {
		room.dianPaoInfo = &DianPaoInfo{User: room.getCurrentCardsUser(), Card: currentCard}
		user.setCards(newCards)
		user.addChiCount()
		room.setCurrentCard(nil)
		user.addDisplayArea(strings.Join(groupCardIDs, "|"))
		/*
			吃牌推送
			push:CHI_Push,玩家ID|牌ID
		*/
		room.pushMessageToUsers("CHI_Push", []string{fmt.Sprintf("%d|%d", user.GetMemberid(), currentCardID)}, room.GetUsers())
		// fmt.Println("玩家：", *user.GetUserID(), " 吃牌：", currentCardID)
	} else {
		qi(user)
	}
}

//碰
func peng(user *User) {
	room := user.GetRoom()
	currentCard := room.getCurrentCard()
	if currentCard == nil {
		qi(user)
		return
	}
	currentCardsUser := room.getCurrentCardsUser()
	// fmt.Println("*********************************************************************************************[碰牌] 牌ID：", currentCard.ID)
	groupCardIDs := []string{strconv.Itoa(currentCard.ID)}
	cards := user.getCards()
	newCards := CardList{}
	newCards = append(newCards, cards...)
	count := 0
	for i := len(newCards) - 1; i >= 0; i-- {
		card := newCards[i]
		if card.ID == currentCard.ID {
			groupCardIDs = append(groupCardIDs, strconv.Itoa(card.ID))
			newCards = append(newCards[:i], newCards[i+1:]...)
			count++
			if count == 2 {
				break
			}
		}
	}
	if count == 2 {
		room.dianPaoInfo = &DianPaoInfo{User: currentCardsUser, Card: currentCard}
		user.setCards(newCards)
		user.addPengCount()
		user.pengInfo[currentCard.ID] = currentCardsUser
		room.setCurrentCard(nil)
		user.addDisplayArea(strings.Join(groupCardIDs, "|"))
		/*
			碰牌推送
			push:PENG_Push,玩家ID|牌ID
		*/
		room.pushMessageToUsers("PENG_Push", []string{fmt.Sprintf("%d|%d", user.GetMemberid(), currentCard.ID)}, room.GetUsers())
		// fmt.Println("玩家：", *user.GetUserID(), " 碰牌：", currentCard.ID)
	} else {
		qi(user)
	}
}

//杠
func gang(user *User, cardid int) {
	// go func() {
	room := user.GetRoom()
	currentCard := room.getCurrentCard()
	canGang := false        //是否可以开杠
	isFromHandCard := false //是否来自手牌
	newCards := CardList{}
	newCards = append(newCards, user.getCards()...)
	if currentCard != nil { //杠别人牌
		isFromHandCard = true
		cardid = currentCard.ID
		count := 0
		for i := len(newCards) - 1; i >= 0; i-- {
			card := newCards[i]
			if card.ID == cardid {
				newCards = append(newCards[:i], newCards[i+1:]...)
				count++
				if count == 3 {
					canGang = true
					break
				}
			}
		}
	} else { //自摸开杠
		if cardid == 0 { //系统选择第一套杠
			cardsid := user.getUserAllCardsID()
			tmpCards := []int{}
			for _, thecardid := range cardsid {
				if len(tmpCards) > 0 && thecardid != tmpCards[0] {
					tmpCards = []int{}
				}
				tmpCards = append(tmpCards, thecardid)
				if len(tmpCards) >= 4 {
					handCardCount := 0 //手牌数量
					for _, card := range user.getCards() {
						if card.ID == tmpCards[0] {
							handCardCount++
						}
					}
					if handCardCount > 0 {
						cardid = tmpCards[0]
						canGang = true
						if handCardCount > 1 {
							isFromHandCard = true
						}
						break
					}
				}
			}
		} else { //杠玩家选择的牌
			cardsid := user.getUserAllCardsID()
			count := 0
			for _, thecardid := range cardsid {
				if thecardid == cardid {
					count++
				}
				if count >= 4 {
					handCardCount := 0 //手牌数量
					for _, card := range user.getCards() {
						if card.ID == cardid {
							handCardCount++
						}
					}
					if handCardCount > 0 {
						canGang = true
						if handCardCount > 1 {
							isFromHandCard = true
						}
						break
					}
				}
			}
		}
		if canGang {
			for i := len(newCards) - 1; i >= 0; i-- {
				card := newCards[i]
				if card.ID == cardid {
					newCards = append(newCards[:i], newCards[i+1:]...)
				}
			}
		}
	}
	// fmt.Println("*********************************************************************************************[杠牌] 牌ID：", cardid)
	if canGang {
		room.dianPaoInfo = &DianPaoInfo{User: room.getCurrentCardsUser(), Card: currentCard}
		user.setCards(newCards)
		room.setCurrentCard(nil)
		room.setGangSendCard(true)
		gangStatus := GANG_MING //明杠
		if currentCard == nil {
			if isFromHandCard {
				gangStatus = GANG_AN //暗杠
			}
		}
		if !isFromHandCard {
			displayArea := strings.Split(user.getDisplayArea(), "$")
			for i, groupInfo := range displayArea {
				group := strings.Split(groupInfo, "#")[0]
				if strconv.Itoa(cardid) == strings.Split(group, "|")[0] {
					displayArea = append(displayArea[:i], displayArea[i+1:]...)
					break
				}
			}
			user.SetDisplayArea(strings.Join(displayArea, "$"))
		}
		groupInfo := fmt.Sprintf("%d|%d|%d|%d", cardid, cardid, cardid, cardid)
		user.addDisplayArea(fmt.Sprintf("%s#%d", groupInfo, gangStatus))
		isReplaceDisplayArea := 0
		if !isFromHandCard {
			isReplaceDisplayArea = 1
		}
		messages := []string{}
		//杠牌实时结算元宝
		baseIngot := room.getBaseScore()
		multipleBase := user.getMultipleBase()
		ingot := 0
		usersIngotChange := map[int]int{}
		if room.GetMatchID() != frame.MATCHID_FRIEND {
			if gangStatus == GANG_AN {
				ingot = baseIngot * multipleBase * 2
				//玩家元宝增减
				sumIngot := 0
				for _, u := range room.GetUsers() {
					if u != user {
						expendIngot := ingot
						userIngot := u.getIngot()
						if userIngot < ingot {
							expendIngot = userIngot
						}
						usersIngotChange[u.GetMemberid()] = -expendIngot
						frame.UpdateIngot(u.GetMemberid(), -expendIngot, 32, PLATFORM)
						sumIngot += expendIngot
					}
				}
				usersIngotChange[user.GetMemberid()] = sumIngot
				frame.UpdateIngot(user.getMemberid(), sumIngot, 32, PLATFORM)
			} else {
				ingot = baseIngot * multipleBase
				var u *User
				if currentCard == nil {
					u = user.pengInfo[cardid]
				} else {
					u = room.dianPaoInfo.User
				}
				//玩家元宝增减
				sumIngot := 0
				expendIngot := ingot
				userIngot := u.getIngot()
				if userIngot < ingot {
					expendIngot = userIngot
				}
				usersIngotChange[u.GetMemberid()] = -expendIngot
				frame.UpdateIngot(u.getMemberid(), -expendIngot, 32, PLATFORM)
				sumIngot += expendIngot
				usersIngotChange[user.GetMemberid()] = sumIngot
				frame.UpdateIngot(user.getMemberid(), sumIngot, 32, PLATFORM)
			}
		}
		ingotChanges := []int{}
		for _, u := range room.GetUsers() {
			ingotChanges = append(ingotChanges, usersIngotChange[u.GetMemberid()])
		}
		ingotChangesStr := common.Join(ingotChanges, "$")
		if gangStatus == GANG_AN {
			user.addAnGangCount()
			for _, u := range room.GetUsers() {
				if u == user || user.getIsTingCard() {
					messages = append(messages, fmt.Sprintf("%d|%d|%d|%d|%s", user.GetMemberid(), cardid, isReplaceDisplayArea, gangStatus, ingotChangesStr))
				} else {
					messages = append(messages, fmt.Sprintf("%d||%d|%d|%s", user.GetMemberid(), isReplaceDisplayArea, gangStatus, ingotChangesStr))
				}
			}
		} else {
			user.addMingGangCount()
			messages = append(messages, fmt.Sprintf("%d|%d|%d|%d|%s", user.GetMemberid(), cardid, isReplaceDisplayArea, gangStatus, ingotChangesStr))
		}
		/*
			杠牌推送
			push:GANG_Push,玩家ID|CardID|是否替换陈列区|杠类型|元宝$元宝...
			des:CardID(暗杠的情况下,其它玩家是”“值)
				是否替换陈列区(0不替换1替换)
		*/
		room.pushMessageToUsers("GANG_Push", messages, room.GetUsers())
		// fmt.Println("玩家：", *user.GetUserID(), " 杠牌：", cardid, "是否替换陈列区：", isReplaceDisplayArea)
	} else {
		qi(user)
	}
	// }()
}

//是否是闲家天听时的操作
func isPlayerTianTingTimeHandle(user *User) bool {
	room := user.GetRoom()
	return user != user.GetRoom().getBanker() && len(room.getTianTingStatusUsers()) > 0
}

//听
func ting(user *User, content string) bool {
	room := user.GetRoom()
	playCardID := common.ParseInt(content)
	tingGroupsInfo := user.getTingGroupsInfo()
	if tingGroupsInfo == "" {
		qi(user)
		return false
	}
	arr := strings.Split(tingGroupsInfo, "@")
	leftInfo := strings.Split(arr[0], "|")
	rightInfo := strings.Split(arr[1], "#")
	index := common.IndexStringOf(leftInfo, &content)
	if index == -1 {
		index = 0
		playCardID = common.ParseInt(leftInfo[0])
	}
	tingGroupInfo := rightInfo[index]
	//打出牌后剩余的牌
	surplusCards := CardList{}
	surplusCards = append(surplusCards, user.getCards()...)
	//玩家出牌的索引
	playIndex := -1
	//是否是闲家天听
	isPlayerTianTing := isPlayerTianTingTimeHandle(user)
	if isPlayerTianTing {
		playCardID = -1
		playIndex = 99
	} else {
		cards := user.getCards()
		for i := len(cards) - 1; i >= 0; i-- {
			card := cards[i]
			if card.ID == playCardID {
				playIndex = i
				surplusCards = append(surplusCards[:i], surplusCards[i+1:]...)
				break
			}
		}
	}
	// fmt.Println("*********************************************************************************************[听牌] 打出：", playCardID, " 听：", tingGroupInfo)
	if playIndex >= 0 {
		isTianTing := 0
		// if user.getSendCardCount() == 1 {
		// fmt.Println("==========", user.getSendCardCount(), user.getChiCount(), user.getPengCount(), user.getGangCount())
		if user.getSendCardCount() == 1 && (user.getChiCount()+user.getPengCount()+user.getGangCount() == 0) {
			user.setTingStatus(TingStatus_TIANTING) //天听
			isTianTing = 1
		} else {
			user.setTingStatus(TingStatus_TING) //听牌
		}
		user.setIsTingCard(true)
		cardsid := []int{}
		for _, cardInfo := range strings.Split(tingGroupInfo, "$") {
			cardid := common.ParseInt(strings.Split(cardInfo, "|")[0])
			cardsid = append(cardsid, cardid)
		}
		user.playCardID = playCardID
		// fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&:", cardsid)
		user.setTingGroupInfo(cardsid)
		//天听状态下先存下庄家要打的牌，等所有人都操作完了，执行庄家打牌的操作
		if tianTingStatusHandle(user, playIndex) {
		} else {
			//不是天听状态下的听牌直接打牌
			go func() {
				defer func() {
					if p := recover(); p != nil {
						logger.Warnf("[recovery] go PlayCardWithUsererr:%v", p)
					}
				}()
				res := PlayCardWithUser(user, playIndex)
				room.Log(fmt.Sprintf("==================%d", res))
			}()
		}
		/*
			听牌推送
			push:TING_Push,玩家ID#是否是天听
		*/
		room.pushMessageToUsers("TING_Push", []string{fmt.Sprintf("%d#%d", user.GetMemberid(), isTianTing)}, room.GetUsers())
		// fmt.Println("玩家：", *user.GetUserID(), "听牌")
		// time.Sleep(time.Second * 1)
		// for i := 0; i < 10; i++ {
		// 	fmt.Println("")
		// }
		return true
	} else {
		qi(user)
	}
	return false
}

//获取胡的类型
func getHuType(user *User) int {
	room := user.GetRoom()
	if room.getSendCardCount() == 1 && room.getPlayCardCount() == 0 {
		return HuStatus_TIANHU
	} else {
		if user != room.getBanker() { //闲家才会(人胡、地胡)
			currentCard := room.getCurrentCard()
			currentCardsUser := room.getCurrentCardsUser()
			if currentCardsUser == room.getBanker() && room.getSendCardCount() == 1 { //庄家起手出的第一张牌
				//不是别人点炮
				if currentCard != nil {
					return HuStatus_DIHU
				}
			}
		}
	}
	return HuStatus_NORMAL
}

//获取胡的类型
func getHuType_(user *User) int {
	room := user.GetRoom()
	if room.getSendCardCount() == 1 && room.getPlayCardCount() == 0 {
		return HuStatus_TIANHU
	} else {
		if user != room.getBanker() { //闲家才会(人胡、地胡)
			currentCard := room.getCurrentCard()
			currentCardUser := room.getCurrentCardsUser()
			if currentCardUser == room.getBanker() && room.getSendCardCount() == 1 { //庄家起手出的第一张牌
				//是别人点炮
				if currentCard != nil {
					return HuStatus_DIHU
				}
			} else if user.getSendCardCount() == 1 { //摸过一次牌
				//不是别人点炮
				if currentCard == nil {
					return HuStatus_DIHU
				}
			}
		}
	}
	return HuStatus_NORMAL
}

//胡
func hu(user *User) bool {
	room := user.GetRoom()
	cards := user.getCards()
	currentCard := room.getCurrentCard()
	currentCardsUser := room.getCurrentCardsUser()
	dianPaoCardID := ""
	if room.dianPaoInfo != nil {
		currentCard = room.dianPaoInfo.Card
		currentCardsUser = room.dianPaoInfo.User
	}
	if currentCard != nil { //点炮
		cards = append(cards, currentCard)
		user.setCards(cards)
		dianPaoCardID = strconv.Itoa(currentCard.ID)
		room.setDianPaoUser(currentCardsUser)
		room.setMatchResult(MatchResult_DianPao)
		room.setHandleLastCard(currentCard)
	} else { //自摸
		room.setMatchResult(MatchResult_ZiMo)
	}
	if room.huCheck(cards) {
		room.setHuStatus(getHuType(user))
		// user.orderCards()
		// fmt.Println("*********************************************************************************************[胡牌] ：", *user.getCardsID("|"))
		/*
			胡牌推送
			push:HU_Push,玩家ID|点炮的牌
			des:点炮的牌(自摸是“”)
		*/
		room := user.GetRoom()
		room.pushMessageToUsers("HU_Push", []string{fmt.Sprintf("%d|%s", user.GetMemberid(), dianPaoCardID)}, room.GetUsers())
		room.setHuUser(user)
		// fmt.Println("玩家：", user.GetMemberid(), "胡牌，比赛结束")
		return true
	} else {
		qi(user)
		return false
	}
}

//弃
func qi(user *User) bool {
	// fmt.Println("*********************************************************************************************[放弃操作] 玩家ID：", *user.GetUserID())
	room := user.GetRoom()
	//系统发牌后触发的玩家天听操作
	if tianTingStatusHandle(user, -1) {
		return true
	}
	if room.getCurrentCard() == nil {
		user.setGiveUp(true)
	} else {
		canHandleUserInfoList := room.getCanHandleUserInfoList()
		if len(canHandleUserInfoList) == 0 {
			currentCardsUser := room.getCurrentCardsUser()
			room.setControllerUser(currentCardsUser)
			nextUser := room.getNextUser()
			room.setControllerUser(nextUser)
			room.setCurrentCard(nil)
		} else if len(canHandleUserInfoList) == 1 {
			currentCardsUser := room.getCurrentCardsUser()
			room.setControllerUser(currentCardsUser)
		}
	}
	return false
}

//天听状态下的操作
func tianTingStatusHandle(user *User, playIndex int) bool {
	room := user.GetRoom()
	room.lockTianTingHandle.Lock()
	defer room.lockTianTingHandle.Unlock()
	tianTingStatusUsers := room.getTianTingStatusUsers()
	if len(tianTingStatusUsers) > 0 {
		for i, u := range tianTingStatusUsers {
			if u == user {
				tianTingStatusUsers = append(tianTingStatusUsers[:i], tianTingStatusUsers[i+1:]...)
				break
			}
		}
		room.setTianTingStatusUsers(tianTingStatusUsers)
		if user == room.getBanker() {
			room.setBankerTingPlayIndex(playIndex)
		}
		if len(tianTingStatusUsers) == 0 {
			//天听状态下,庄家听牌
			if room.getBankerTingPlayIndex() >= 0 {
				go PlayCardWithUser(room.getBanker(), room.getBankerTingPlayIndex())
			} else {
				room.triggerUserHandle()
			}
		}
		return true
	}
	return false
}
