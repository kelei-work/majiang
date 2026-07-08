/*
玩家
*/

package engine

import (
	"bytes"
	"fmt"
	"net"
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

//玩家当前状态
const (
	UserStatus_NoPass = iota //没过牌
	UserStatus_Pass          //过牌
)

const (
	PlayType_Normal  = iota //正常出牌
	PlayType_EndGame        //重新进游戏，残局下，当前轮的出牌信息
)

const (
	ROLETYPE_PLAYER = iota //闲家
	ROLETYPE_BANKER        //庄家
)

const (
	TingStatus_NO       = iota //没听牌
	TingStatus_TING            //听牌
	TingStatus_TIANTING        //天听
)

const (
	MatchResult_Win  = 1  //胜
	MatchResult_Flat = 0  //平
	MatchResult_Lose = -1 //负
)

type User struct {
	memberid              int                 //平台id
	conn                  net.Conn            //链接
	room                  *Room               //房间
	status                int                 //玩家状态
	cards                 []*Card             //牌列表
	index                 int                 //座位编号
	autoTimes             int                 //倒计时结束自动操作的次数
	trusteeship           bool                //是否托管
	roundIntegral         int                 //每轮积分，每轮清零
	online                bool                //玩家是否在线
	chatLastTime          time.Time           //发言的最后时间,用来做冷却
	taskSystem            *delay.TaskSystem   //延迟任务系统
	ranking               int                 //排名(从0开始)
	lockHandle            sync.Mutex          //玩家操作的锁
	sendMsgLock           sync.Mutex          //发送消息的锁
	RoleType              int                 //角色类型
	giveUp                bool                //放弃操作
	isTingCard            bool                //是否听牌
	tingGroupsInfo        string              //所有可以听牌的信息
	tingGroupInfo         []int               //选中一组听牌的信息
	displayArea           string              //陈列区(多个组合)
	discardArea           string              //丢弃区
	repairFlowerArea      string              //补花区
	chiCount              int                 //吃牌次数
	pengCount             int                 //碰牌次数
	mingGangCount         int                 //明杠数量
	anGangCount           int                 //暗杠数量
	sendCardCount         int                 //发牌次数
	checkTing             bool                //检测过听
	tingStatus            int                 //听牌状态
	danDiaoJiang          bool                //单调将
	multipleAddCard       int                 //加番牌
	multipleRepairFlower  int                 //补花番
	multipleCardType      map[int]int         //牌型番
	multipleBase          int                 //基本番
	multipleAwardFlower   int                 //奖花番
	playCardID            int                 //为听牌打出的牌
	playTingSolidifyCards map[string]CardList //打牌听牌实体化牌面
	maxMultipleCards      CardList            //最大胡牌番牌面
	pengInfo              map[int]*User       //碰牌的信息
}

func (u *User) GetUserImage() string {
	buff := bytes.Buffer{}
	room := u.GetRoom()
	if room == nil {
		buff.WriteString(fmt.Sprintf("是否在房间中:不在房间中\n"))
		buff.WriteString(fmt.Sprintf("玩家状态:%d\n", u.getStatus()))
		buff.WriteString(fmt.Sprintf("玩家在线:%v\n", u.getOnline()))
	} else {
		buff.WriteString(fmt.Sprintf("是否在房间中:在房间中\n"))
		buff.WriteString(fmt.Sprintf("玩家状态:%d\n", u.getStatus()))
		buff.WriteString(fmt.Sprintf("玩家在线:%v\n", u.getOnline()))
		buff.WriteString(fmt.Sprintf("玩家所在房间状态:%d\n", room.GetRoomState()))
	}
	return buff.String()
}

//重置玩家
func (u *User) reset() {
	u.autoTimes = 0
	u.trusteeship = false
	u.giveUp = false
	u.isTingCard = false
	u.tingGroupsInfo = ""
	u.tingGroupInfo = []int{}
	u.displayArea = ""
	u.discardArea = ""
	u.repairFlowerArea = ""
	u.chiCount = 0
	u.pengCount = 0
	u.mingGangCount = 0
	u.anGangCount = 0
	u.sendCardCount = 0
	u.checkTing = false
	u.tingStatus = TingStatus_NO
	u.danDiaoJiang = false
	u.multipleAddCard = 0
	u.multipleAwardFlower = 0
	u.multipleBase = 0
	u.multipleCardType = map[int]int{}
	u.multipleRepairFlower = 0
	u.setStatus(UserStatus_NoPass)
	u.pengInfo = map[int]*User{}
	u.stop()
	u.setRoleType(ROLETYPE_PLAYER)
	u.playTingSolidifyCards = map[string]CardList{}
}

//获取redis中的key
func (u *User) getKey() string {
	return fmt.Sprintf("user:%d", u.getMemberid())
}

//获取redis中的key
func (u *User) getMemberKey() string {
	return fmt.Sprintf("user:%d", u.getMemberid())
}

//克隆
func (u *User) clone() *User {
	memberid := u.getMemberid()
	conn := u.GetConn()
	user := UserManage.AddUser(memberid, conn)
	return user
}

//获取玩家是否有操作权限
func (u *User) getHandlePerm() bool {
	return u.currCtlIsSelf()
}

//当前牌权是不是自己
func (u *User) currCtlIsSelf() bool {
	room := u.GetRoom()
	//房间当前牌权的玩家
	if room.getControllerUser() == u {
		return true
	}
	return false
}

//获取全部牌的ID列表
func (u *User) getCardsID(separator string) *string {
	return u.getPartCardsID(u.getCards(), separator)
}

//获取全部牌的ID列表
func (u *User) getCardsIDArray() []int {
	arr := []int{}
	for _, card := range u.getCards() {
		arr = append(arr, card.ID)
	}
	return arr
}

//获取部分牌的ID列表
func (u *User) getPartCardsID(cards []*Card, separator string) *string {
	buff := bytes.Buffer{}
	for _, card := range cards {
		buff.WriteString(fmt.Sprintf("%d%s", card.ID, separator))
	}
	cardsid := common.RemoveLastChar(buff)
	return cardsid
}

//是否过牌
func (u *User) isPass() bool {
	return u.getStatus() == UserStatus_Pass
}

//往玩家手里添加牌
func (u *User) addCards(cards []*Card) {
	userCards := CardList{}
	userCards = u.getCards()
	for _, card := range cards {
		userCards = append(userCards, card)
	}
	sort.Sort(userCards)
	u.setCards(userCards)
}

//删除玩家手里的牌
func (u *User) updateUserCards(playIndex int) {
	defer func() {
		if p := recover(); p != nil {
			logger.Fatalf("[recovery] updateUserCards err:%v,%d", p, playIndex)
		}
	}()
	userCards := u.getCards()
	userCards = append(userCards[:playIndex], userCards[playIndex+1:]...)
	u.setCards(userCards)
}

//玩家手牌排序
func (u *User) orderCards() {
	cards := u.getCards()
	//排序
	sort.Sort(cards)
	//修改索引
	for i := 0; i < len(cards); i++ {
		cards[i].Index = i
	}
	//设置玩家牌面
	u.setCards(cards)
	// u.push("Opening_Push", u.getCardsID("|"))
}

func (u *User) GetConn() net.Conn {
	return u.conn
}

func (u *User) SetConn(conn net.Conn) {
	u.conn = conn
}

func (u *User) GetOnline() bool {
	return u.online
}

func (u *User) getOnline() bool {
	return u.online
}

func (u *User) setOnline(online bool) {
	u.online = online
}

func (u *User) getChatLastTime() time.Time {
	return u.chatLastTime
}

func (u *User) setChatLastTime() {
	u.chatLastTime = time.Now()
}

func (u *User) getRoundIntegral() int {
	return u.roundIntegral
}

func (u *User) setRoundIntegral(roundIntegral int) {
	u.roundIntegral = roundIntegral
}

func (u *User) GetMemberid() int {
	return u.memberid
}

func (u *User) getMemberid() int {
	return u.memberid
}

func (u *User) setMemberid(memberid int) {
	u.memberid = memberid
}

func (u *User) SetRoom(room *Room) {
	u.room = room
}

func (u *User) setRoom(room *Room) {
	u.room = room
}

func (u *User) GetRoom() *Room {
	return u.room
}

func (u *User) getIndex() int {
	return u.index
}

func (u *User) setIndex(index int) {
	u.index = index
}

func (u *User) getStatus() int {
	return u.status
}

func (u *User) setStatus(status int) {
	u.status = status
}

func (u *User) getCards() CardList {
	return u.cards
}

func (u *User) SetCards(cards CardList) {
	u.cards = cards
}

func (u *User) setCards(cards CardList) {
	u.cards = cards
}

func (u *User) getRanking() int {
	return u.ranking
}

func (u *User) setRanking(ranking int) {
	u.ranking = ranking
}

func (u *User) getRoleType() int {
	return u.RoleType
}

func (u *User) setRoleType(RoleType int) {
	u.RoleType = RoleType
}

func (u *User) getGiveUp() bool {
	return u.giveUp
}

func (u *User) setGiveUp(giveUp bool) {
	u.giveUp = giveUp
}

func (u *User) getIsTingCard() bool {
	return u.isTingCard
}

func (u *User) setIsTingCard(isTingCard bool) {
	u.isTingCard = isTingCard
}

func (u *User) getTingGroupsInfo() string {
	return u.tingGroupsInfo
}

func (u *User) setTingGroupsInfo(tingGroupsInfo string) {
	u.tingGroupsInfo = tingGroupsInfo
}

func (u *User) getTingGroupInfo() string {
	realCards := []*Card{}
	realCards = append(realCards, u.getCards()...)
	tingCardList := TingCardList{}
	room := u.GetRoom()
	for _, cardid := range u.tingGroupInfo {
		newCards := u.playTingSolidifyCards[fmt.Sprintf("%d-%d", u.playCardID, cardid)]
		u.setCards(newCards)
		newcardsid := u.getUserAllCardsID()
		sort.Ints(newcardsid)
		// fmt.Println("xxxxxxxxxx:", newcardsid)
		tingCard := &TingCard{
			CardID:       cardid,
			SurplusCount: u.getCardSurplusCount(cardid),
			Multiple:     room.getHuCardMultiple(u, newcardsid),
		}
		tingCardList = append(tingCardList, tingCard)
		// cardInfo = append(cardInfo, fmt.Sprintf("%d|%d|%d", cardid, u.getCardSurplusCount(cardid), room.getHuCardMultiple(u, newcardsid)))
	}
	sort.Sort(tingCardList)
	u.setCards(realCards)
	arr := []string{}
	for _, tingCard := range tingCardList {
		arr = append(arr, fmt.Sprintf("%d|%d|%d", tingCard.CardID, tingCard.SurplusCount, tingCard.Multiple))
	}
	res := strings.Join(arr, "$")
	return res
}

func (u *User) setTingGroupInfo(tingGroupInfo []int) {
	u.tingGroupInfo = tingGroupInfo
}

func (u *User) getRepairFlowerArea() string {
	return u.repairFlowerArea
}

func (u *User) addRepairFlowerArea(cardid int) {
	if u.repairFlowerArea == "" {
		u.repairFlowerArea = strconv.Itoa(cardid)
	} else {
		u.repairFlowerArea = fmt.Sprintf("%s|%d", u.repairFlowerArea, cardid)
	}
}

func (u *User) getDiscardArea() string {
	return u.discardArea
}

func (u *User) addDiscardArea(cardid int) {
	if u.discardArea == "" {
		u.discardArea = strconv.Itoa(cardid)
	} else {
		u.discardArea = fmt.Sprintf("%s|%d", u.discardArea, cardid)
	}
}

func (u *User) removeDiscardAreaLastCard() {
	index := strings.LastIndex(u.discardArea, "|")
	if index >= 0 {
		u.discardArea = u.discardArea[0:index]
	}
}

func (u *User) getDisplayArea() string {
	return u.displayArea
}

func (u *User) SetDisplayArea(groupInfo string) {
	u.displayArea = groupInfo
}

func (u *User) addDisplayArea(groupInfo string) {
	if u.displayArea == "" {
		u.displayArea = groupInfo
	} else {
		u.displayArea = fmt.Sprintf("%s$%s", u.displayArea, groupInfo)
	}
}

func (u *User) getDisplayCardIDs() []int {
	cardids := []int{}
	if u.displayArea != "" {
		groups := strings.Split(u.displayArea, "$")
		for _, group := range groups {
			arr := strings.Split(group, "|")
			for _, cardid := range arr {
				cardids = append(cardids, common.ParseInt(cardid))
			}
		}
	}
	return cardids
}

func (u *User) getChiCount() int {
	return u.chiCount
}

func (u *User) AddChiCount() {
	u.chiCount++
}

func (u *User) addChiCount() {
	u.chiCount++
}

func (u *User) getPengCount() int {
	return u.pengCount
}

func (u *User) addPengCount() {
	u.pengCount++
}

func (u *User) getMingGangCount() int {
	return u.mingGangCount
}

func (u *User) setMingGangCount(mingGangCount int) {
	u.mingGangCount = mingGangCount
}

func (u *User) AddMingGangCount() {
	u.mingGangCount += 1
}

func (u *User) addMingGangCount() {
	u.mingGangCount += 1
}

func (u *User) getAnGangCount() int {
	return u.anGangCount
}

func (u *User) setAnGangCount(anGangCount int) {
	u.anGangCount = anGangCount
}

func (u *User) AddAnGangCount() {
	u.anGangCount += 1
}

func (u *User) addAnGangCount() {
	u.anGangCount += 1
}

func (u *User) getGangCount() int {
	return u.getAnGangCount() + u.getMingGangCount()
}

func (u *User) getSendCardCount() int {
	return u.sendCardCount
}

func (u *User) AddSendCardCount() {
	u.sendCardCount += 1
}

func (u *User) addSendCardCount() {
	u.sendCardCount += 1
}

func (u *User) getCheckTing() bool {
	return u.checkTing
}

func (u *User) setCheckTing(checkTing bool) {
	u.checkTing = checkTing
}

func (u *User) getTingStatus() int {
	return u.tingStatus
}

func (u *User) setTingStatus(tingStatus int) {
	u.tingStatus = tingStatus
}

func (u *User) getDanDiaoJiang() bool {
	return u.danDiaoJiang
}

func (u *User) setDanDiaoJiang(danDiaoJiang bool) {
	u.danDiaoJiang = danDiaoJiang
}

//获取托管状态
func (u *User) getTrusteeship() bool {
	return u.trusteeship
}

func (u *User) getMultipleAddCard() int {
	return u.multipleAddCard
}

func (u *User) setMultipleAddCard(multipleAddCard int) {
	u.multipleAddCard = multipleAddCard
}

func (u *User) addMultipleAddCard(value int) {
	u.multipleAddCard += value
}

func (u *User) minusMultipleAddCard(value int) {
	u.multipleAddCard -= value
}

func (u *User) getMultipleRepairFlower() int {
	return u.multipleRepairFlower
}

func (u *User) setMultipleRepairFlower(multipleRepairFlower int) {
	u.multipleRepairFlower = multipleRepairFlower
}

func (u *User) addMultipleRepairFlower(value int) {
	u.multipleRepairFlower += value
}

func (u *User) getMultipleBase() int {
	return u.multipleBase
}

func (u *User) setMultipleBase(multipleBase int) {
	u.multipleBase = multipleBase
}

func (u *User) addMultipleBase(value int) {
	u.multipleBase += value
}

func (u *User) getMultipleCardType() map[int]int {
	return u.multipleCardType
}

func (u *User) getSumMultipleWithCardType() (multiple int) {
	for cardType, count := range u.getMultipleCardType() {
		if count > 0 {
			multiple += CardTypeGroup_Multiples[cardType] * count
		}
	}
	return multiple
}

func (u *User) setMultipleCardType(multipleCardType map[int]int) {
	u.multipleCardType = multipleCardType
}

func (u *User) addMultipleCardType(multipleCardType int, count int) {
	u.multipleCardType[multipleCardType] += count
}

func (u *User) getMultipleAwardFlower() int {
	return u.multipleAwardFlower
}

func (u *User) setMultipleAwardFlower(multipleAwardFlower int) {
	u.multipleAwardFlower = multipleAwardFlower
}

func (u *User) addMultipleAwardFlower(value int) {
	u.multipleAwardFlower += value
}

//是否是自己
func (u *User) isSelf(user *User) bool {
	return u.GetMemberid() == user.GetMemberid()
}

//给一个玩家推送残局
func (u *User) pushEndGame() {
	logger.Debugf("推送残局")
	room := u.GetRoom()
	//推送房间的状态信息
	room.matchingPush(u)
	time.Sleep(time.Millisecond * 5)
	//推送比赛的信息
	u.pushMatchInfo()
	//玩家回来,设置为在线
	u.setOnline(true)
}

//推送比赛的信息
func (u *User) pushMatchInfo() {
	room := u.GetRoom()
	//推送此人剩余的牌
	u.pushSurplusCards()
	time.Sleep(time.Millisecond * 10)
	//推送当前轮的出牌信息
	u.pushCyclePlayCardInfo()
	time.Sleep(time.Millisecond * 5)
	//推送托管状态
	u.TG_Push()
	//推送当前出牌状态
	ctlMsg := room.getSetCtlMsg()
	if len(ctlMsg) > 0 {
		u.setController(ctlMsg)
	}
}

//获取下手玩家
func (u *User) getNextUser() *User {
	index := u.getIndex()
	index += 1
	if index >= rule.PCount {
		index = 0
	}
	return u.GetRoom().getUsers()[index]
}

//断线重连获取比赛信息
func (u *User) Reconnect() {
	u.setOnline(true)
	//推送房间匹配信息
	u.pushRoomMatchingInfo()
	//推送比赛的信息
	u.pushMatchInfo()
}

// 推送房间匹配信息
func (u *User) pushRoomMatchingInfo() {
	room := u.GetRoom()
	matchingPush(room.GetRoomID(), u.GetMemberid())
}

//推送此人剩余的牌
func (u *User) pushSurplusCards() {
	message := u.getCardsID("|")
	if *message != "" {
		u.push("Opening_Push", *message)
	}
}

//推送当前轮的出牌信息
func (u *User) pushCyclePlayCardInfo() {
	room := u.GetRoom()
	if room.getCurrentCard() != nil {
		message := fmt.Sprintf("%d,%d,%d", room.getControllerUser().GetMemberid(), room.getCurrentCard().ID, PlayType_EndGame)
		time.Sleep(time.Millisecond * 5)
		u.push("Play", message)
	}
}

//暂停倒计时
func (u *User) pause() {
	u.taskSystem.PauseAllTask()
}

//恢复倒计时
func (u *User) resume() {
	u.taskSystem.ResumeAllTask()
}

//关闭倒计时
func (u *User) stop() {
	u.taskSystem.StopAllTask()
}

//开启出牌倒计时
func (u *User) countDown_playCard(waitTime int) {
	if u.getTrusteeship() {
		waitTime = 1
	}
	t, _ := time.ParseDuration(strconv.Itoa(waitTime) + "s")
	if frame.GetMode() == frame.MODE_TOPSPEED {
		t = time.Millisecond * 50
	}
	logger.Debugf("开启出牌倒计时:%d", waitTime)
	key := "playcard"
	task := &delay.Task{
		Key:      key,
		TimeMode: delay.TIMEMODE_SUR,
		Exec: func() {
			u.timeEnd(waitTime)
		},
		SurplusTime: t,
	}
	u.taskSystem.AddTask(task)
	u.taskSystem.StartTask(key)
}

//关闭出牌倒计时
func (u *User) close_countDown_playCard() {
	u.taskSystem.StopTask("playcard")
}

//出牌倒计时结束
func (u *User) timeEnd(waitTime int) {
	room := u.GetRoom()
	currentCard := room.getCurrentCard()
	currentCard = currentCard
	if u.getTrusteeship() { //托管
		u.trusteeshipPlayCard()
	} else if u.getIsTingCard() { //听牌
		u.trusteeshipPlayCard()
	} else {
		//超时
		if waitTime >= room.playWaitTime {
			//托管处理
			u.trusteeshipHandle()
		}
	}
}

//开启操作倒计时
func (u *User) countDown_handle(waitTime int, handleType int, content string) {
	if u.getTrusteeship() {
		waitTime = 1
	}
	t, _ := time.ParseDuration(strconv.Itoa(waitTime) + "s")
	// if ACCELERATE {
	// 	t = time.Millisecond * 100
	// }
	logger.Debugf("开启操作倒计时:%d", waitTime)
	key := "handle"
	task := &delay.Task{
		Key:      key,
		TimeMode: delay.TIMEMODE_SUR,
		Exec: func() {
			//托管推送
			// if !u.trusteeship {
			// 	u.trusteeship = true
			// 	message := "1"
			// 	u.push("TG_Push", &message)
			// }
			HandleWithUser(u, handleType, content)
		},
		SurplusTime: t,
	}
	u.taskSystem.AddTask(task)
	u.taskSystem.StartTask(key)
}

//关闭操作倒计时
func (u *User) close_countDown_handle() {
	u.taskSystem.StopTask("handle")
}

//托管出牌
func (u *User) trusteeshipPlayCard() {
	PlayCardWithUser(u, len(u.getCards())-1)
}

//托管处理
func (u *User) trusteeshipHandle() {
	u.autoTimes += 1
	//倒计时结束自动操作的次数>=1次,进行托管
	if u.autoTimes >= 1 {
		u.setTrusteeship(true)
	}
}

//设置玩家托管
func (u *User) setTrusteeship(status bool) {
	if u.trusteeship == status {
		u.TG_Push()
		return
	}
	u.trusteeship = status
	if !u.trusteeship {
		u.close_countDown_playCard()
		if u.GetRoom().getControllerUser() == u {
			u.countDown_playCard(u.GetRoom().playWaitTime)
		}
	}
	u.autoTimes = 0
	u.TG_Push()
}

/*
托管推送
out:托管状态(0不托1托)
*/
func (u *User) TG_Push() {
	status := u.trusteeship
	//托管推送
	funcName := "TG_Push"
	message := "0"
	//托管
	if status {
		message = "1"
		room := u.GetRoom()
		//如果玩家是（当前控牌人 或 等待烧牌状态）,关闭倒计时,玩家立即出牌
		if room.getControllerUser() == u {
			u.close_countDown_playCard()
			//比赛结束前的玩家剩最后一套牌的时候，逃跑了，托管出牌（会导致ExitMatch方法的死锁），所以开启一个新线程执行避免死锁！！！！
			go func() {
				defer func() {
					if p := recover(); p != nil {
						logger.Warnf("[recovery] TG_Push err:%v", p)
					}
				}()
				u.timeEnd(0)
			}()
		}
	}
	u.push(funcName, message)
}

func (u *User) setController(message string) {
	u.GetRoom().pushMessageToUsers("SetController_Push", []string{message}, []*User{u})
}

func (u *User) setControllerUsers(message string) {
	u.GetRoom().pushMessageToUsers("SetController_Push", []string{message}, u.GetRoom().getUsers())
}

func (u *User) setCtlUsers(userid string, userstatus string, info string, setControllerStatus int) {
	message := fmt.Sprintf("%s|%s|%s,,%d,", userid, userstatus, info, setControllerStatus)
	u.setControllerUsers(message)
}

//给此玩家推送信息
func (u *User) push(funcName string, message interface{}) {
	if !u.getOnline() {
		return
	}
	u.sendMsgLock.Lock()
	defer u.sendMsgLock.Unlock()
	conn := u.GetConn()
	if conn != nil {
		kvs := frame.NewPushData(u.GetMemberid(), funcName, message)
		if u.GetRoom() != nil {
			u.GetRoom().Log(fmt.Sprintf("推送数据:{%v}", kvs))
		}
		kvs.Set(frame.TIMESTAMP, common.GetTimestamp(common.TIMESTAMP_MICRO))
		frame.SendMsgToRpcClientByConn(conn, kvs.Byte())
	}
}

func (u *User) getUserMatchKey() string {
	return frame.GetUserMatchKey(u.getMemberid())
}

//玩家关闭连接
func (u *User) close() {
	//开赛后,玩家托管,并设置玩家离线
	u.setOnline(false)
}

//是观战玩家
func (u *User) isIdleUser() bool {
	idleUsers := u.GetRoom().getIdleUsers()
	for _, idleUser := range idleUsers {
		if u == idleUser {
			return true
		}
	}
	return false
}
