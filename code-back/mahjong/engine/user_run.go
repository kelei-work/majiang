/*
玩家-操作-明牌
*/

package engine

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"combine.com/utils/common"
)

//基本番
func (u *User) multiple_base(cardsID []int) {
	u.setMultipleBase(4)
	// u.setMultipleBase(0)
	// for _, cardid := range cardsID {
	// 	//只算万饼条
	// 	if isWBT(cardid) {
	// 		u.addMultipleBase(cardid % 10)
	// 	} else if isFJ(cardid) {
	// 		u.addMultipleBase(arrowWindCardMultiple)
	// 	}
	// }
}

//牌型番
func (u *User) Test_multiple_cardType(cardsID []int) {
	u.multiple_cardType(cardsID)
}

//牌型番
func (u *User) multiple_cardType(cardsID []int) {
	u.setMultipleCardType(map[int]int{})
	room := u.GetRoom()
	//牌总数量
	cardCount := len(cardsID)
	//获取各类型牌的数量
	myriadCardCount, cakeCardCount, stripCardCount, windCardCount, arrowCardCount := u.getCardTypeCount(cardsID)
	windCardCount, arrowCardCount = windCardCount, arrowCardCount
	if u.getPPH(cardsID) { //碰碰胡
		u.addMultipleCardType(CardTypeGroup_PPH, 1)
	}
	if u.getHYS(myriadCardCount, cakeCardCount, stripCardCount, windCardCount, arrowCardCount) { //混一色
		u.addMultipleCardType(CardTypeGroup_HYS, 1)
	} else if cardCount == myriadCardCount || cardCount == cakeCardCount || cardCount == stripCardCount { //清一色
		u.addMultipleCardType(CardTypeGroup_QYS, 1)
	}
	if room.getGangSendCard() { //杠上开花
		u.addMultipleCardType(CardTypeGroup_GSKH, 1)
	}
	tingStatus := u.getTingStatus()
	if tingStatus > TingStatus_NO {
		if tingStatus == TingStatus_TING { //听牌
			u.addMultipleCardType(CardTypeGroup_ST, 1)
		} else if tingStatus == TingStatus_TIANTING {
			u.addMultipleCardType(CardTypeGroup_TT, 1) //天听
		}
	}
	huStatus := getHuType(u)
	if huStatus > HuStatus_NORMAL {
		if huStatus == HuStatus_TIANHU {
			u.addMultipleCardType(CardTypeGroup_TH, 1) //天胡
		} else if huStatus == HuStatus_DIHU {
			u.addMultipleCardType(CardTypeGroup_DH, 1) //地胡
		}
	}
	if u.getSSBK(cardsID) { //十三不靠
		u.addMultipleCardType(CardTypeGroup_SSBK, 1)
	}
	if u.getQD(cardsID) { //七对
		u.addMultipleCardType(CardTypeGroup_QD, 1)
	}
	if u.getQL(cardsID) { //青龙
		u.addMultipleCardType(CardTypeGroup_QL, 1)
	}
	if u.getHDLY() { //海底捞月
		u.addMultipleCardType(CardTypeGroup_HDLY, 1)
	}
	if u.getDDJ() { //单调将
		u.addMultipleCardType(CardTypeGroup_DDJ, 1)
	}
	if u.fourMix() {
		u.addMultipleCardType(CardTypeGroup_SH, 1)
	}
	if PRINT_CRUX_LOG {
		buff := bytes.Buffer{}
		buff.WriteString("<番数类型>")
		for multipleCardType, count := range u.getMultipleCardType() {
			if count > 0 {
				buff.WriteString(fmt.Sprintf("%s:%d   ", CardTypeGroup_Names[multipleCardType], count))
			}
		}
		fmt.Println(buff.String())
	}
}

//四混
func (u *User) fourMix() bool {
	mixCount := 0
	for _, card := range u.getCards() {
		if card.IsMix == 1 {
			mixCount++
		}
	}
	if mixCount >= 4 {
		return true
	}
	return false
}

