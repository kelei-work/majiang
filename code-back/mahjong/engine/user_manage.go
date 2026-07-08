/*
玩家管理
*/

package engine

import (
	"net"
	"sync"

	"combine.com/utils/delay"
	"combine.com/utils/types"
)

type userManage struct {
	judgmentUser *User
	users        map[int]*User //玩家列表
	Lock         sync.RWMutex
}

func userManage_init() *userManage {
	UserManage := userManage{}
	UserManage.users = map[int]*User{}
	return &UserManage
}

var (
	UserManage = *userManage_init()
)

//获取玩家
func (u *userManage) GetAllUsers() map[int]*User {
	return u.users
}

//获取玩家
func GetUser(args types.KVS) *User {
	return UserManage.GetUser(args.GetMemberID())
}

//获取玩家
func (u *userManage) GetUser(memberid int) *User {
	u.Lock.Lock()
	defer u.Lock.Unlock()
	return u.users[memberid]
}

//获取玩家根据Memberid
func (u *userManage) GetUserByMemberid(memberid int) *User {
	u.Lock.Lock()
	defer u.Lock.Unlock()
	for _, user := range u.users {
		if user.getMemberid() == memberid {
			return user
		}
	}
	return nil
}

//获取玩家数量
func (u *userManage) GetUserCount() int {
	u.Lock.Lock()
	defer u.Lock.Unlock()
	return len(u.users)
}

//创建一个玩家
func (u *userManage) createUser() *User {
	user := User{}
	user.setOnline(true)
	return &user
}

//添加玩家
func (u *userManage) AddUser(memberid int, conn net.Conn) *User {
	u.Lock.Lock()
	defer u.Lock.Unlock()
	user := u.users[memberid]
	if user == nil {
		user = u.createUser()
		user.memberid = memberid
		user.ranking = -1
		user.conn = conn
		user.online = true
		user.cards = []*Card{}
		user.multipleCardType = map[int]int{}
		user.playTingSolidifyCards = map[string]CardList{}
		user.pengInfo = map[int]*User{}
		user.taskSystem = delay.NewTaskSystem()
		u.users[memberid] = user
	} else {
		user.conn = conn
	}
	return user
}

//删除一个玩家（从比赛玩家列表中）
func (u *userManage) RemoveUser(user *User) {
	memberid := user.getMemberid()
	u.Lock.Lock()
	delete(u.users, memberid)
	u.Lock.Unlock()
}

//强制删除一个玩家
func (u *userManage) ForceDeleteUser(memberid int) {
	user := u.GetUser(memberid)
	if user != nil {
		UserManage.RemoveUser(user)
	}
}
