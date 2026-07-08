/*
房间开赛前
*/

package engine

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"combine.com/utils/common"
	"combine.com/utils/delay"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
)

// 初始化牌局
var (
	// InitCard = []string{
	// 	"15|15|15|16|16|17|18|19|32|33|34|71|21|22",
	// 	"",
	// 	"",
	// 	"",
	// }
	InitCard = []string{}
)

/*
推送房间的状态信息
push:Matching_Push,userid$玩家状态$是否是vip$等级$玩家积分|#等待席#当前轮次#房主userid
*/
func (r *Room) matchingPush(user *User) {
	//获取此房间的信息
	users, statuss := r.getRoomMatchingInfo()
	if user != nil {
		users = []*User{user}
	}
	r.pushMessageToUsers("Matching_Push", statuss, users)
}

/*
推送红包赛匹配信息
push:MatchingHBS_Push,64$1$0$5$0|||||#等待席#当前轮次
*/
func (r *Room) matchingHFSPush(user *User) {
	//获取此房间的信息
	users, statuss := r.getRoomMatchingInfo()
	if user != nil {
		users = []*User{user}
	}
	r.pushMessageToUsers("MatchingHFS_Push", statuss, users)
}

// 初始化房间中的玩家信息
func (r *Room) initUsersInfo() {
	for i, user := range r.users {
		user.setStatus(UserStatus_NoPass)
		user.setIndex(i)
		user.close_countDown_playCard()
	}
}

// 比赛开局
func (r *Room) match_Opening() {
	if r.GetRoomState() == frame.ROOMSTATE_DEAL {
		return
	}
	//设置房间为发牌状态
	r.SetRoomState(frame.ROOMSTATE_DEAL)
	f := func() {
		//初始化房间中的玩家信息
		r.initUsersInfo()
		//推送开赛牌局
		r.Opening_Push()
		//设置玩家基本番
		r.setUserMultipleBase()
		//加番牌
		r.doubleCard()
		//等待发牌
		r.dealTask = &delay.Task{
			Exec: func() {
				//开启补花操作
				r.openRepairFlowerHandle()
			},
			SurplusTime: r.getDealTime(),
		}
		r.dealTask.Start()
	}
	f()
}

// 设置玩家基本番
func (r *Room) setUserMultipleBase() {
	for _, user := range r.GetUsers() {
		user.multiple_base(nil)
	}
}

// 开启补花操作
func (r *Room) openRepairFlowerHandle() {
	r.SetRoomState(frame.ROOMSTATE_HANDLE)
	var wg sync.WaitGroup
	wg.Add(len(r.GetUsers()))
	for _, u := range r.GetUsers() {
		user_ := u
		go func(user *User) {
			defer func() {
				wg.Done()
				if p := recover(); p != nil {
					logger.Errorf("[recovery] openRepairFlowerHandle err : %v", p)
				}
			}()
			for {
				cards := user.getCards()
				flowerCount := 0
				newCards := append([]*Card{}, cards...)
				for i := len(cards) - 1; i >= 0; i-- {
					card := cards[i]
					if card.Type == CardType_Flower {
						flowerCount++
						//移除花牌
						newCards = append(newCards[:i], newCards[i+1:]...)
						//加入补花区
						user.addRepairFlowerArea(card.ID)
						//从牌池中取出一张牌
						newCard := r.getCardFromDeck()
						// fmt.Println("花牌：", card.ID, " 补牌：", newCard.ID)
						user.addMultipleRepairFlower(r.getBaseScore())
						//添加进手牌
						newCards = append(newCards, newCard)
						user.setCards(newCards)
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
									usersIngotChange[u.getMemberid()] = -expendIngot
									frame.UpdateIngot(u.getMemberid(), -expendIngot, 33, PLATFORM)
									sumIngot += expendIngot
								}
							}
							usersIngotChange[user.getMemberid()] = sumIngot
							frame.UpdateIngot(user.getMemberid(), sumIngot, 33, PLATFORM)
						}
						ingotChanges := []int{}
						for _, u := range r.GetUsers() {
							ingotChanges = append(ingotChanges, usersIngotChange[u.getMemberid()])
						}
						ingotChangesStr := common.Join(ingotChanges, "$")
						messages := []string{}
						message := ""
						for _, theUser := range r.GetUsers() {
							if theUser == user {
								message = fmt.Sprintf("%d|%d|%d|%s", user.getMemberid(), card.ID, newCard.ID, ingotChangesStr)
							} else {
								message = fmt.Sprintf("%d|%d||%s", user.getMemberid(), card.ID, ingotChangesStr)
							}
							messages = append(messages, message)
						}
						/*
							补花牌推送
							push:RepairFlower_Push,userid|花牌ID|新牌ID|元宝
							des:新牌ID(其它玩家是”“值)
						*/
						r.pushMessageToUsers("RepairFlower_Push", messages, r.GetUsers())
						r.pushDeckCount()
						time.Sleep(time.Second)
					}
				}
				if flowerCount == 0 {
					break
				}
			}
		}(user_)
	}
	wg.Wait()
	for _, user := range r.GetUsers() {
		//手牌排序
		user.orderCards()
	}
	//检测开赛
	r.checkStart()
}

