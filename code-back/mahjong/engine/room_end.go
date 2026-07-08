/*
房间比赛结束
*/

package engine

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"combine.com/utils/types"

	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
)

var (
	addMultipleCardMultiple = 8  //加番牌
	arrowWindCardMultiple   = 10 //风箭
	awardFlowerCardMultiple = 5  //奖花
)

func RefreshValue() {
	cardtypes_str := ""
	memberDB := frame.GetMemberDB()
	rows, _ := memberDB.Query("select value from config_games where platform=? and name in('addmultiple','repairflower','arrowwind','awardflower','cardtypes');", PLATFORM)
	defer rows.Close()
	index := 0
	for rows.Next() {
		if index == 0 {
			rows.Scan(&addMultipleCardMultiple)
		} else if index == 1 {
			// rows.Scan(&repairFlowerCardMultiple)
		} else if index == 2 {
			rows.Scan(&arrowWindCardMultiple)
		} else if index == 3 {
			// rows.Scan(&awardFlowerCardMultiple)
		} else if index == 4 {
			rows.Scan(&cardtypes_str)
		}
		index++
	}
	cardtypes_arr := strings.Split(cardtypes_str, "|")
	cardtypes := common.StrArrToIntArr(cardtypes_arr)
	CardTypeGroup_Multiples = cardtypes
}

func Test_checkMatchingOver() {
	// uid1, uid2, uid3, uid4, uid5 := "h1", "h2", "h3", "h4", "h5"
	// userid1, userid2, userid3, userid4, userid5 := "24196", "24197", "24198", "24199", "24200"
	// user1 := UserManage.AddUser(&uid1, &userid1, nil)
	// user2 := UserManage.AddUser(&uid2, &userid2, nil)
	// user3 := UserManage.AddUser(&uid3, &userid3, nil)
	// user4 := UserManage.AddUser(&uid4, &userid4, nil)
	// user5 := UserManage.AddUser(&uid5, &userid5, nil)
	// room := RoomManage.AddRoom()
	// user1.setRoom(room)
	// user1.setRoleType(ROLETYPE_EMPEROR)
	// user2.setRoom(room)
	// user3.setRoom(room)
	// user4.setRoom(room)
	// user5.setRoom(room)
	// room.setMatchID(Match_JD)
	// room.users = []*User{user1, user2, user3, user4, user5}
	// room.ranking = []*User{nil, nil, nil, nil, nil}
	// room.setRanking(user1)
	// room.setRanking(user2)
	// room.setRanking(user3)
	// room.setRanking(user4)
	// room.setRanking(user5)
	// room.setEmperor(user1)
	// //	room.setJack(user5)
	// //	room.setJackCard(&Card{})
	// //	room.setProtectStatus(PROTECTSTATUS_MB)
	// room.checkMatchingOver()
}

var counter = 0

// 比赛结束的检测
func (r *Room) checkMatchingOver() {
	//设置比赛结束
	r.SetRoomState(frame.ROOMSTATE_OVER)
	//比赛结束处理
	go r.matchingOverHandle()
}

/*
比赛结束的处理
*/
func (r *Room) matchingOverHandle() {
	defer func() {
		if p := recover(); p != nil {
			errInfo := fmt.Sprintf("matchingOverHandle : { %v }", p)
			logger.Errorf(errInfo)
		}
	}()
	//展示剩余牌面
	r.showSurplusCards()
	//番数处理
	r.multipleHandle()
	//推送比赛结束的信息给所有玩家
	r.pushMatchingEndInfo()
}

// 获取玩家模拟听牌所有牌列表(包括陈列区)
func (r *Room) getUserSimulateTingCardAllCardsID(user *User, simulateHandCard CardList) []int {
	allCardsID := []int{}
	for _, card := range simulateHandCard {
		allCardsID = append(allCardsID, card.ID)
	}
	displayArea := user.getDisplayArea()
	if displayArea != "" {
		displayAreaInfo := strings.Split(displayArea, "$")
		for _, groupInfo := range displayAreaInfo {
			group := strings.Split(groupInfo, "#")[0]
			cardsid := common.StrArrToIntArr(strings.Split(group, "|"))
			allCardsID = append(allCardsID, cardsid...)
		}
	}
	//将牌排序
	sort.Ints(allCardsID)
	// fmt.Println("玩家所有牌(模拟听牌)：", allCardsID)
	return allCardsID
}

