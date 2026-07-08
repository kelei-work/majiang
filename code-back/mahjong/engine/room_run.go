/*
房间比赛中
*/

package engine

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"

	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
)

var (
	ACCELERATE     = false //是否是加速模式
	ETA            = false //效率优化
	PRINT_CRUX_LOG = false //打印番数类型
)

/*
触发玩家操作
*/
func (r *Room) triggerUserHandle() {
	// go func() {
	canHandleUserInfo := &CanHandleUserInfo{}
	qiStatus := 0
	canHandle := false
	if r.getCurrentCard() == nil {
		//系统发牌
		canHandleUserInfo = r.sendCard()
		if canHandleUserInfo.HaveHandle() {
			canHandle = true
		}
	} else {
		//玩家打牌
		canHandleUserInfo = r.playCard()
		if canHandleUserInfo.HaveHandle() {
			qiStatus = 1
			canHandle = true
		}
	}
	// if canHandle {
	// fmt.Sprintf("玩家操作列表：%v",canHandleUserInfo)
	// }
	//取消此次触发
	if canHandleUserInfo.CancelStatus == 1 {
		return
	}
	waitTime := r.getWaitTime()
	//默认操作
	defaultHandle := func(u *User) {
		handleWaitTime := 10
		handleType := HANDLE_QI
		content := ""
		if ACCELERATE {
			if canHandleUserInfo.HuStatus > 0 {
				handleType = HANDLE_HU
			} else if canHandleUserInfo.TingStatus > 0 {
				handleType = HANDLE_TING
				tingCardsInfo := u.getTingGroupsInfo()
				arr := strings.Split(tingCardsInfo, "@")
				content = strings.Split(arr[0], "|")[0]
				handleWaitTime = waitTime
			} else if canHandleUserInfo.GangStatus > 0 {
				handleType = HANDLE_GANG
				content = "0"
			} else if canHandleUserInfo.PengStatus > 0 {
				handleType = HANDLE_PENG
			}
		}
		if u.getIsTingCard() {
			if canHandleUserInfo.HuStatus > 0 {
				handleWaitTime = 1
				handleType = HANDLE_HU
			}
		}
		u.countDown_handle(handleWaitTime, handleType, content)
	}
	//默认出牌
	defaultPlayCard := func(u *User) {
		//听牌系统帮出牌
		if u.getIsTingCard() {
			waitTime = 1
		}
		//出牌倒计时
		u.countDown_playCard(waitTime)
	}
	controllerUser := r.getControllerUser()
	cards := controllerUser.getCards()
	isTianTingStatus := false
	if controllerUser.getSendCardCount() == 1 {
		if !controllerUser.getCheckTing() {
			controllerUser.setCheckTing(true)
			if r.tingCheck(cards, controllerUser) {
				if controllerUser.getChiCount()+controllerUser.getPengCount()+controllerUser.getGangCount() == 0 {
					isTianTingStatus = true
				}
			}
		}
	}
	// tianTingUsers := r.getTianTingStatusUsers()
	// isTianTingStatus := len(tianTingUsers) > 0

	//玩家可操作时的系统行为
	userTrusteeshipBehavior := func(u *User) {
		if ACCELERATE {
			//加速模式,能操作就操作
			defaultHandle(u)
		} else {
			if u.userCanPlayCard() == 1 {
				defaultPlayCard(u)
			} else {
				defaultHandle(u)
			}
		}
	}
	//推送玩家操作
	pushUserHandle := func(u *User) {
		if isTianTingStatus { //天听操作
			canHandleUserInfo.TingStatus = TingStatus_TIANTING
			// qiStatus = 1
		}
		//听牌没有弃
		if u.getIsTingCard() {
			qiStatus = 0
		}
		handles := fmt.Sprintf("%d|%d|%d|%d|%d|%d", canHandleUserInfo.ChiStatus, canHandleUserInfo.PengStatus, canHandleUserInfo.GangStatus, canHandleUserInfo.TingStatus, canHandleUserInfo.HuStatus, qiStatus)
		multiple := 0
		if canHandleUserInfo.HuStatus > 0 {
			multiple = r.getHuCardMultiple(u, canHandleUserInfo.CardsID)
		}
		if canHandle {
			if canHandleUserInfo.TingStatus != 1 {
				waitTime = 10
			}
		}
		message := fmt.Sprintf("%d,%s,%d,%d,%d", u.GetMemberid(), handles, u.userCanPlayCard(), waitTime, multiple)
		//记录牌权信息
		r.setSetCtlMsg(message)
		/*
			玩家操作推送
			push:SetController_Push,玩家ID,操作状态列表(吃|碰|杠|听|胡|弃),是否可出牌,等待时间,胡牌番数
			des:操作状态列表(0暗1亮)
				胡牌番数(可胡牌时才有用)
		*/
		// if isTianTingStatus {
		// 	u.push("SetController_Push", &message)
		// } else {
		r.pushMessageToUsers("SetController_Push", []string{message}, r.getUsers())
		// }
	}
	if isTianTingStatus { //天听操作
		// fmt.Println("有天听！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！")
		userTrusteeshipBehavior(controllerUser)
		pushUserHandle(controllerUser)
		for _, user := range r.GetUsers() {
			if user != controllerUser {
				res := fmt.Sprintf("%d", waitTime)
				/*
					等待玩家天听
					push:WaitTT_Push,等待时间
				*/
				user.push("WaitTT_Push", res)
			}
		}
	} else {
		//获取牌权玩家
		user := r.getControllerUser()
		if canHandle { //玩家可操作
			if user.userCanPlayCard() == 1 { //玩家也可打牌
				userTrusteeshipBehavior(user)
			} else {
				defaultHandle(user) //玩家只可操作
			}
		} else { //只能打牌
			defaultPlayCard(user)
		}
		pushUserHandle(user)
	}
	// }()
}

// 获取胡牌番数
func (r *Room) getHuCardMultiple(user *User, cardsid []int) (multiple int) {
	//基本番
	user.multiple_base(cardsid)
	//牌型番
	// fmt.Println("aaaaaaaaa:", cardsid)
	user.multiple_cardType(cardsid)
	//加番牌
	user.multiple_addCard(cardsid)
	//相加
	// fmt.Println(user.getMultipleBase(), user.getSumMultipleWithCardType())
	// fmt.Println("获取胡牌番数：", *user.GetUserID(), user.getMultipleBase(), user.getMultipleRepairFlower(), user.getSumMultipleWithCardType(), user.getMultipleAddCard())
	multiple = user.getMultipleBase() + user.getSumMultipleWithCardType() + user.getMultipleAddCard()
	return multiple
}