// 推送剩余牌量
func (r *Room) pushDeckCount() {
	surplusCardCount := strconv.Itoa(len(r.getDeck()))
	/*
		牌池剩余牌数
		push:DeckCount_Push,牌池剩余牌数
	*/
	r.pushMessageToUsers("DeckCount_Push", []string{surplusCardCount}, r.GetUsers())
}

// 获取发牌时间
func (r *Room) getDealTime() time.Duration {
	dealTime := 9
	t, _ := time.ParseDuration(strconv.Itoa(dealTime) + "s")
	return t
}

// 推送开赛牌局
func (r *Room) Opening_Push() {
	//初始化
	r.initDeck()
	//随机庄家
	r.randomBanker()
	//生成所有人的牌
	cardsList := []CardList{}
	for {
		cardsList = r.generateCards()
		if len(cardsList) > 0 {
			break
		}
	}
	//牌面排序
	users := r.getUsers()
	messages := []string{}
	for i, user := range users {
		//排序
		sort.Sort(cardsList[i])
		//修改索引
		for k := 0; k < len(cardsList[i]); k++ {
			cardsList[i][k].Index = k
		}
		//设置玩家牌面
		user.setCards(cardsList[i])
		buffer := bytes.Buffer{}
		for _, card := range cardsList[i] {
			buffer.WriteString(fmt.Sprintf("%d|", card.ID))
		}
		str := *common.RemoveLastChar(buffer)
		messages = append(messages, str)
	}
	/*
		推送牌局
		push:Opening_Push,CardID|CardID
	*/
	r.pushMessageToUsers("Opening_Push", messages, users)
	time.Sleep(time.Millisecond * 10)
}

func (r *Room) initDeck() {
	//初始化整付牌
	roomDeck := []Card{}
	// roomDeck = append(roomDeck, deck...)
	if r.GetMatchID() == frame.MATCHID_FRIEND {
		if r.GetPlayType() == frame.MAHJONG_PLAY_TYPE_NOMIX {
			roomDeck = generateDeck_without_mix()
		} else {
			roomDeck = generateDeck()
		}
	} else {
		roomDeck = generateDeck()
	}
	r.setDeck(roomDeck)
	//初始化map牌池
	for _, card := range roomDeck {
		r.mapDeck[card.ID]++
	}
	r.setBankerTingPlayIndex(-1)
}

// 加番牌
func (r *Room) doubleCard() {
	return
	deck := r.getDeck()
	var addMultipleCard Card
	for {
		index := common.Random(0, len(deck))
		card := deck[index]
		if card.Type != CardType_Flower && card.ID != HUN {
			addMultipleCard = card
			break
		}
	}
	r.setMultipleCardID(addMultipleCard.ID)
	/*
		加番牌推送
		push:MultipleCard_Push,CardID|番数
	*/
	r.pushMessageToUsers("MultipleCard_Push", []string{fmt.Sprintf("%d|%d", addMultipleCard.ID, addMultipleCardMultiple)}, r.GetUsers())
}