//单调将
func (u *User) getDDJ() bool {
	room := u.GetRoom()
	handleLastCard := room.getHandleLastCard()
	if handleLastCard == nil {
		return false
	}
	jiangCard := room.getJiangCard()
	if jiangCard == nil {
		return false
	}
	if handleLastCard.ID == jiangCard.ID {
		return true
	}
	return false
	// cards := u.getCards()
	// newCards := CardList{}
	// newCards = append(newCards, cards...)
	// room.orderCards(newCards)
	// fmt.Println("aaaa:", room.huCheck(newCards))
	// for _, card := range newCards {
	// 	fmt.Println("1111:", card.ID)
	// }
	// removeCount := 0
	// for i := len(newCards) - 1; i >= 0; i-- {
	// 	if newCards[i].ID == handleLastCard.ID {
	// 		removeCount++
	// 		newCards = append(newCards[:i], newCards[i+1:]...)
	// 	}
	// 	if removeCount >= 2 {
	// 		break
	// 	}
	// }
	// for _, card := range newCards {
	// 	fmt.Println("2222:", card.ID)
	// }
	// fmt.Println("bbbb:", room.huCheck(newCards))
	return false
}

//缺一门
func (u *User) getQYM(cardsID []int) bool {
	cardMap := map[int]bool{}
	for _, cardid := range cardsID {
		if isWBT(cardid) {
			cardMap[cardid/10] = true
		}
	}
	if len(cardMap) == 2 {
		return true
	}
	return false
}

//双暗刻
func (u *User) getSAK() bool {
	cardMap := map[int]int{}
	for _, cardid := range u.getCardsIDArray() {
		cardMap[cardid]++
	}
	akCount := 0
	for _, count := range cardMap {
		if count == 3 {
			akCount++
		}
	}
	if akCount >= 2 {
		return true
	}
	return false
}

//双同刻
func (u *User) getSTK(cardsID []int) bool {
	cardMap := map[int]int{}
	for _, cardid := range cardsID {
		if isWBT(cardid) {
			cardMap[cardid]++
		}
	}
	m := map[int]int{}
	for cardid, count := range cardMap {
		if count == 3 {
			m[cardid%10]++
		}
	}
	for _, count := range m {
		if count >= 2 {
			return true
		}
	}
	return false
}

//门前清
func (u *User) getMQQ() bool {
	huStatus := getHuType(u)
	if huStatus == HuStatus_RENHU {
		return true
	}
	room := u.GetRoom()
	if u.getChiCount()+u.getPengCount()+u.getMingGangCount() == 0 && room.getMatchResult() == MatchResult_DianPao {
		return true
	}
	return false
}

//双明杠
func (u *User) getSMG() bool {
	if u.getMingGangCount() >= 2 {
		return true
	}
	return false
}

//不求人
func (u *User) getBQR() bool {
	huStatus := getHuType(u)
	if huStatus == HuStatus_TIANHU || huStatus == HuStatus_DIHU {
		return true
	}
	room := u.GetRoom()
	if u.getChiCount()+u.getPengCount()+u.getMingGangCount() == 0 && room.getMatchResult() == MatchResult_ZiMo {
		return true
	}
	return false
}

//全求人
func (u *User) getQQR() bool {
	if len(u.getCards()) == 2 {
		if u.GetRoom().getMatchResult() == MatchResult_DianPao {
			return true
		}
	}
	return false
}

//获取各类型牌的数量
func (u *User) getCardTypeCount(cardsID []int) (myriadCardCount, cakeCardCount, stripCardCount, windCardCount, arrowCardCount int) {
	for _, cardid := range cardsID {
		cardType := cardid / 10
		if cardType == CardType_Myriad {
			myriadCardCount++
		} else if cardType == CardType_Cake {
			cakeCardCount++
		} else if cardType == CardType_Strip {
			stripCardCount++
		} else if cardType == CardType_Wind {
			windCardCount++
		} else if cardType == CardType_Arrow {
			arrowCardCount++
		}
	}
	return
}

//混一色
func (u *User) getHYS(myriadCardCount, cakeCardCount, stripCardCount, windCardCount, arrowCardCount int) bool {
	//mapCardType := map[int]bool{}
	cardTypeCount := 0
	if myriadCardCount > 0 {
		cardTypeCount++
	}
	if cakeCardCount > 0 {
		cardTypeCount++
	}
	if stripCardCount > 0 {
		cardTypeCount++
	}
	if cardTypeCount == 1 && (windCardCount+arrowCardCount == 2) {
		return true
	}
	return false
}

