/*
房间
*/

package engine

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"combine.com/utils/common"
	"combine.com/utils/delay"
	"combine.com/utils/logger"
)

const (
	CARDMODE_RANDOM = iota //随机
)

//比赛结束方式
const (
	MatchResult_LiuJu   = 0 //流局
	MatchResult_ZiMo    = 1 //自摸
	MatchResult_DianPao = 2 //点炮
)

const (
	MatchingStatus_Run   = iota //进行中
	MatchingStatus_Pause        //暂停
	MatchingStatus_Over         //结束
)

const (
	PlayWaitTime = 15 //行牌等待时间
)

const (
	HuStatus_NORMAL = iota //正常胡
	HuStatus_TIANHU        //天胡
	HuStatus_DIHU          //地胡
	HuStatus_RENHU         //人胡
)

type DianPaoInfo struct {
	User *User
	Card *Card
}

type Room struct {
	id                    string                //id
	matchid               int                   //比赛类型
	roomtype              int                   //房间类型
	createTime            string                //创建时间
	state                 int                   //房间状态
	users                 []*User               //玩家列表
	idleusers             []*User               //未落座玩家列表
	cuser                 *User                 //牌权的玩家
	currentCard           *Card                 //当前牌(待丢弃区)
	currentCardUser       *User                 //当前牌的玩家
	bankerFirstHandle     bool                  //庄家第一次操作
	playTimes             int                   //出牌的次数
	matchResult           int                   //比赛结果
	inning                int                   //当前局数
	innings               int                   //总局数
	setCtlMsg             string                //设置牌权的内容,推送残局的时候用
	baseScore             int                   //底分
	playWaitTime          int                   //行牌等待时间
	firstController       *User                 //第一个出牌的人
	cardMode              int                   //牌的模式(随机)
	canHandleUser         *User                 //当前可操作的玩家
	lockGetCard           sync.Mutex            //取牌锁
	lockTianTingHandle    sync.Mutex            //天听操作锁
	roomLogger            *log.Logger           //房间日志系统
	logFile               *os.File              //房间日志文件
	daemonThread          *delay.Task           //守护线程
	roomMaster            *User                 //组队玩法的房主
	dealTask              *delay.Task           //发牌任务
	banker                *User                 //庄家
	deck                  []Card                //牌池
	deckSize              int                   //牌数量
	mapDeck               map[int]int           //牌池(记录数量的)
	dianPaoUser           *User                 //点炮玩家
	huUser                *User                 //胡牌玩家
	multipleCardID        int                   //加番牌ID
	gangSendCard          bool                  //开杠后系统发牌
	gangBloom             bool                  //杠上开花
	huStatus              int                   //胡牌状态
	sendCardCount         int                   //发牌次数
	playCardCount         int                   //打牌次数
	tianTingStatusUsers   []*User               //在天听状态下的玩家列表
	bankerTingPlayIndex   int                   //庄家听牌的Index
	handleLastCard        *Card                 //操作的最后一张牌
	jiangCard             *Card                 //将牌
	canHandleUserInfoList CanHandleUserInfoList //当前出牌轮次可操作的玩家信息列表(吃碰杠听胡)
	dianPaoInfo           *DianPaoInfo          //点炮信息
	playType              int                   //玩法类型
	friendRoomCost        int                   //好友约局房费
}

//房间配置
func (r *Room) config() {
	r.playWaitTime = PlayWaitTime
}

func (r *Room) GetRoomID() string {
	return r.id
}

func (r *Room) SetRoomID(roomid string) {
	r.id = roomid
}

func (r *Room) GetCreateTime() string {
	return r.createTime
}

func (r *Room) SetCreateTime(createtime string) {
	r.createTime = createtime
}

//获取创建时长(分钟)
func (r *Room) GetCreateDuration() int {
	createTime, _ := common.ParseTime(r.createTime)
	return int(time.Now().Sub(createTime).Minutes())
}