// 系统发牌
func (r *Room) sendCard() *CanHandleUserInfo {
	// fmt.Println("系统发牌")
	canHandleUserInfo := &CanHandleUserInfo{}
	var user *User
	//获取庄家第一次操作,不需要发牌了,生成手牌的时候已经加进去了
	if r.getBankerFirstHandle() {
		user = r.getBanker()
		if user.getGiveUp() {
			return canHandleUserInfo
		}
		//暗杠检测
		if r.gangCheck() {
			canHandleUserInfo.GangStatus = 2
		}
		huWay := r.calculateHuMaxMultiple(user, nil)
		if huWay.isHu() { //胡检测
			canHandleUserInfo.HuStatus = 1
			user.maxMultipleCards = huWay.Cards
			canHandleUserInfo.CardsID = user.getUserMaxMultipleCardsID()
			sort.Ints(canHandleUserInfo.CardsID)
		}
	} else {
		user = r.getControllerUser()
		if user.getGiveUp() {
			return canHandleUserInfo
		}
		cards := user.getCards()
		//手牌缺一张
		if len(cards)%3 == 1 {
			//系统发牌
			if r.insertIntoUserCardsFromCardPoll(user, true) == nil { //没牌发了,平局
				canHandleUserInfo.CancelStatus = 1
				// fmt.Println("平牌-sendCard")
				r.setMatchResult(MatchResult_LiuJu)
				r.checkMatchingOver()
				return canHandleUserInfo
			}
			r.dianPaoInfo = nil
		}
		cards = user.getCards()
		//听牌后只检测胡牌
		if user.getIsTingCard() {
			huWay := r.calculateHuMaxMultiple(user, nil)
			if huWay.isHu() { //胡检测
				canHandleUserInfo.HuStatus = 1
				user.maxMultipleCards = huWay.Cards
				canHandleUserInfo.CardsID = user.getUserMaxMultipleCardsID()
				sort.Ints(canHandleUserInfo.CardsID)
			}
		} else {
			//暗杠检测
			if r.gangCheck() {
				canHandleUserInfo.GangStatus = 2
			}
			huWay := r.calculateHuMaxMultiple(user, nil)
			if huWay.isHu() { //胡检测
				canHandleUserInfo.HuStatus = 1
				user.maxMultipleCards = huWay.Cards
				canHandleUserInfo.CardsID = user.getUserMaxMultipleCardsID()
				sort.Ints(canHandleUserInfo.CardsID)
			} else if r.tingCheck(cards, user) { //听检测
				canHandleUserInfo.TingStatus = 1
			}
		}
	}
	return canHandleUserInfo
}

var (
	sendCount = 0
)

// 从牌池中取出牌添加进玩家手牌中
func (r *Room) insertIntoUserCardsFromCardPoll(user *User, isPushSendCardMsg bool) *Card {
	card := r.getCardFromDeck()
	if card == nil {
		return card
	}
	// sendCount++
	// if sendCount == 1 {
	// 	card = NewCard(11)
	// } else if sendCount == 2 {
	// 	card = NewCard(16)
	// }
	r.setHandleLastCard(card)
	userCards := user.getCards()
	userCards = append(userCards, card)
	user.setCards(userCards)
	user.addSendCardCount()
	r.addSendCardCount()
	// fmt.Println("===================================================[系统发牌]:", *user.GetUserID(), card.ID, *user.getCardsID("|"))
	if isPushSendCardMsg {
		messages := []string{}
		for _, u := range r.GetUsers() {
			if u == user || u.getTingStatus() > 0 {
				messages = append(messages, fmt.Sprintf("%d|%d", user.GetMemberid(), card.ID))
			} else {
				messages = append(messages, fmt.Sprintf("%d|", user.GetMemberid()))
			}
		}
		/*
			系统发牌推送
			push:SendCard_Push,玩家ID|CardID
			des:CardID(其它玩家是”“值)
		*/
		r.pushMessageToUsers("SendCard_Push", messages, r.GetUsers())
	}
	r.pushDeckCount()
	card = r.repairFlower(user, card)
	time.Sleep(time.Millisecond * 20)
	return card
}

func (r *Room) repairFlower(user *User, card *Card) *Card {
	if card.Type != CardType_Flower {
		return card
	}
	time.Sleep(time.Second)
	//移除花牌
	cards := user.getCards()
	cards = cards[:len(cards)-1]
	user.setCards(cards)
	user.addMultipleRepairFlower(r.getBaseScore())
	//加入补花区
	user.addRepairFlowerArea(card.ID)
	// fmt.Println("移除花牌：", card.ID)
	r.setGangSendCard(false)
	//补花实时结算元宝
	usersIngotChange := map[int]int{}
	if r.GetMatchID() != frame.MATCHID_FRIEND {
		baseIngot := r.getRoomBaseIngot()
		sumIngot := 0
		for _, u := range r.GetUsers() {
			if u != user {
				expendIngot := baseIngot
				userIngot := u.getIngot()
				if userIngot < baseIngot {
					expendIngot = userIngot
				}
				usersIngotChange[u.GetMemberid()] = -expendIngot
				frame.UpdateIngot(u.getMemberid(), -expendIngot, 33, PLATFORM)
				sumIngot += expendIngot
			}
		}
		usersIngotChange[user.GetMemberid()] = sumIngot
		frame.UpdateIngot(user.getMemberid(), sumIngot, 33, PLATFORM)
	}
	ingotChanges := []int{}
	for _, u := range r.GetUsers() {
		ingotChanges = append(ingotChanges, usersIngotChange[u.GetMemberid()])
	}
	ingotChangesStr := common.Join(ingotChanges, "$")
	newCard := r.insertIntoUserCardsFromCardPoll(user, false)
	newCardID := ""
	if newCard != nil {
		newCardID = fmt.Sprintf("%d", newCard.ID)
	}
	messages := []string{}
	message := ""
	for _, theUser := range r.GetUsers() {
		if theUser == user {
			message = fmt.Sprintf("%d|%d|%s|%s", user.GetMemberid(), card.ID, newCardID, ingotChangesStr)
		} else {
			message = fmt.Sprintf("%d|%d||%s", user.GetMemberid(), card.ID, ingotChangesStr)
		}
		messages = append(messages, message)
	}
	/*
		移除花牌(行牌过程中)
		push:RemoveFlower_Push,userid|cardid|元宝$元宝...
	*/
	r.pushMessageToUsers("RemoveFlower_Push", messages, r.GetUsers())
	return newCard
}

type CanHandleUserInfo struct {
	ChiStatus    int
	PengStatus   int
	GangStatus   int
	TingStatus   int
	HuStatus     int
	CancelStatus int
	CardsID      []int
	User         *User
}