//碰碰胡
func (u *User) getPPH(cardsID []int) bool {
	//陈列区检测
	displayArea := u.getDisplayArea()
	if displayArea != "" {
		validCount := 0
		groups := strings.Split(displayArea, "$")
		for _, info := range groups {
			arr := strings.Split(info, "#")
			if len(arr) > 1 {
				validCount++
			} else {
				arr2 := strings.Split(info, "|")
				if arr2[0] == arr2[1] && arr2[1] == arr2[2] {
					validCount++
				}
			}
		}
		if validCount != len(groups) {
			return false
		}
	}
	//手牌检测
	cardMap := map[int]int{}
	for _, cardid := range cardsID {
		cardMap[cardid]++
	}
	//小于三张的数量
	lessThreeCount := 0
	for _, count := range cardMap {
		if count < 3 {
			lessThreeCount++
		}
	}
	if lessThreeCount > 1 {
		return false
	}
	return true

}

var (
// groups = [][]int{
// 	[]int{11, 14, 17, 22, 25, 28, 33, 36, 39},
// 	[]int{11, 14, 17, 32, 35, 38, 23, 26, 29},
// 	[]int{21, 24, 27, 12, 15, 18, 33, 36, 39},
// 	[]int{21, 24, 27, 32, 35, 38, 13, 16, 19},
// 	[]int{31, 34, 37, 12, 15, 18, 23, 26, 29},
// 	[]int{31, 34, 37, 22, 25, 28, 13, 16, 19},
// }
)

var SSBKGroups []map[int]bool = []map[int]bool{
	map[int]bool{11: true, 14: true, 17: true, 22: true, 25: true, 28: true, 33: true, 36: true, 39: true},
	map[int]bool{11: true, 14: true, 17: true, 32: true, 35: true, 38: true, 23: true, 26: true, 29: true},
	map[int]bool{21: true, 24: true, 27: true, 12: true, 15: true, 18: true, 33: true, 36: true, 39: true},
	map[int]bool{21: true, 24: true, 27: true, 32: true, 35: true, 38: true, 13: true, 16: true, 19: true},
	map[int]bool{31: true, 34: true, 37: true, 12: true, 15: true, 18: true, 23: true, 26: true, 29: true},
	map[int]bool{31: true, 34: true, 37: true, 22: true, 25: true, 28: true, 13: true, 16: true, 19: true},
}

func newWordGroup() map[int]bool {
	return map[int]bool{41: true, 42: true, 43: true, 44: true, 52: true, 53: true}
}

func (u *User) getMixCount() {

}

//十三不靠
func (u *User) getSSBK(cardids []int) bool {
	if u.getDisplayArea() != "" {
		return false
	}
	mapFJ := map[int]bool{}
	otherCards := []int{}
	for _, cardid := range cardids {
		if isFJ(cardid) {
			mapFJ[cardid] = true
		} else {
			otherCards = append(otherCards, cardid)
		}
	}
	fjCount := len(mapFJ)
	if fjCount != 5 {
		return false
	}
	mapWBT := map[int]bool{}
	if len(otherCards) > 0 {
		sort.Ints(otherCards)
		for _, group := range SSBKGroups {
			if len(mapWBT) > 0 {
				break
			}
			for _, cardid := range otherCards {
				if group[cardid] {
					mapWBT[cardid] = true
				} else {
					mapWBT = map[int]bool{}
					break
				}
			}
		}
	}
	if len(mapWBT) != 9 {
		return false
	}
	return true
}

//七对
func (u *User) getQD(cardids []int) bool {
	if u.getDisplayArea() != "" {
		return false
	}
	mapCards := map[int]int{}
	for _, cardid := range cardids {
		mapCards[cardid]++
	}
	count := 0
	for _, c := range mapCards {
		if c == 2 {
			count++
		}
	}
	if count >= 7 {
		return true
	}
	return false
}

//青龙
func (u *User) getQL(cardsID []int) bool {
	mapCards := map[int]bool{}
	cardid := cardsID[0]
	mapCards[cardid] = true
	for i := 1; i < len(cardsID); i++ {
		nextCardID := cardsID[i]
		if nextCardID == cardid+1 {
			mapCards[nextCardID] = true
			if len(mapCards) >= 9 {
				return true
			}
			cardid = nextCardID
		} else if nextCardID-cardid > 1 {
			cardid = nextCardID
			mapCards = map[int]bool{}
			mapCards[cardid] = true
		}
	}
	return false
}