func (r *Room) GetRoomImage() string {
	buff := bytes.Buffer{}
	buff.WriteString(fmt.Sprintf("房间状态:%d\n", r.GetRoomState()))
	buff.WriteString(fmt.Sprintf("玩家列表:%s\n", r.GetAllUsersInfo()))
	if r.getControllerUser() == nil {
		buff.WriteString("牌权的玩家:nil \n")
	} else {
		buff.WriteString(fmt.Sprintf("牌权的玩家:%d\n", r.getControllerUser().GetMemberid()))
	}
	buff.WriteString(fmt.Sprintf("当前牌:%v\n", r.getCurrentCard()))
	if r.getCurrentCardsUser() == nil {
		buff.WriteString("当前牌的玩家:nil\n")
	} else {
		buff.WriteString(fmt.Sprintf("当前牌的玩家:%d\n", r.getCurrentCardsUser().GetMemberid()))
	}
	buff.WriteString(fmt.Sprintf("当前轮出牌信息:%v\n", r.getCurrentCard()))
	buff.WriteString("所有玩家的状态:")
	for _, user := range r.GetUsers() {
		buff.WriteString(fmt.Sprintf("[%d:%d]", user.GetMemberid(), user.getStatus()))
	}
	return buff.String()
}

//重置
func (r *Room) reset() {
	r.huUser = nil
	r.gangSendCard = false
	r.gangBloom = false
	r.huStatus = HuStatus_NORMAL
	r.sendCardCount = 0
	r.playCardCount = 0
	r.setPlayTimes(0)
	r.setControllerUser(nil)
	r.setCurrentCard(nil)
	r.setCurrentCardUser(nil)
	r.setSetCtlMsg("")
	r.mapDeck = map[int]int{}
	r.handleLastCard = nil
	r.jiangCard = nil
	r.canHandleUserInfoList = nil
}

//获取组队玩法的房主
func (r *Room) getRoomMaster() *User {
	return r.roomMaster
}

//设置组队玩法的房主
func (r *Room) setRoomMaster(u *User) {
	r.roomMaster = u
}

//转换房主
func (r *Room) transitionRoomMaster() {
	users := r.getAllUsers()
	for _, user := range users {
		if user != nil {
			if user != r.getRoomMaster() {
				r.setRoomMaster(user)
				break
			}
		}
	}
}

//获取牌的模式
func (r *Room) GetCardMode() int {
	return r.cardMode
}

//设置牌的模式
func (r *Room) SetCardMode(cardMode int) {
	r.cardMode = cardMode
}

//获取庄家
func (r *Room) getBanker() *User {
	return r.banker
}

//设置庄家
func (r *Room) setBanker(banker *User) {
	r.banker = banker
}

//获取牌池
func (r *Room) getDeck() []Card {
	return r.deck
}

//设置牌池
func (r *Room) SetDeck(deck []Card) {
	r.deck = deck
}

//设置牌池
func (r *Room) setDeck(deck []Card) {
	r.deck = deck
	r.deckSize = len(deck)
}

//获取map牌池
func (r *Room) getMapDeck() map[int]int {
	return r.mapDeck
}

//更新map牌池
func (r *Room) updateMapDeck(cardid int) {
	r.mapDeck[cardid] = r.mapDeck[cardid] - 1
}

//获取点炮玩家
func (r *Room) getDianPaoUser() *User {
	return r.dianPaoUser
}

//设置点炮玩家
func (r *Room) setDianPaoUser(dianPaoUser *User) {
	r.dianPaoUser = dianPaoUser
}

//获取胡牌玩家
func (r *Room) getHuUser() *User {
	return r.huUser
}

//设置胡牌玩家
func (r *Room) setHuUser(huUser *User) {
	r.huUser = huUser
}

func (r *Room) getMultipleCardID() int {
	return r.multipleCardID
}

func (r *Room) setMultipleCardID(multipleCardID int) {
	r.multipleCardID = multipleCardID
}

func (r *Room) getGangSendCard() bool {
	return r.gangSendCard
}

func (r *Room) setGangSendCard(gangSendCard bool) {
	r.gangSendCard = gangSendCard
}

func (r *Room) getGangBloom() bool {
	return r.gangBloom
}

func (r *Room) SetGangBloom(gangBloom bool) {
	r.gangBloom = gangBloom
}

func (r *Room) setGangBloom(gangBloom bool) {
	r.gangBloom = gangBloom
}

func (r *Room) getHuStatus() int {
	return r.huStatus
}

func (r *Room) setHuStatus(huStatus int) {
	r.huStatus = huStatus
}

func (r *Room) getSendCardCount() int {
	return r.sendCardCount
}

func (r *Room) addSendCardCount() {
	r.sendCardCount += 1
}