type HuWay struct {
	Cards    []*Card
	JiangPai *Card
	Multiple int
}

func (this *HuWay) isHu() bool {
	return this.Multiple > 0
}

type HuWayList []*HuWay

func (list HuWayList) Len() int {
	return len(list)
}

func (list HuWayList) Less(i, j int) bool {
	iV := list[i].Multiple
	jV := list[j].Multiple
	if iV > jV {
		return true
	} else {
		return false
	}
}

func (list HuWayList) Swap(i, j int) {
	var temp *HuWay = list[i]
	list[i] = list[j]
	list[j] = temp
}

// 计算胡牌最大番
func (r *Room) calculateHuMaxMultiple(u *User, addCard *Card) *HuWay {
	log := false
	theCards := CardList{}
	theCards = append(theCards, u.getCards()...)
	if addCard != nil {
		theCards = append(theCards, addCard)
	}
	//陈列区
	displayArea := u.getDisplayArea()
	displayAreaCards := CardList{}
	if displayArea != "" {
		displayAreaInfo := strings.Split(displayArea, "$")
		for _, groupInfo := range displayAreaInfo {
			group := strings.Split(groupInfo, "#")[0]
			cardsid := common.StrArrToIntArr(strings.Split(group, "|"))
			for _, cardid := range cardsid {
				card := NewCard(cardid)
				displayAreaCards = append(displayAreaCards, card)
			}
		}
	}
	//判断是否检测过的键值对
	var mapCards map[string]bool = map[string]bool{}
	//胡牌的方式列表
	huWayList := HuWayList{}
	//当手牌可以胡牌的时候,检测胡牌的方式
	check := func(cards []*Card) {
		str := ""
		for _, card := range cards {
			str = fmt.Sprintf("%s|%d", str, card.ID)
		}
		if mapCards[str] {
			return
		}
		mapCards[str] = true
		cardids := []int{}
		for _, card := range cards {
			cardids = append(cardids, card.ID)
		}
		addWay := func(cards []*Card, jiangPai *Card) {
			huWay := &HuWay{}
			huWay.Cards = cards
			huWay.JiangPai = jiangPai
			cardsid := r.getUserSimulateTingCardAllCardsID(u, cards)
			huWay.Multiple = r.getHuCardMultiple(u, cardsid)
			huWayList = append(huWayList, huWay)
		}
		if u.getSSBK(cardids) {
			addWay(cards, nil)
		} else if u.getQD(cardids) {
			addWay(cards, nil)
		} else if len(cards) == 2 {
			if cards[0].ID == cards[1].ID {
				addWay(cards, cards[0])
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
						jiangPai := tmpCards[0]
						addWay(cards, jiangPai)
					}
				}
			}
		}
	}
	cards := CardList{}
	cards = append(cards, theCards...)
	r.orderCards(cards)
	//红中实体化
	var maybeCardListSet []CardList = []CardList{}
	var zhongSolidify func([]*Card)
	zhongSolidify = func(cards []*Card) {
		b, surplusCards, addCardsID := r.takeOutZhong(cards)
		if b {
			for _, cardid := range addCardsID {
				newCards := CardList{}
				newCards = append(newCards, surplusCards...)
				newCard := NewCard(cardid)
				newCard.IsMix = 1
				newCards = append(newCards, newCard)
				r.orderCards(newCards)
				zhongSolidify(newCards)
			}
		} else {
			maybeCardListSet = append(maybeCardListSet, cards)
		}
	}
	zhongSolidify(cards)
	for _, cardList := range maybeCardListSet {
		if r.huCheck(cardList) {
			check(cardList)
		}
	}
	sort.Sort(huWayList)
	huWay := &HuWay{}
	if len(huWayList) > 0 {
		huWay = huWayList[0]
		if PRINT_CRUX_LOG {
			cardsid := ""
			for _, card := range huWay.Cards {
				cardsid = fmt.Sprintf("%s-(%d.%d)", cardsid, card.ID, card.IsMix)
			}
			fmt.Println("胡牌牌型：", cardsid, huWay.JiangPai, huWay.Multiple)
		}
	}
	return huWay
	// for _, huWay := range huWayList {
	// 	cardsid := ""
	// 	for _, card := range huWay.Cards {
	// 		cardsid = fmt.Sprintf("%s-(%d.%d)", cardsid, card.ID, card.IsMix)
	// 	}
	// 	fmt.Println(cardsid, huWay.JiangPai, huWay.Multiple)
	// }
}