//海底捞月
func (u *User) getHDLY() bool {
	room := u.GetRoom()
	surplusCardCount := len(room.getDeck())
	if surplusCardCount == 0 && room.getMatchResult() == MatchResult_ZiMo {
		return true
	}
	return false
}

//加番牌
func (u *User) multiple_addCard(cardsID []int) {
	return
	u.setMultipleAddCard(0)
	multipleCardID := u.GetRoom().getMultipleCardID()
	// fmt.Println("加番牌是：", multipleCardID)
	for _, cardid := range cardsID {
		if cardid == multipleCardID {
			u.addMultipleAddCard(addMultipleCardMultiple)
			// fmt.Println("++++++")
		}
	}
	for _, card := range u.getCards() {
		if card.ID == multipleCardID && card.IsMix == 1 {
			u.minusMultipleAddCard(addMultipleCardMultiple)
			// fmt.Println("------")
		}
	}
}

//奖花
func (u *User) multiple_awardFlower(cardsID []int) {
	room := u.GetRoom()
	mapCardsID := map[int]bool{}
	for {
		if len(mapCardsID) >= 5 {
			break
		}
		index := common.Random(0, room.deckSize)
		card := room.getDeck()[index]
		if card.Type != CardType_Flower {
			mapCardsID[card.ID] = true
		}
	}
	awardFlowers := []string{}
	for awardCardID, _ := range mapCardsID {
		multiple := 0
		for _, userCardID := range cardsID {
			if awardCardID == userCardID {
				multiple += awardFlowerCardMultiple
				u.addMultipleAwardFlower(awardFlowerCardMultiple)
			}
		}
		awardFlowers = append(awardFlowers, fmt.Sprintf("%d|%d", awardCardID, multiple))
	}
	awardFlowerInfo := strings.Join(awardFlowers, "$")
	// fmt.Println("奖花信息：", awardFlowerInfo)
	/*
		奖花推送
		push:AwardFlower_Push,cardid|番数$cardid|番数$...
	*/
	room.pushMessageToUsers("AwardFlower_Push", []string{awardFlowerInfo}, room.GetUsers())
}

//玩家是否可以打牌
func (u *User) userCanPlayCard() (canPlayCard int) {
	room := u.GetRoom()
	isTianTingStatus := len(room.getTianTingStatusUsers()) > 0
	if isTianTingStatus { //天听操作
		canPlayCard = 0
	} else if u.getIsTingCard() { //玩家已听牌
		canPlayCard = 0
	} else {
		if room.getCurrentCard() == nil { //系统发牌
			canPlayCard = 1
		}
	}
	return canPlayCard
}

//获取玩家所有牌列表(包括陈列区)
func (u *User) getUserAllCardsID() []int {
	allCardsID := []int{}
	for _, card := range u.getCards() {
		allCardsID = append(allCardsID, card.ID)
	}
	displayArea := u.getDisplayArea()
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
	// fmt.Println("玩家所有牌：", allCardsID)
	return allCardsID
}

//获取玩家最大番数牌列表(包括陈列区)
func (u *User) getUserMaxMultipleCardsID() []int {
	allCardsID := []int{}
	for _, card := range u.maxMultipleCards {
		allCardsID = append(allCardsID, card.ID)
	}
	displayArea := u.getDisplayArea()
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
	// fmt.Println("玩家所有牌：", allCardsID)
	return allCardsID
}

//获取牌剩余数量
func (u *User) getCardSurplusCount(cardid int) (count int) {
	room := u.GetRoom()
	//听牌后,看到真正剩余牌的数量
	count = room.getMapDeck()[cardid]
	if !u.getIsTingCard() { //没听牌,需要加上别人手中看不见的(手牌、暗杠的牌)
		for _, user := range room.GetUsers() {
			if u == user {
				continue
			}
			for _, card := range user.getCards() {
				if card.ID == cardid {
					count++
				}
			}
			for _, info := range strings.Split(user.getDisplayArea(), "$") {
				arr := strings.Split(info, "#")
				if len(arr) > 1 {
					if common.ParseInt(arr[1]) == 2 { //暗杠
						if common.ParseInt(strings.Split(arr[0], "|")[0]) == cardid {
							count += 4
							break
						}
					}
				}
			}
		}
	}
	return
}

func (u *User) haveMixCard() bool {
	for _, card := range u.getCards() {
		if card.IsMix == 1 {
			return true
		}
	}
	return false
}