func (r *Room) getPlayCardCount() int {
	return r.playCardCount
}

func (r *Room) addPlayCardCount() {
	r.playCardCount += 1
}

func (r *Room) getTianTingStatusUsers() []*User {
	return r.tianTingStatusUsers
}

func (r *Room) setTianTingStatusUsers(tianTingStatusUsers []*User) {
	r.tianTingStatusUsers = tianTingStatusUsers
}

func (r *Room) getBankerTingPlayIndex() int {
	return r.bankerTingPlayIndex
}

func (r *Room) setBankerTingPlayIndex(bankerTingPlayIndex int) {
	r.bankerTingPlayIndex = bankerTingPlayIndex
}

func (r *Room) getHandleLastCard() *Card {
	return r.handleLastCard
}

func (r *Room) SetHandleLastCard(handleLastCard *Card) {
	r.handleLastCard = handleLastCard
}

func (r *Room) setHandleLastCard(handleLastCard *Card) {
	r.handleLastCard = handleLastCard
}

func (r *Room) getJiangCard() *Card {
	return r.jiangCard
}

func (r *Room) SetJiangCard(jiangCard *Card) {
	r.jiangCard = jiangCard
}

func (r *Room) setJiangCard(jiangCard *Card) {
	r.jiangCard = jiangCard
}

//从牌池中取出一张牌
func (r *Room) getCardFromDeck() *Card {
	r.lockGetCard.Lock()
	defer r.lockGetCard.Unlock()
	//分配牌
	roomDeck := r.getDeck()
	if len(roomDeck) == 0 {
		return nil
	}
	index := common.Random(0, len(roomDeck))
	//测试
	if len(InitCard) > 0 {
		if r.getSendCardCount() == 1 {
			for i, card := range roomDeck {
				if card.ID == 15 {
					index = i
					break
				}
			}
		} else {
			// if index < len(roomDeck)-1 {
			// 	for {
			// 		count := 0
			// 		for _, card := range roomDeck {
			// 			if card.ID == 15 {
			// 				count++
			// 			}
			// 		}
			// 		if count == 1 {
			// 			index = common.Random(0, len(roomDeck))
			// 			if roomDeck[index].ID == 15 {
			// 				continue
			// 			} else {
			// 				break
			// 			}
			// 		}
			// 	}
			// }
		}
	}
	card := roomDeck[index].clone()
	roomDeck = append(roomDeck[:index], roomDeck[index+1:]...)
	r.setDeck(roomDeck)
	//更新map牌池
	r.updateMapDeck(card.ID)
	return card
}

//重开
func (r *Room) reStart() {
	r.resetUsers()
	r.reset()
}

//获取房间底分
func (r *Room) getBaseScore() int {
	return r.baseScore
}

//设置房间底分
func (r *Room) setBaseScore(baseScore int) {
	r.baseScore = baseScore
}

//更新出牌的次数
func (r *Room) updatePlayTimes() int {
	r.playTimes += 1
	return r.playTimes
}

//获取出牌的次数
func (r *Room) getPlayTimes() int {
	return r.playTimes
}

//设置出牌的次数
func (r *Room) setPlayTimes(playTimes int) {
	r.playTimes = playTimes
}

//获取设置牌权的命令
func (r *Room) getSetCtlMsg() string {
	return r.setCtlMsg
}

//设置牌权的内容,推送残局时候用
func (r *Room) setSetCtlMsg(setCtlMsg string) {
	r.setCtlMsg = setCtlMsg
}

//获取房间人数
func (r *Room) GetPCount() int {
	return len(r.users)
}

//获取房间观战人数
func (r *Room) GetIdlePCount() int {
	return len(r.idleusers)
}

//获取房间总人数(场上+观众席)
func (r *Room) GetTotalPCount() int {
	return r.GetPCount() + r.GetIdlePCount()
}

//获取所有玩家
func (r *Room) GetUsers() []*User {
	return r.users
}

//获取(UserID+IdleUserID)玩家集合
func (r *Room) getAllUsers() []*User {
	users := r.getUsers()
	idleUsers := r.getIdleUsers()
	allUsers := append(users, idleUsers...)
	return allUsers
}