// 番数处理
func (r *Room) multipleHandle() {
	huUser := r.getHuUser()
	if huUser == nil {
		return
	}
	//计算胡牌最大番
	huWay := r.calculateHuMaxMultiple(huUser, nil)
	huUser.setCards(huWay.Cards)
	r.setJiangCard(huWay.JiangPai)
	//获得玩家所有的牌
	cardsid := r.getHuUser().getUserAllCardsID()
	//基本番
	huUser.multiple_base(cardsid)
	//牌型番
	huUser.multiple_cardType(cardsid)
	//加番牌
	huUser.multiple_addCard(cardsid)
	//奖花番
	// huUser.multiple_awardFlower(cardsid)
	// fmt.Println("基本番", huUser.getMultipleBase(), "补花番", huUser.getMultipleRepairFlower(), "牌型番", huUser.getMultipleCardType(), "加番牌", huUser.getMultipleAddCard(), "奖花番", huUser.getMultipleAwardFlower())
}

/*
ShowSurplusCards_Push(展示剩余牌面)
push:userid|,userid|1$1$16@

	userid#CardID|CardID|CardID$CardID|CardID|CardID,userid#CardID|CardID|CardID
*/
func (r *Room) showSurplusCards() {
	buffLeft := bytes.Buffer{}
	users := r.GetUsers()
	for _, user := range users {
		buffLeft.WriteString(fmt.Sprintf("%d|%s,", user.GetMemberid(), *user.getCardsID("$")))
	}
	leftStr := *common.RemoveLastChar(buffLeft)
	buffRight := bytes.Buffer{}
	for _, user := range users {
		memberid := user.getMemberid()
		kvs := types.KVS{frame.MEMBERID: memberid, "target": memberid}
		displayArea := (&Engine{}).GetDisplayArea(ctx, kvs)
		buffRight.WriteString(displayArea.GetMsg())
		buffRight.WriteString(",")
	}
	rightStr := *common.RemoveLastChar(buffRight)
	message := fmt.Sprintf("%s@%s", leftStr, rightStr)
	r.pushMessageToUsers("ShowSurplusCards_Push", []string{message}, r.getUsers())
}

// 处理番数
func (r *Room) handleMultiple() (multipleListInfo string, huCardMultiple int, multipleAwardFlower int, multipleAddCard int, sumMultiple int) {
	user := r.getHuUser()
	if user == nil {
		return
	}
	multipleList := MultipleList{}
	//加番牌
	multipleAddCard = user.getMultipleAddCard()
	if multipleAddCard > 0 {
		multipleList = append(multipleList, &Multiple{Name: "加番牌", Multiple: multipleAddCard})
	}
	//基本番
	multipleBase := user.getMultipleBase()
	multipleList = append(multipleList, &Multiple{Name: "基本番", Multiple: multipleBase})
	//没有混子
	noHaveMix := !user.haveMixCard()
	//牌型番
	cardTypeMultiple := 0
	for cardType, count := range user.getMultipleCardType() {
		if count > 0 {
			m := count * CardTypeGroup_Multiples[cardType]
			if !(r.GetMatchID() == frame.MATCHID_FRIEND && r.GetRoomType() == frame.ROOMTYPE_NORM) {
				if noHaveMix {
					m = m * 2
				}
			}
			cardTypeMultiple += m
			multipleList = append(multipleList, &Multiple{Name: CardTypeGroup_Names[cardType], Multiple: m})
		}
	}
	//补花番
	// multipleRepairFlower := user.getMultipleRepairFlower()
	// if multipleRepairFlower > 0 {
	// 	multipleList = append(multipleList, &Multiple{Name: "补花", Multiple: multipleRepairFlower})
	// }
	sort.Sort(multipleList)
	buff := bytes.Buffer{}
	for _, v := range multipleList {
		buff.WriteString(fmt.Sprintf("%s|%d番$", v.Name, v.Multiple))
	}
	multipleListInfo = *common.RemoveLastChar(buff)
	huCardMultiple = user.getMultipleBase() + cardTypeMultiple + multipleAddCard
	sumMultiple += huCardMultiple
	if r.getMatchResult() == MatchResult_ZiMo {
		sumMultiple = sumMultiple * 2
	}
	return
}