// 随机庄家
func (r *Room) randomBanker() {
	i := common.Random(0, rule.PCount)
	if len(InitCard) > 0 {
		i = 0
	}
	banker := r.GetUsers()[i]
	banker.setRoleType(ROLETYPE_BANKER)
	r.setControllerUser(banker)
	r.setBanker(banker)
	r.setBankerFirstHandle(true)
}

// 检测开赛
func (r *Room) checkStart() {
	//开赛
	r.start()
}

// 开赛
func (r *Room) start() {
	//重置所有玩家的排名
	r.resetUsersRanking()
	//设置比赛开始
	r.SetRoomState(frame.ROOMSTATE_MATCH)
	//触发玩家操作
	r.triggerUserHandle()
}

// 重置玩家排名
func (r *Room) resetUsersRanking() {
	for _, user := range r.getUsers() {
		user.ranking = -1
	}
}

// Clone 完整复制数据
func (r *Room) clone(a, b interface{}) error {
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	dec := gob.NewDecoder(buff)
	if err := enc.Encode(a); err != nil {
		return err
	}
	if err := dec.Decode(b); err != nil {
		return err
	}
	return nil
}

func (r *Room) Test_generateCardsWithRandom() {
	r.mapDeck = map[int]int{}
	r.initDeck()
	userA := &User{}
	r.users = append(r.users, userA)
	userB := &User{}
	r.users = append(r.users, userB)
	r.setBanker(userA)
	userA.setRoleType(ROLETYPE_BANKER)
	r.generateCardsWithRandom()
}

// 随机生成牌
func (r *Room) generateCardsWithRandom() []CardList {
	//初始化所有玩家默认的牌列表
	cardLists := make([]CardList, rule.PCount)
	for i, user := range r.GetUsers() {
		//庄家多一张
		if user.getRoleType() == ROLETYPE_BANKER {
			cardLists[i] = CardList(make(CardList, rule.PerCapitaCount+1))
		} else {
			cardLists[i] = CardList(make(CardList, rule.PerCapitaCount))
		}
	}
	roomDeck := r.getDeck() //整副牌
	removeIndexs := []int{} //要删除的牌序列
	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	indexs := rr.Perm(r.deckSize) //打乱的牌序列
	//自定义牌列表
	indexsArr := [][]int{}
	//分配牌
	if len(InitCard) == 0 {
		//自定义手牌
		indexsArr = r.customHandCards(indexs)
		// indexsArr = [][]int{[]int{0, 1, 2, 3, 4, 5}, []int{6, 7, 8, 9, 10, 11}}
	} else {
		users := r.GetUsers()
		users[0].setRoleType(ROLETYPE_BANKER)
		r.setControllerUser(users[0])
		r.setBanker(users[0])
		mapIndex := map[int]bool{}
		for _, v := range InitCard {
			indexs = []int{}
			if v == "" {
				l := len(roomDeck)
				for j := 0; j < rule.PerCapitaCount; j++ {
					for {
						i := common.Random(0, l)
						if mapIndex[i] == false {
							mapIndex[i] = true
							indexs = append(indexs, i)
							break
						}
					}
				}
			} else {
				carids := common.StrArrToIntArr(strings.Split(v, "|"))
				for _, cardid := range carids {
					for i := 0; i < len(roomDeck); i++ {
						if roomDeck[i].ID == cardid {
							if mapIndex[i] == false {
								mapIndex[i] = true
								indexs = append(indexs, i)
								break
							}
						}
					}
				}
			}
			indexsArr = append(indexsArr, indexs)
		}
	}
	// fmt.Println("=====", indexsArr)
	m := map[int]bool{}
	for _, indexs := range indexsArr {
		for _, index := range indexs {
			m[index] = true
		}
	}
	// fmt.Println("预分配的牌==========:", m)
	index := 0
	x := 0
	for i := 0; i < rule.PCount; i++ {
		customIndexs := []int{}
		if len(indexsArr) > i {
			customIndexs = indexsArr[i]
		}
		for j := 0; j < len(cardLists[i]); j++ {
			if len(customIndexs) > 0 && j < len(customIndexs) {
				x = customIndexs[j]
			} else {
				for {
					x = indexs[index]
					index = index + 1
					if m[x] == false {
						// fmt.Println("未使用:", index, x)
						break
					}
					// fmt.Println("重寻找:", index, x)
				}
			}
			//发给玩家
			card := roomDeck[x].clone()
			cardLists[i][j] = card
			//更新map牌池
			r.updateMapDeck(card.ID)
			//记录要移除的牌序号
			removeIndexs = append(removeIndexs, x)
			if j == rule.PerCapitaCount {
				r.setHandleLastCard(card)
			}
		}
	}

	//测试
	// handCount := 0
	// xxx := map[int]int{}
	// for _, card := range cardLists[0] {
	// 	xxx[card.ID]++
	// 	handCount++
	// }
	// for _, card := range cardLists[1] {
	// 	xxx[card.ID]++
	// 	handCount++
	// }
	// fmt.Println("手牌池:", handCount, xxx)
	// for cardid, count := range xxx {
	// 	if count > 4 {
	// 		fmt.Println("	手牌异常:", cardid, count)
	// 	}
	// }

	//庄家系统发过一次牌
	r.getBanker().AddSendCardCount()
	r.addSendCardCount()
	//更新牌池
	sort.Sort(sort.Reverse(sort.IntSlice(removeIndexs)))
	for _, index := range removeIndexs {
		roomDeck = append(roomDeck[:index], roomDeck[index+1:]...)
	}
	r.setDeck(roomDeck)

	// deckCount := 0
	// for _, card := range r.getDeck() {
	// 	xxx[card.ID]++
	// 	deckCount++
	// }
	// fmt.Println("总牌池:", handCount+deckCount, xxx)
	// for cardid, count := range xxx {
	// 	if count > 4 {
	// 		fmt.Println("	总牌池异常:", cardid, count)
	// 		logger.Fatalf("异常")
	// 	}
	// }

	return cardLists
}