type CanHandleUserInfoList []*CanHandleUserInfo

func (list CanHandleUserInfoList) Len() int {
	return len(list)
}

func (list CanHandleUserInfoList) Less(i, j int) bool {
	iV := list[i].HuStatus*100 + (list[i].PengStatus+list[i].GangStatus)*10 + list[i].ChiStatus
	jV := list[j].HuStatus*100 + (list[j].PengStatus+list[j].GangStatus)*10 + list[j].ChiStatus
	if iV > jV {
		return true
	} else {
		return false
	}
}

func (list CanHandleUserInfoList) Swap(i, j int) {
	var temp *CanHandleUserInfo = list[i]
	list[i] = list[j]
	list[j] = temp
}

func (this *CanHandleUserInfo) HaveHandle() bool {
	if this.ChiStatus+this.PengStatus+this.GangStatus+this.TingStatus+this.HuStatus > 0 {
		return true
	}
	return false
}

// 玩家打牌
func (r *Room) playCard() *CanHandleUserInfo {
	// fmt.Println("玩家打牌")
	r.dianPaoInfo = nil
	r.setTianTingStatusUsers([]*User{})
	r.setBankerFirstHandle(false)
	r.setGangSendCard(false)
	controllerUser := r.getControllerUser()
	controllerUser.orderCards()
	controllerUser.setGiveUp(false)
	canHandleUserInfo := r.transitionController()
	return canHandleUserInfo
}

// 触发玩家吃碰杠后转换牌权
func (r *Room) transitionController() *CanHandleUserInfo {
	currentCard := r.getCurrentCard()
	controllerUser := r.getControllerUser()
	canHandleUserInfo := &CanHandleUserInfo{}
	canHandleUserInfoList := r.getCanHandleUserInfoList()
	if len(canHandleUserInfoList) == 0 {
		//打牌没有触发别的玩家操作
		// fmt.Println("===================================================[没有触发别的玩家操作]")
		controllerUser.addDiscardArea(currentCard.ID)
		//转换牌权
		user := r.getNextUser()
		r.setControllerUser(user)
		r.setCurrentCard(nil)
		//===
		canHandleUserInfo.CancelStatus = 1
		//触发玩家操作
		go func() {
			defer func() {
				if p := recover(); p != nil {
					logger.Warnf("[recovery] triggerUserHandle err:%v,%d,%d", p, r.GetMatchID(), r.GetRoomType())
				}
			}()
			r.triggerUserHandle()
		}()
		return canHandleUserInfo
	}
	// fmt.Println("aaaaa",)
	canHandleUserInfo = canHandleUserInfoList[0]
	//删除可操作的玩家
	surplusCanHandleUserInfoList := CanHandleUserInfoList{}
	surplusCanHandleUserInfoList = append(surplusCanHandleUserInfoList, r.canHandleUserInfoList[1:]...)
	r.canHandleUserInfoList = surplusCanHandleUserInfoList
	//转换牌权
	r.setControllerUser(canHandleUserInfo.User)
	return canHandleUserInfo
}

// 按顺序获取此玩家后面的其它玩家
func (r *Room) getOtherUsersByOrder(user *User) []*User {
	userIndex := user.getIndex()
	indexs := []int{}
	for i := 1; i < rule.PCount; i++ {
		nextUserIndex := userIndex + i
		if nextUserIndex >= rule.PCount {
			nextUserIndex = nextUserIndex - rule.PCount
		}
		indexs = append(indexs, nextUserIndex)
	}
	users := r.GetUsers()
	otherUsers := []*User{}
	for _, index := range indexs {
		otherUsers = append(otherUsers, users[index])
	}
	return otherUsers
}

// 设置可以吃碰杠的玩家信息列表
func (r *Room) setCanHandleUserList() {
	r.canHandleUserInfoList = CanHandleUserInfoList{}
	currentCard := r.getCurrentCard()
	//红中不需要检测吃碰杠
	if currentCard.ID == HUN {
		return
	}
	controllerUser := r.getControllerUser()
	otherUsers := r.getOtherUsersByOrder(controllerUser)
	for _, user := range otherUsers {
		huCanHandleUserInfo := &CanHandleUserInfo{}
		pengGangCanHandleUserInfo := &CanHandleUserInfo{}
		chiCanHandleUserInfo := &CanHandleUserInfo{}
		cards := user.getCards()
		count := 0
		othenCards := []*Card{}
		for _, card := range cards {
			if currentCard.ID == card.ID {
				count++
			} else {
				othenCards = append(othenCards, card)
			}
		}
		//没听牌状态下才检测吃碰杠
		if !user.getIsTingCard() {
			if r.chiCheck(user) { //吃牌检测
				chiCanHandleUserInfo.User = user
				chiCanHandleUserInfo.ChiStatus = 1
				// task := delay.Task{}
				// task.CycleMode = delay.CYCLEMODE_ONCE
				// task.SurplusTime = time.Second * 5
				// task.Exec = func() {
				// 	Handle([]string{*user.GetUserID(), strconv.Itoa(HANDLE_CHI), "0"})
				// }
				// task.Start()
			}
			if count >= 2 { //碰杠检测
				pengGangCanHandleUserInfo.User = user
				pengGangCanHandleUserInfo.PengStatus = 1 //碰
				if count == 3 {
					pengGangCanHandleUserInfo.GangStatus = 1 //明杠
				}
			}
		}
		huWay := r.calculateHuMaxMultiple(user, currentCard)
		if huWay.isHu() { //胡检测
			huCanHandleUserInfo.User = user
			huCanHandleUserInfo.HuStatus = 1
			user.maxMultipleCards = huWay.Cards
			huCanHandleUserInfo.CardsID = user.getUserMaxMultipleCardsID()
			sort.Ints(huCanHandleUserInfo.CardsID)
		}
		//排序(胡>碰杠>吃)
		if huCanHandleUserInfo.HaveHandle() {
			r.canHandleUserInfoList = append(r.canHandleUserInfoList, huCanHandleUserInfo)
		}
		if pengGangCanHandleUserInfo.HaveHandle() {
			r.canHandleUserInfoList = append(r.canHandleUserInfoList, pengGangCanHandleUserInfo)
		}
		if chiCanHandleUserInfo.HaveHandle() {
			r.canHandleUserInfoList = append(r.canHandleUserInfoList, chiCanHandleUserInfo)
		}
	}
	sort.Sort(r.canHandleUserInfoList)
}

// 获取可以吃碰杠的玩家信息列表
func (r *Room) getCanHandleUserInfoList() []*CanHandleUserInfo {
	return r.canHandleUserInfoList
}