// 获取房间底分
func (r *Room) getRoomBaseIngot() int {
	baseIngot := 10
	roomData, _ := r.GetRoomData("baseingot")
	if len(roomData) > 0 {
		baseIngot = roomData[0]
	}
	return baseIngot
}

// 获取Memberids
func (r *Room) getMemberids() string {
	memberids := []int{}
	for _, user := range r.getUsers() {
		memberids = append(memberids, user.getMemberid())
	}
	return common.Join(memberids, ",")
}

// 获取胡牌玩家和点炮玩家的memberid
func (r *Room) getHuDianPaoUserMemberid() (huMemberID int, dianPaoMemberID int) {
	if r.getHuUser() != nil {
		huMemberID = r.getHuUser().getMemberid()
	}
	if r.getDianPaoUser() != nil {
		dianPaoMemberID = r.getDianPaoUser().getMemberid()
	}
	return
}

// 推送比赛结束的信息-经典
func (r *Room) pushMatchingEndInfo_JD() {
	//番数处理
	multipleListInfo, huCardMultiple, multipleAwardFlower, multipleAddCard, sumMultiple := r.handleMultiple()
	huMemberID, dianPaoMemberID := r.getHuDianPaoUserMemberid()
	rds := frame.GetBuildRds()
	values := []interface{}{}
	values = append(values, frame.MEMBERIDS, r.getMemberids())
	values = append(values, "multipleListInfo", multipleListInfo)
	values = append(values, "huCardMultiple", huCardMultiple)
	values = append(values, "multipleAwardFlower", multipleAwardFlower)
	values = append(values, "multipleAddCard", multipleAddCard)
	values = append(values, "sumMultiple", sumMultiple)
	values = append(values, "matchResult", r.getMatchResult())
	values = append(values, "huMemberID", huMemberID)
	values = append(values, "dianPaoMemberID", dianPaoMemberID)
	rds.HMSet(ctx, frame.GetRoomInfoKey(r.GetRoomID()), values...)
	//结算
	matchEnd(r.GetRoomID())
	//释放房间
	RoomManage.ReleaseRoom(r.GetRoomID())
	//切换build
	matchEndBuild(types.KVS{frame.ROOMID: r.GetRoomID(), frame.ENDMODE: frame.ENDMODE_DISSOLVE})
}

// 获取离线的玩家
func (r *Room) getOfflines() []int {
	offlines := []int{}
	for _, user := range r.GetUsers() {
		if !user.GetOnline() {
			offlines = append(offlines, user.GetMemberid())
		}
	}
	return offlines
}

// 获取比赛积分
func (r *Room) getIntegrals() string {
	integrals := []int{}
	for _, user := range r.GetUsers() {
		integrals = append(integrals, user.getRoundIntegral())
	}
	return common.Join(integrals, ",")
}