// 自定义手牌
func (r *Room) customHandCards(indexs []int) [][]int {
	// for _, user := range r.GetUsers() {
	// 	user.addDisplayArea("11|11|11")
	// 	user.addDisplayArea("12|12|12")
	// 	user.addDisplayArea("13|13|13")
	// }
	indexsArr := [][]int{}
	usedIndexs := map[int]bool{}
	test := [][]int{}
	for k := 0; k < rule.PCount; k++ {
		customIndexs := []int{}
		test2 := []int{}
		rd := common.Random(1, 11)
		// fmt.Println("================随机数:", rd)
		// rd = 4
		if rd <= 5 {
			roomDeck := r.getDeck()
			if rd <= 3 {
				cardType := rd
				//随机生成n张万饼条
				for i := 0; i < 5; i++ {
					for _, j := range indexs {
						if roomDeck[j].Type == cardType && usedIndexs[j] == false {
							test2 = append(test2, roomDeck[j].ID)
							customIndexs = append(customIndexs, j)
							usedIndexs[j] = true
							break
						}
					}
				}
			} else if rd == 4 {
				// times := 0
				//随机生成n个对
				for {
					if len(customIndexs) >= 6 {
						for _, index := range customIndexs {
							test2 = append(test2, roomDeck[index].ID)
						}
						break
					}
					tmpIndexs := []int{}
					rd := common.Random(0, len(indexs))
					// if k == 0 {
					// 	times++
					// 	if times == 1 {
					// 		rd = 29
					// 	} else if times == 2 {
					// 		rd = 29
					// 	}
					// }
					index := indexs[rd]
					// fmt.Println("kkkkkkk:", index, usedIndexs)
					if usedIndexs[index] == false {
						rdCard := roomDeck[index]
						if rdCard.Type == CardType_Mix || rdCard.Type == CardType_Flower {
							continue
						}
						cardid := rdCard.ID
						for i, card := range roomDeck {
							if card.ID == cardid {
								if usedIndexs[i] == false {
									tmpIndexs = append(tmpIndexs, i)
									if len(tmpIndexs) >= 2 {
										break
									}
								}
							}
						}
					}
					if len(tmpIndexs) < 2 {
						tmpIndexs = []int{}
					} else {
						for _, index := range tmpIndexs {
							usedIndexs[index] = true
						}
						customIndexs = append(customIndexs, tmpIndexs...)
					}
				}
			} else if rd == 5 {
				// times := 0
				//随机生成n个刻
				for {
					if len(customIndexs) >= 6 {
						for _, index := range customIndexs {
							test2 = append(test2, roomDeck[index].ID)
						}
						break
					}
					tmpIndexs := []int{}
					rd := common.Random(0, len(indexs))
					// if k == 0 {
					// 	times++
					// 	if times == 1 {
					// 		rd = 29
					// 	} else if times == 2 {
					// 		rd = 29
					// 	}
					// }
					index := indexs[rd]
					// fmt.Println("kkkkkkk:", index, usedIndexs)
					if usedIndexs[index] == false {
						rdCard := roomDeck[index]
						if rdCard.Type == CardType_Mix || rdCard.Type == CardType_Flower {
							continue
						}
						cardid := rdCard.ID
						for i, card := range roomDeck {
							if card.ID == cardid {
								if usedIndexs[i] == false {
									tmpIndexs = append(tmpIndexs, i)
									if len(tmpIndexs) >= 3 {
										break
									}
								}
							}
						}
					}
					if len(tmpIndexs) < 3 {
						tmpIndexs = []int{}
					} else {
						for _, index := range tmpIndexs {
							usedIndexs[index] = true
						}
						customIndexs = append(customIndexs, tmpIndexs...)
					}
				}
			}
		}
		indexsArr = append(indexsArr, customIndexs)
		test = append(test, test2)
	}
	// fmt.Println("========xxxxxxxx:", test, indexsArr)
	return indexsArr
}