func GetChiAllGroups(card *Card) [][]int {
	groups := [][]int{}
	var cardid1, cardid2 int
	if card.Type == CardType_Myriad || card.Type == CardType_Cake || card.Type == CardType_Strip {
		x := card.ID % 10
		y := card.ID / 10 * 10
		//前两个
		cardid1 = x - 2
		cardid2 = x - 1
		if cardid1 >= 1 && cardid2 >= 1 {
			groups = append(groups, []int{cardid1 + y, cardid2 + y})
		}
		//前后两个
		cardid1 = x - 1
		cardid2 = x + 1
		if cardid1 >= 1 && cardid2 <= 9 {
			groups = append(groups, []int{cardid1 + y, cardid2 + y})
		}
		//后两个
		cardid1 = x + 1
		cardid2 = x + 2
		if cardid1 <= 9 && cardid2 <= 9 {
			groups = append(groups, []int{cardid1 + y, cardid2 + y})
		}
	}
	// fmt.Println("1111111111111:", groups)
	return groups
}

// 是否是出牌人的下家
func (r *Room) isCurrentCardUserNextUser(user *User) bool {
	currentCardUser := r.getCurrentCardsUser()
	if currentCardUser == nil {
		return false
	}
	currentCardUserIndex, userIndex := currentCardUser.getIndex(), user.getIndex()
	if currentCardUserIndex+1 == rule.PCount {
		if userIndex == 0 {
			return true
		}
	} else {
		if currentCardUserIndex+1 == userIndex {
			return true
		}
	}
	return false
}

// 吃检测
func (r *Room) chiCheck(user *User) bool {
	if !r.isCurrentCardUserNextUser(user) {
		return false
	}
	currentCard := r.getCurrentCard()
	groups := GetChiAllGroups(currentCard)
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
				return true
			}
		}
	}
	return false
}

// 杠检测(自摸)
func (r *Room) gangCheck() bool {
	user := r.getControllerUser()
	cardsid := user.getUserAllCardsID()
	tmpCards := []int{}
	for _, cardid := range cardsid {
		if len(tmpCards) > 0 && cardid != tmpCards[0] {
			tmpCards = []int{}
		}
		tmpCards = append(tmpCards, cardid)
		//红中不能杠
		if len(tmpCards) == 4 && cardid != HUN {
			handCardCount := 0 //手牌数量
			for _, card := range user.getCards() {
				if card.ID == tmpCards[0] {
					handCardCount++
				}
			}
			if handCardCount == 4 { //全部是手牌
				return true
			} else if handCardCount == 1 { //一张是手牌，三张在陈列区
				displayArea := user.getDisplayArea()
				if displayArea != "" {
					groups := strings.Split(displayArea, "$")
					//陈列区中的三张必须是刻字
					for _, group := range groups {
						count := 0
						cards := strings.Split(group, "#")
						for _, card := range cards {
							if strconv.Itoa(cardid) == strings.Split(card, "|")[0] {
								count++
							}
						}
						if count == 3 {
							return true
						}
					}
				}
				if true {
					return true
				}
			}
		}
	}
	return false
}

// 筛选听牌列表
func (r *Room) filtrateTingCards(cards []*Card) CardList {
	tingCards := CardList{}
	tingCards = append(tingCards, cards...)
	return tingCards
}

// 是否可能是七对,如果是返回缺的牌
func (r *Room) maybeQD(cards CardList) (bool, []int) {
	// return false, nil
	m := map[int]int{}
	for _, card := range cards {
		m[card.ID]++
	}
	deletionCardsID := []int{}
	duiCount := 0
	for cardid, count := range m {
		if count == 1 {
			deletionCardsID = append(deletionCardsID, cardid)
		} else if count == 2 {
			duiCount++
		}
	}
	if duiCount >= 3 {
		// fmt.Println("xxxxxxx", duiCount, deletionCardsID)
		return true, deletionCardsID
	}
	return false, nil
}

// 是否可能是十三不靠,如果是返回缺的牌
func (r *Room) maybeSSBK(cards CardList) (bool, []int) {
	mapCards := map[int]bool{}
	otherCards := []int{}
	for _, card := range cards {
		cardid := card.ID
		if isFJ(cardid) {
			mapCards[cardid] = true
		} else {
			otherCards = append(otherCards, cardid)
		}
	}
	cardCount := len(otherCards)
	wbtMaxCount := 0
	usedGroup := map[int]bool{}
	if cardCount > 0 {
		for _, group := range SSBKGroups {
			mapWBT := map[int]bool{}
			for _, cardid := range otherCards {
				if group[cardid] {
					mapWBT[cardid] = true
				}
			}
			count := len(mapWBT)
			if count > wbtMaxCount {
				wbtMaxCount = count
				usedGroup = map[int]bool{}
				for k, v := range group {
					usedGroup[k] = v
				}
			}
		}
	}
	if len(mapCards)+wbtMaxCount >= 10 {
		wordGroup := newWordGroup()
		for _, card := range cards {
			if usedGroup[card.ID] {
				delete(usedGroup, card.ID)
			}
			if wordGroup[card.ID] {
				delete(wordGroup, card.ID)
			}
		}
		deletionCardsID := []int{}
		for cardid, _ := range usedGroup {
			deletionCardsID = append(deletionCardsID, cardid)
		}
		for cardid, _ := range wordGroup {
			deletionCardsID = append(deletionCardsID, cardid)
		}
		return true, deletionCardsID
	}
	return false, nil
}