/*
获取(UserID+IdleUserID)字符串集合
in:是否刷新
*/
func (r *Room) GetAllUsersInfo() string {
	users := r.getUsers()
	idleUsers := r.getIdleUsers()
	allUsers := append(users, idleUsers...)
	buff := bytes.Buffer{}
	buff.WriteString("{\n")
	for _, user := range allUsers {
		if user != nil {
			buff.WriteString(fmt.Sprintf("          userid:%d,是否在线:%v\n", user.GetMemberid(), user.getOnline()))
		}
	}
	buff.WriteString("     }")
	return buff.String()
}

//获取比赛类型
func (r *Room) GetMatchID() int {
	return r.matchid
}

//设置比赛类型
func (r *Room) setMatchID(matchID int) {
	r.matchid = matchID
}

//获取玩法类型
func (r *Room) GetPlayType() int {
	return r.playType
}

//设置玩法类型
func (r *Room) setPlayType(playType int) {
	r.playType = playType
}

//获取好友约局房费
func (r *Room) GetFriendRoomCost() int {
	return r.friendRoomCost
}

//设置好友约局房费
func (r *Room) setFriendRoomCost(roomCost int) {
	r.friendRoomCost = roomCost
}

//获取总轮次
func (r *Room) getInnings() int {
	return r.innings
}

//设置当前轮次
func (r *Room) setInnings(innings int) {
	r.innings = innings
}

//获取当前轮次
func (r *Room) getInning() int {
	return r.inning
}

//设置当前轮次
func (r *Room) setInning(inning int) {
	r.inning = inning
}

func (r *Room) getMatchResult() int {
	return r.matchResult
}

func (r *Room) SetMatchResult(matchResult int) {
	r.matchResult = matchResult
}

func (r *Room) setMatchResult(matchResult int) {
	r.matchResult = matchResult
}

//获取房间类型
func (r *Room) GetRoomType() int {
	return r.roomtype
}

//设置房间类型
func (r *Room) setRoomType(roomType int) {
	r.roomtype = roomType
}

//获取牌权玩家
func (r *Room) getControllerUser() *User {
	return r.cuser
}

//设置牌权玩家
func (r *Room) setControllerUser(user *User) {
	r.cuser = user
}

//获取当前牌
func (r *Room) getCurrentCard() *Card {
	return r.currentCard
}

//设置当前牌
func (r *Room) setCurrentCard(card *Card) {
	r.currentCard = card
}

//获取当前牌的玩家
func (r *Room) getCurrentCardsUser() *User {
	return r.currentCardUser
}

//设置当前牌的玩家
func (r *Room) setCurrentCardUser(user *User) {
	r.currentCardUser = user
}

//获取庄家第一次操作
func (r *Room) getBankerFirstHandle() bool {
	return r.bankerFirstHandle
}

//设置庄家第一次操作
func (r *Room) setBankerFirstHandle(b bool) {
	r.bankerFirstHandle = b
}

//获取房间状态
func (r *Room) GetRoomState() int {
	return r.state
}

//设置房间状态
func (r *Room) SetRoomState(state int) {
	r.state = state
}

//获取落座的所有玩家
func (r *Room) getUsers() []*User {
	return r.users
}

//获取未落座的所有玩家
func (r *Room) getIdleUsers() []*User {
	return r.idleusers
}

/*
重置房间中所有的玩家
*/
func (r *Room) resetUsers() {
	users := r.getUsers()
	for _, user := range users {
		if user != nil {
			user.reset()
		}
	}
}

//关闭房间
func (r *Room) close() {
	r.CloseLogFile()
	r.daemonThread.Stop()
	RoomManage.RemoveRoom(r)
}

//设置所有人托管状态
func (r *Room) SetAllUsersTrusteeshipStatus(status bool) {
	for _, user := range r.getUsers() {
		if user != nil {
			user.trusteeship = status
		}
	}
}

//给玩家列表推送信息(比赛中的推送)
func (r *Room) pushMessageToUsers(funcName string, messages []string, users []*User) {
	msgCount := len(messages)
	//消息的数量是多个,但是数量与要推送的人数不符
	if msgCount > 1 && len(users) != msgCount {
		logger.Errorf("pushMessageToUsers : %s", "消息的数量是多个,但是数量与要推送的人数不符")
		return
	}
	for i, user := range users {
		if user != nil {
			message := ""
			if msgCount <= 1 {
				message = messages[0]
			} else {
				message = messages[i]
			}
			user.push(funcName, message)
		}
	}
}