// 推送比赛结束的信息-好友同玩
func (r *Room) pushMatchingEndInfo_HYTW() {
	//番数处理
	multipleListInfo, huCardMultiple, multipleAwardFlower, multipleAddCard, sumMultiple := r.handleMultiple()
	huMemberID, dianPaoMemberID := r.getHuDianPaoUserMemberid()
	rds := frame.GetBuildRds()
	values := []interface{}{}
	values = append(values, frame.MEMBERIDS, r.getMemberids())
	values = append(values, "multipleListInfo", multipleListInfo)
	values = append(values, "huCardMultiple", huCardMultiple)
	values = append(values, "multipleAwardFlower", multipleAwardFlower)
	values = append(values, "multipleAddCard", multipleAddCard)
	values = append(values, "sumMultiple", sumMultiple)
	values = append(values, "matchResult", r.getMatchResult())
	values = append(values, "huMemberID", huMemberID)
	values = append(values, "dianPaoMemberID", dianPaoMemberID)
	values = append(values, frame.ROOMID, r.GetRoomID())
	values = append(values, frame.CREATETIME, r.GetCreateTime())
	values = append(values, frame.INNINGS, r.getInnings())
	values = append(values, frame.INNING, r.getInning())
	values = append(values, frame.INTEGRALS, r.getIntegrals())
	rds.HMSet(ctx, frame.GetRoomInfoKey(r.GetRoomID()), values...)
	//结算
	reply := matchEnd(r.GetRoomID())
	//更新好友同玩积分
	integrals := reply.GetInts(frame.INTEGRALS)
	//获取离线的玩家
	offlines := r.getOfflines()
	kvs := types.KVS{}
	kvs.Set(frame.MEMBERIDS, r.getMemberids())
	kvs.Set(frame.ROOMID, r.GetRoomID())
	kvs.Set(frame.INTEGRALS, integrals)
	kvs.Set(frame.OFFLINES, offlines)
	kvs.Set(frame.ENDMODE, frame.ENDMODE_FRIEND)
	//释放房间
	RoomManage.ReleaseRoom(r.GetRoomID())
	//切换build
	matchEndBuild(kvs)
}

// 推送比赛结束的信息-话费赛
func (r *Room) pushMatchingEndInfo_HFS() {
	//番数处理
	multipleListInfo, huCardMultiple, multipleAwardFlower, multipleAddCard, sumMultiple := r.handleMultiple()
	huMemberID, dianPaoMemberID := r.getHuDianPaoUserMemberid()
	rds := frame.GetBuildRds()
	values := []interface{}{}
	values = append(values, frame.MEMBERIDS, r.getMemberids())
	values = append(values, "multipleListInfo", multipleListInfo)
	values = append(values, "huCardMultiple", huCardMultiple)
	values = append(values, "multipleAwardFlower", multipleAwardFlower)
	values = append(values, "multipleAddCard", multipleAddCard)
	values = append(values, "sumMultiple", sumMultiple)
	values = append(values, "matchResult", r.getMatchResult())
	values = append(values, "huMemberID", huMemberID)
	values = append(values, "dianPaoMemberID", dianPaoMemberID)
	rds.HMSet(ctx, frame.GetRoomInfoKey(r.GetRoomID()), values...)
	//结算
	matchEnd(r.GetRoomID())
	//释放房间
	RoomManage.ReleaseRoom(r.GetRoomID())
	//切换build
	matchEndBuild(types.KVS{frame.ROOMID: r.GetRoomID(), frame.ENDMODE: frame.ENDMODE_DISSOLVE})
}

/*
MatchingEnd_Push(推送比赛结束的信息)
out:

	够级英雄: 等级|积分|经验|当前级别经验|当前级别升级经验|房费|比赛结果(1胜0平-1负)|是否升级|金币,userid$获得元宝$是否破产$buff道具列表$红蓝队$玩家等级|
	好友同玩: 等级|积分|第几局|是否是最后一局|红队积分|蓝队积分,userid$获得积分$是否胜利$红蓝队$玩家等级|
*/
func (r *Room) pushMatchingEndInfo() {
	if r.GetMatchID() == frame.MATCHID_NORM {
		r.pushMatchingEndInfo_JD()
	} else if r.GetMatchID() == frame.MATCHID_FRIEND {
		r.pushMatchingEndInfo_HYTW()
	} else if r.GetMatchID() == frame.MATCHID_HFS {
		r.pushMatchingEndInfo_HFS()
	}
	time.Sleep(time.Millisecond * 500)
}