// 取出红中并获得可能变成的牌列表
func (r *Room) takeOutZhong(cards []*Card) (b bool, surplusCards []*Card, addCardsID []int) {
	surplusCards = []*Card{}
	addCardsID = []int{}
	playCard := cards[0]
	if playCard.ID != HUN {
		return false, surplusCards, addCardsID
	}
	// tmpCount++
	// fmt.Println("=======", tmpCount)
	surplusCards = append(surplusCards, cards[:0]...)
	surplusCards = append(surplusCards, cards[1:]...)
	if false {
		for _, card := range surplusCards {
			fmt.Print(card.ID, " ")
		}
		fmt.Println("")
	}
	// if b, deletionCardsID := r.maybeQD(surplusCards); b {
	// 	for _, cardid := range deletionCardsID {
	// 		if common.IndexIntOf(addCardsID, cardid) < 0 {
	// 			addCardsID = append(addCardsID, cardid)
	// 		}
	// 	}
	// } else
	if b, deletionCardsID := r.maybeSSBK(surplusCards); b {
		for _, cardid := range deletionCardsID {
			if common.IndexIntOf(addCardsID, cardid) < 0 {
				addCardsID = append(addCardsID, cardid)
			}
		}
		// fmt.Println("addCardsID:", addCardsID)
	} else {
		for _, card := range surplusCards {
			selfCardID := card.ID
			leftCardID, rightCardID := 0, 0
			cardType := card.Type
			if cardType == CardType_Myriad || cardType == CardType_Cake || cardType == CardType_Strip {
				if card.ID%10 > 1 {
					leftCardID = selfCardID - 1
					if common.IndexIntOf(addCardsID, leftCardID) < 0 {
						addCardsID = append(addCardsID, leftCardID)
					}
				}
				if common.IndexIntOf(addCardsID, selfCardID) < 0 {
					addCardsID = append(addCardsID, selfCardID)
				}
				if card.ID%10 < 9 {
					rightCardID = selfCardID + 1
					if common.IndexIntOf(addCardsID, rightCardID) < 0 {
						addCardsID = append(addCardsID, rightCardID)
					}
				}
			} else if cardType == CardType_Wind || cardType == CardType_Arrow {
				if common.IndexIntOf(addCardsID, selfCardID) < 0 {
					addCardsID = append(addCardsID, selfCardID)
				}
			}
			if ETA {
				surplusCardIDs := []int{}
				for _, card := range surplusCards {
					surplusCardIDs = append(surplusCardIDs, card.ID)
				}
				meaningfulCardsID := []int{}
				for _, cardid := range addCardsID {
					if r.cardMeaningful(surplusCardIDs, cardid) {
						meaningfulCardsID = append(meaningfulCardsID, cardid)
					}
				}
				addCardsID = meaningfulCardsID
			}
		}
	}
	return true, surplusCards, addCardsID
}

// 牌有没有意义
func (r *Room) cardMeaningful(cardids []int, cardid int) bool {
	leftCount := 0
	lrCount := 0
	rightCount := 0
	for _, thecardid := range cardids {
		if thecardid == cardid {
			return true
		}
		if thecardid == cardid-2 || thecardid == cardid-1 {
			leftCount++
		}
		if thecardid == cardid-1 || thecardid == cardid+1 {
			lrCount++
		}
		if thecardid == cardid+1 || thecardid == cardid+2 {
			rightCount++
		}
	}
	if leftCount >= 2 || lrCount >= 2 || rightCount >= 2 {
		return true
	}
	return false
	// cardCount := len(cardids)
	// count := 0
	// m := map[int]int{}
	// for _, thecardid := range cardids {
	// 	m[thecardid]++
	// 	if thecardid == cardid {
	// 		count++
	// 	}
	// }
	// //超过4张
	// if count >= 4 {
	// 	return false
	// }
	// //当将的牌的数量
	// jiangCount := 0
	// for _, c := range m {
	// 	if c == 2 {
	// 		jiangCount++
	// 	}
	// }
	// if jiangCount >= 2 {
	// 	return false
	// }
	// return true
}

// 牌有没有意义
func (r *Room) cardMeaningful_back(cardids []int, cardid int) bool {
	// fmt.Println(cardids, cardid)
	cardCount := len(cardids)
	count := 0
	m := map[int]int{}
	for _, thecardid := range cardids {
		m[thecardid]++
		if thecardid == cardid {
			count++
		}
	}
	//超过4张
	if count >= 4 {
		return false
	}
	//当将的牌的数量
	jiangCount := 0
	for _, c := range m {
		if c == 2 {
			jiangCount++
		}
	}
	if count == 0 {
		return true
		//如果能增加有效牌组合数量就有意义
		// ids := []int{}
		// for _, thecardid := range cardids {
		// 	if thecardid/10 == cardid/10 {
		// 		ids = append(ids, thecardid)
		// 	}
		// }
		// beforeShunCount := getShunCount(ids)
		// fmt.Println("前：", ids)
		// ids = append(ids, cardid)
		// fmt.Println("后：", ids)
		// afterShunCount := getShunCount(ids)
		// fmt.Println(beforeShunCount, afterShunCount)
		// if beforeShunCount == afterShunCount {
		// 	return false
		// }
	} else if count == 1 {
		//11 12 21 22 23 23 23
		if jiangCount == 0 { //没有将
			//凑将（是字直接凑，是万饼条需要看看有没有拆散了顺子）
			index := common.IndexIntOf(cardids, cardid)
			if isWBT(cardid) {
				//左边第2张
				left2 := 0
				if i := index - 2; i >= 0 {
					left2 = cardids[i]
				}
				//左边第1张
				left1 := 0
				if i := index - 1; i >= 0 {
					left1 = cardids[i]
				}
				//右边第1张
				right1 := 0
				if i := index + 1; i <= cardCount-1 {
					right1 = cardids[i]
				}
				//右边第2张
				right2 := 0
				if i := index + 2; i <= cardCount-1 {
					right2 = cardids[i]
				}
				// 13 14  27	15*2-3
				// 14 16  30 	15*2
				// 16 17  33	15*2+3
				// fmt.Println(left2, left1, right1, right2)
				// fmt.Println(cardid*2-(left1+left2), cardid*2-(left1+right1), cardid*2-(right1+right2))
				//拆散了顺子
				if cardid*2-(left1+left2) == 3 || cardid*2-(left1+right1) == 0 || cardid*2-(right1+right2) == -3 {
					return false
				}
			} else {
				return true
			}
		} else {
			//不是单独的
			// if !isSolitary(cardids, cardid) {
			// 	return false
			// }
		}
	} else if count == 2 {
		//11 11 ? 12 13
		//只有一个将
		if jiangCount <= 1 { //新加的牌跟唯一的将是同一张牌
			//如果能增加有效牌组合数量就有意义
			if isWBT(cardid) {
				ids := []int{}
				for _, thecardid := range cardids {
					if thecardid/10 == cardid/10 {
						if thecardid != cardid {
							ids = append(ids, thecardid)
						}
					}
				}
				beforeShunCount := getShunCount(ids)
				ids = append(ids, cardid)
				afterShunCount := getShunCount(ids)
				if beforeShunCount == afterShunCount {
					return false
				}
			} else {
				return false
			}
		}
	}
	return true
}

// 是否是单独的
func isSolitary(cardids []int, cardid int) bool {
	//单独的
	solitary := true
	if isWBT(cardid) {
		//左面的牌
		left := cardid - 1
		//右面的牌
		right := cardid + 1
		for _, cardid := range cardids {
			if cardid == left || cardid == right {
				solitary = false
				break
			}
		}
	} else {
		solitary = true
	}
	return solitary
}