// 生成所有人的牌
func (r *Room) generateCards() []CardList {
	//初始化所有玩家默认的牌列表
	cardLists := make([]CardList, rule.PCount)
	//生成玩家的牌
	if r.GetCardMode() == CARDMODE_RANDOM {
		cardLists = r.generateCardsWithRandom()
	}
	return cardLists
}

// 处理生成的牌
func (r *Room) handleGenerateCards(cardLists []CardList) []CardList {
	//	userStrs := make([]string, pcount)
	//	for i := 0; i < pcount; i++ {
	//		if i%2 == 0 {
	//			userStrs[i] = "2-1-1-12|2-1-1-12|2-1-1-12"
	//		} else {
	//			userStrs[i] = "10-9-1-7|11-10-1-8|12-11-1-9|13-12-1-10|14-13-1-11"
	//		}
	//	}
	//	for i := 0; i < pcount; i++ {
	//		if userStrs[i] == "" {
	//			cardLists[i] = cardLists[i][:cardCount[i]]
	//			continue
	//		}
	//		arr := strings.Split(userStrs[i], "|")
	//		cards := CardList{}
	//		for i, cardInfo := range arr {
	//			cardInfo = cardInfo
	//			arr2 := strings.Split(cardInfo, "-")
	//			id_, value, suit, priority, role := arr2[0], arr2[1], arr2[2], arr2[3], arr2[4]
	//			id, _ := strconv.Atoi(id_)
	//			v, _ := strconv.Atoi(value)
	//			s, _ := strconv.Atoi(suit)
	//			p, _ := strconv.Atoi(priority)
	//			r, _ := strconv.Atoi(role)
	//			cards = append(cards, Card{id, v, s, p, i, r})
	//		}
	//		cardLists[i] = cards
	//	}
	return cardLists
}

// 比赛结束(测试用的)
func (r *Room) matchEnd() bool {
	r.checkMatchingOver()
	return true
}