func getShunCount(ids []int) int {
	// sort.Ints(ids)
	c := 0
	//取刻字
	// m := map[int]int{}
	// for _, id := range ids {
	// 	m[id]++
	// }
	// surIDS := []int{}
	// for k, v := range m {
	// 	if v < 3 {
	// 		for i := 0; i < v; i++ {
	// 			surIDS = append(surIDS, k)
	// 		}
	// 	} else {
	// 		c++
	// 	}
	// }
	// ids = surIDS
	// fmt.Println("刻字数量：", c, ids)
	//取顺子
	var f func([]int)
	f = func(ids []int) {
		sort.Ints(ids)
		// fmt.Println("*****:", ids)
		if len(ids) == 0 {
			return
		}
		x := -1
		y := 0
		storeIDS := []int{}
		for _, id := range ids {
			if x == -1 {
				x = id
				y++
			} else {
				if id-x == 1 {
					x = id
					y++
					// fmt.Println("aaaaa:", id, y)
					if y == 3 {
						c++
						//删除这三个
						surIDS := []int{}
						surIDS = append(surIDS, ids[3+len(storeIDS):]...)
						// fmt.Println("xxxxx", len(surIDS), surIDS)
						surIDS = append(surIDS, storeIDS...)
						f(surIDS)
						break
					}
				} else if id-x == 0 {
					// fmt.Println("bbbbb")
					storeIDS = append(storeIDS, id)
				} else {
					//删除第一个
					surIDS := []int{}
					surIDS = append(surIDS, ids[1+len(storeIDS):]...)
					surIDS = append(surIDS, storeIDS...)
					// fmt.Println("cccc", surIDS)
					f(surIDS)
					break
				}
			}
		}
	}
	f(ids)
	return c
}

var tmpCount int = 0
var tmpCount2 int = 0

// 听检测
func (r *Room) tingCheck(cards []*Card, user *User) (tingStatus bool) {
	log := false
	playCardsID := []string{}
	tingCardsIDList := []string{}
	// buff := bytes.Buffer{}
	//是否是闲家检测天听
	isPlayerCheckTianTing := user != r.getBanker() && r.getPlayCardCount() == 0
	//判断是否检测过的键值对
	var mapCards map[string]bool = map[string]bool{}
	//听牌检测
	check := func(cards []*Card) {
		tingGroup := r.filtrateTingCards(cards)
		str := ""
		for _, card := range tingGroup {
			str = fmt.Sprintf("%s|%d", str, card.ID)
		}
		if mapCards[str] {
			return
		}
		// fmt.Println(str)
		mapCards[str] = true
		tmpCount++
		// fmt.Println("=======", tmpCount)
		r.orderCards(tingGroup)
		if log {
			for _, card := range tingGroup {
				fmt.Print(card.ID, " ")
			}
			fmt.Println("")
		}
		playCards := CardList{}
		tingCards := []CardList{}
		for i := 0; i < len(tingGroup); i++ {
			playCard := tingGroup[i]
			if i > 0 {
				//当前牌跟前一张牌一样
				if playCard.ID == tingGroup[i-1].ID {
					continue
				}
			}
			haveHu := false
			if log {
				fmt.Println("")
				fmt.Print("打出：", playCard.ID, " 剩余：")
			}
			surplusCards := []*Card{}
			if isPlayerCheckTianTing {
				// fmt.Println("不需要移除一张牌了，因为本来就少一张")
				surplusCards = append(surplusCards, tingGroup...)
			} else {
				surplusCards = append(surplusCards, tingGroup[:i]...)
				surplusCards = append(surplusCards, tingGroup[i+1:]...)
			}
			if log {
				for _, card := range surplusCards {
					fmt.Print(card.ID, " ")
				}
				fmt.Println("")
			}
			addCardsID := []int{}
			// if b, deletionCardsID := r.maybeQD(surplusCards); b {
			// 	for _, cardid := range deletionCardsID {
			// 		if common.IndexIntOf(addCardsID, cardid) < 0 {
			// 			addCardsID = append(addCardsID, cardid)
			// 		}
			// 	}
			// } else
			if b, deletionCardsID := r.maybeSSBK(surplusCards); b {
				for _, cardid := range deletionCardsID {
					if common.IndexIntOf(addCardsID, cardid) < 0 {
						addCardsID = append(addCardsID, cardid)
					}
				}
			} else {
				for _, card := range surplusCards {
					selfCardID := card.ID
					leftCardID, rightCardID := 0, 0
					cardType := card.Type
					if cardType == CardType_Myriad || cardType == CardType_Cake || cardType == CardType_Strip {
						if card.ID%10 > 1 {
							leftCardID = selfCardID - 1
							if common.IndexIntOf(addCardsID, leftCardID) < 0 {
								addCardsID = append(addCardsID, leftCardID)
							}
						}
						if common.IndexIntOf(addCardsID, selfCardID) < 0 {
							addCardsID = append(addCardsID, selfCardID)
						}
						if card.ID%10 < 9 {
							rightCardID = selfCardID + 1
							if common.IndexIntOf(addCardsID, rightCardID) < 0 {
								addCardsID = append(addCardsID, rightCardID)
							}
						}
					} else if cardType == CardType_Wind || cardType == CardType_Arrow {
						if common.IndexIntOf(addCardsID, selfCardID) < 0 {
							addCardsID = append(addCardsID, selfCardID)
						}
					}
				}
				if ETA {
					surplusCardIDs := []int{}
					for _, card := range surplusCards {
						surplusCardIDs = append(surplusCardIDs, card.ID)
					}
					meaningfulCardsID := []int{}
					// fmt.Println("前:", addCardsID)
					for _, cardid := range addCardsID {
						// fmt.Println("aaaaa", cardid)
						if r.cardMeaningful(surplusCardIDs, cardid) {
							// fmt.Println("bbbbb", cardid)
							meaningfulCardsID = append(meaningfulCardsID, cardid)
						}
					}
					addCardsID = meaningfulCardsID
					// fmt.Println("后:", addCardsID)
				}
			}

			if log {
				fmt.Println("")
				fmt.Print("有可能胡的牌：")
				for _, cardid := range addCardsID {
					fmt.Print(cardid, " ")
				}
				fmt.Println("")
			}
			tingCardsID := []string{}
			// fmt.Println("*************", addCardsID)
			for _, cardid := range addCardsID {
				newCards := CardList{}
				newCards = append(newCards, surplusCards...)
				//要添加的牌手中已满4张
				count := 0
				for _, card := range surplusCards {
					if card.ID == cardid {
						count++
					}
				}
				if count == 4 {
					continue
				}
				newCard := NewCard(cardid)
				newCards = append(newCards, newCard)
				r.orderCards(newCards)
				if log {
					fmt.Println("")
					fmt.Print("新牌型：")
				}
				count++
				if log {
					for _, card := range newCards {
						fmt.Print(card.ID, " ")
					}
					fmt.Println("")
				}
				valid := r.huCheck(newCards)
				if valid {
					if log {
						fmt.Println("胡啦")
					}
					if !haveHu {
						haveHu = true
						playCards = append(playCards, playCard)
						playCardsID = append(playCardsID, strconv.Itoa(playCard.ID))
					}
					if len(playCards) > len(tingCards) {
						tingCards = append(tingCards, CardList{})
					}
					tingCards[len(tingCards)-1] = append(tingCards[len(tingCards)-1], newCard)
					user.playTingSolidifyCards[fmt.Sprintf("%d-%d", playCard.ID, newCard.ID)] = newCards
					cardsid := r.getUserSimulateTingCardAllCardsID(user, newCards)
					tingCardsID = append(tingCardsID, fmt.Sprintf("%d|%d|%d", newCard.ID, user.getCardSurplusCount(newCard.ID), r.getHuCardMultiple(user, cardsid)))
					// fmt.Println("长度", len(playCards), len(tingCards))
					// return true, nil, nil
					// fmt.Println("")
					// fmt.Print("打出牌列表：")
					// for _, card := range playCards {
					// 	fmt.Print(card.ID, " ")
					// }
					// fmt.Println("")
					// fmt.Print("听牌列表：")
					// for _, cardList := range tingCards {
					// 	for _, card := range cardList {
					// 		fmt.Print(card.ID, " ")
					// 	}
					// 	fmt.Print(",")
					// }
					// fmt.Println("")
				}
			}
			if len(tingCardsID) > 0 {
				tingCardsIDList = append(tingCardsIDList, strings.Join(tingCardsID, "$"))
				// buff.WriteString( + "#")
			}
			//闲家检测天听,只需检测当前手牌就可以了,不需要轮训删除牌
			if isPlayerCheckTianTing {
				break
			}
		}
		// fmt.Println("计算量：", count, " 胡法：", len(playCards))
		// fmt.Print("打出牌列表：")
		// for _, card := range playCards {
		// 	fmt.Print(card.ID, " ")
		// }
		// fmt.Println("")
		// fmt.Print("听牌列表：")
		// for _, cardList := range tingCards {
		// 	for _, card := range cardList {
		// 		fmt.Print(card.ID, " ")
		// 	}
		// 	fmt.Print(",")
		// }
		// fmt.Println("")

	}
	//红中实体化
	var zhongSolidify func([]*Card) bool
	zhongSolidify = func(cards []*Card) bool {
		b, surplusCards, addCardsID := r.takeOutZhong(cards)
		if b {
			// fmt.Println("ting添加的牌列表：", addCardsID)
			for _, cardid := range addCardsID {
				newCards := CardList{}
				newCards = append(newCards, surplusCards...)
				newCard := NewCard(cardid)
				newCard.IsMix = 1
				newCards = append(newCards, newCard)
				r.orderCards(newCards)
				zhongSolidify(newCards)
			}
			return true
		} else {
			//听牌检测
			check(cards)
			return false
		}
	}
	zhongSolidify(cards)
	//存入玩家听牌信息中
	tingStatus = len(playCardsID) > 0
	if tingStatus {
		strA := strings.Join(playCardsID, "|")
		strB := strings.Join(tingCardsIDList, "#")
		// fmt.Println("=====", strA)
		// fmt.Println("=====", strB)
		nodupPlayCardsID := []string{}
		nodupTingCardsIDList := []string{}
		for i, playCardsID := range playCardsID {
			index := common.IndexStringOf(nodupPlayCardsID, &playCardsID)
			if index < 0 {
				nodupPlayCardsID = append(nodupPlayCardsID, playCardsID)
				nodupTingCardsIDList = append(nodupTingCardsIDList, tingCardsIDList[i])
			} else {
				mapCardsID := map[string]bool{}
				for _, v := range strings.Split(nodupTingCardsIDList[index], "$") {
					mapCardsID[v] = true
				}
				for _, v := range strings.Split(tingCardsIDList[i], "$") {
					mapCardsID[v] = true
				}
				list := ""
				for k, _ := range mapCardsID {
					list = list + k + "$"
				}
				nodupTingCardsIDList[index] = r.orderTingCards(list[:len(list)-1])
			}
		}
		strA = strings.Join(nodupPlayCardsID, "|")
		strB = strings.Join(nodupTingCardsIDList, "#")
		// fmt.Println("=====", strA)
		// fmt.Println("=====", strB)
		if isPlayerCheckTianTing {
			strA = ""
		}
		user.setTingGroupsInfo(fmt.Sprintf("%s@%s", strA, strB))
		// fmt.Println("玩家听牌信息", user.getTingGroupsInfo())
	}
	return tingStatus
}

type TingCard struct {
	CardID       int
	SurplusCount int
	Multiple     int
}

type TingCardList []*TingCard

func (list TingCardList) Len() int {
	return len(list)
}

func (list TingCardList) Less(i, j int) bool {
	iV := list[i].Multiple*100 - list[i].CardID
	jV := list[j].Multiple*100 - list[j].CardID
	if iV > jV {
		return true
	} else {
		return false
	}
}

func (list TingCardList) Swap(i, j int) {
	var temp *TingCard = list[i]
	list[i] = list[j]
	list[j] = temp
}

// 排序听牌的信息
func (r *Room) orderTingCards(str string) string {
	tingCardList := TingCardList{}
	arr := strings.Split(str, "$")
	for _, s := range arr {
		arr2 := common.StrArrToIntArr(strings.Split(s, "|"))
		cardid, surplusCount, multiple := arr2[0], arr2[1], arr2[2]
		repeated := false
		for i, tingCard := range tingCardList {
			if tingCard.CardID == cardid {
				repeated = true
				if multiple > tingCard.Multiple {
					tingCardList[i].Multiple = multiple
				}
				break
			}
		}
		if !repeated {
			tingCard := &TingCard{
				CardID:       cardid,
				SurplusCount: surplusCount,
				Multiple:     multiple,
			}
			tingCardList = append(tingCardList, tingCard)
		}
	}
	sort.Sort(tingCardList)
	arr3 := []string{}
	for _, tingCard := range tingCardList {
		arr3 = append(arr3, fmt.Sprintf("%d|%d|%d", tingCard.CardID, tingCard.SurplusCount, tingCard.Multiple))
	}
	newStr := strings.Join(arr3, "$")
	return newStr
}

// 排序
func (r *Room) orderCards(cards CardList) {
	//排序
	sort.Sort(cards)
	//修改索引
	for i := 0; i < len(cards); i++ {
		cards[i].Index = i
	}
}

// 胡检测
func (r *Room) HuCheck(theCards []*Card) bool {
	b := r.huCheck(theCards)
	return b
}

// 胡检测
func (r *Room) huCheck(theCards []*Card) (huStatus bool) {
	// return false
	log := false
	check := func(cards []*Card) {
		cardids := []int{}
		for _, card := range cards {
			cardids = append(cardids, card.ID)
		}
		// fmt.Println("======", cardids)
		u := &User{}
		if u.getSSBK(cardids) {
			huStatus = true
		} else if u.getQD(cardids) {
			huStatus = true
		} else if len(cards) == 2 {
			if cards[0].ID == cards[1].ID {
				huStatus = true
			}
		} else {
			tmpCards := []*Card{}
			for _, card := range cards {
				if len(tmpCards) > 0 && tmpCards[0].ID != card.ID {
					tmpCards = []*Card{card}
					continue
				}
				tmpCards = append(tmpCards, card)
				if len(tmpCards) == 2 {
					if log {
						fmt.Println("将牌：", tmpCards[0].ID)
					}
					surplusCards := []*Card{}
					for _, card := range cards {
						if card != tmpCards[0] && card != tmpCards[1] {
							surplusCards = append(surplusCards, card)
						}
					}
					groupCount := len(surplusCards) / 3
					validCount := 0
					padding := 0
					for {
						groupCards := []*Card{}
						groupCardsIndexs := []int{}
						if log {
							fmt.Println("")
							for _, card := range surplusCards {
								fmt.Print(" aaa-", card.ID)
							}
							fmt.Println("")
						}
						for i, card := range surplusCards {
							if len(surplusCards) != padding {
								if i < padding {
									continue
								}
							}
							addCard := func() {
								groupCards = append(groupCards, card)
								groupCardsIndexs = append(groupCardsIndexs, i)
							}
							groupLen := len(groupCards)
							if groupLen == 0 {
								addCard()
							} else {
								if groupLen < 3 {
									if card.ID-groupCards[0].ID > 2 {
										padding++
										break
									}
									if groupLen == 1 {
										addCard()
									} else if groupLen == 2 {
										if groupCards[0].ID == groupCards[1].ID { //刻字
											if card.ID == groupCards[1].ID {
												addCard()
											}
										} else {
											if isMyriadCakeStrip(card) {
												if groupCards[0].ID == groupCards[1].ID-1 {
													if card.ID == groupCards[1].ID+1 {
														addCard()
													}
												}
											}
										}
									}
								}
								groupLen = len(groupCards)
								if groupLen >= 3 {
									padding = 0
									validCount++
									for i := groupLen - 1; i >= 0; i-- {
										removeIndex := groupCardsIndexs[i]
										surplusCards = append(surplusCards[:removeIndex], surplusCards[removeIndex+1:]...)
									}
									break
								} else {
									if i+1 == len(surplusCards) {
										padding++
										break
									}
								}
							}
							//最后一张牌
							if i+1 == len(surplusCards) {
								surplusCards = []*Card{}
							}
						}
						if log {
							fmt.Println("")
							for _, card := range surplusCards {
								fmt.Print(" bbb-", card.ID)
							}
							fmt.Println("")
						}
						if len(surplusCards) == 0 {
							break
						}
					}
					if log {
						fmt.Println("===", groupCount, validCount)
					}
					if groupCount == validCount {
						// r.setJiangCard(tmpCards[0])
						huStatus = true
					}
				}
			}
		}
	}
	cards := CardList{}
	cards = append(cards, theCards...)
	r.orderCards(cards)
	//红中实体化
	var zhongSolidify func([]*Card)
	zhongSolidify = func(cards []*Card) {
		if huStatus {
			return
		}
		b, surplusCards, addCardsID := r.takeOutZhong(cards)
		if b {
			// fmt.Println("hu添加的牌列表：", addCardsID)
			for _, cardid := range addCardsID {
				newCards := CardList{}
				newCards = append(newCards, surplusCards...)
				newCard := NewCard(cardid)
				newCards = append(newCards, newCard)
				r.orderCards(newCards)
				zhongSolidify(newCards)
			}
			return
		} else {
			//胡牌检测
			check(cards)
			return
		}
	}
	zhongSolidify(cards)
	return huStatus
}

// 获取操作倒计时
func (r *Room) getWaitTime() int {
	waitTime := r.playWaitTime
	return waitTime
}

// 获取下一顺位出牌人
func (r *Room) getNextUser() *User {
	controllerUser := r.getControllerUser()
	users := r.getUsers()
	index := controllerUser.getIndex()
	var user *User
	for i := 0; i < rule.PCount-1; i++ {
		index = getNextUserIndex(index)
		u := users[index]
		//不是出牌人
		if u != controllerUser {
			user = u
			break
		}
	}
	return user
}

// 获取下一个玩家index
func getNextUserIndex(index int) int {
	index = index + 1
	if index > rule.PCount-1 {
		index = 0
	}
	return index
}

// 获取房间匹配信息
func (r *Room) getRoomMatchingInfo() ([]*User, []string) {
	//已落座和未落座的玩家集合
	allUsers := r.getAllUsers()
	//获取已落座的玩家状态
	getUsersStatuss := func() string {
		bfStatuss := bytes.Buffer{}
		for _, user := range r.getUsers() {
			if user != nil {
				userInfo, err := redis.Ints(user.GetUserInfo("vip", "level"))
				logger.CheckError(err)
				vip, level := userInfo[0], userInfo[1]
				bfStatuss.WriteString(fmt.Sprintf("%d$%d$%d$%d$%d|", user.GetMemberid(), user.getStatus(), vip, level, user.getRoundIntegral()))
			} else {
				bfStatuss.WriteString("|")
			}
		}
		strStatuss := bfStatuss.String()
		strStatuss = strStatuss[:len(strStatuss)-1]
		return strStatuss
	}
	//获取未落座的玩家状态
	getIdleUsersStatuss := func() string {
		bfStatuss := bytes.Buffer{}
		for _, user := range r.getIdleUsers() {
			bfStatuss.WriteString(fmt.Sprintf("%d|", user.GetMemberid()))
		}
		strStatuss := bfStatuss.String()
		if strStatuss != "" {
			strStatuss = strStatuss[:len(strStatuss)-1]
		}
		return strStatuss
	}
	roomMasterUserID := 0
	user := r.getRoomMaster()
	if user != nil {
		roomMasterUserID = user.GetMemberid()
	}
	statuss := []string{fmt.Sprintf("%s#%s#%d#%d", getUsersStatuss(), getIdleUsersStatuss(), r.getInning(), roomMasterUserID)}
	return allUsers, statuss
}
