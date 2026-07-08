package engine

import (
	"context"
	"fmt"
	"net"
	"sync"

	"combine.com/utils/common"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
	"combine.com/utils/types"
)

var (
	PLATFORM    = 3
	ctx         = context.Background()
	mapUserConn sync.Map
)

type Engine struct {
}

func New() *Engine {
	logger.Infof("[启动引擎]")
	engine_init()
	return &Engine{}
}

/*
	服务关闭
*/
func ShutDown() {
	//通知网关服务器关闭
	closeCount := 0
	mapUserConn.Range(func(key interface{}, value interface{}) bool {
		memberid := key.(int)
		conn := value.(net.Conn)
		data := frame.NewPushData(memberid, frame.SHUTDOWN)
		rpcChange := frame.RpcChange{Name: frame.RPC_DOUDIZHU}
		data.Set(frame.RPC_CHANGE, rpcChange)
		frame.SendMsgToRpcClientByConn(conn, data.Byte())
		closeCount++
		return true
	})
	logger.Infof("关闭rpc连接数:%d", closeCount)
	//清空所有房间
	RoomManage.ClearALLRoom()
}

//获取玩家rpc连接
func getConn(memberid int) net.Conn {
	conn, ok := mapUserConn.Load(memberid)
	if !ok {
		return nil
	}
	return conn.(net.Conn)
}

/*
	获取服务地址
*/
func (this *Engine) GetServerAddr(ctx context.Context, args types.KVS) (reply types.KVS) {
	reply = types.KVS{"msg": *(frame.GetArgs().RpcxServer.Addr)}
	return
}

/*
	连接
*/
func (this *Engine) Connect(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetMemberID()
	conn := frame.GetRpcRemoteConn(ctx)
	mapUserConn.Store(memberid, conn)
	if user := UserManage.GetUser(memberid); user != nil {
		user.SetConn(conn)
	}
	return
}

/*
	连接关闭
*/
func (this *Engine) ConnectClose(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetMemberID()
	mapUserConn.Delete(memberid)
	user := UserManage.GetUser(memberid)
	if user != nil {
		user.close()
	}
	return
}

/*
断线重连获取比赛信息
in:
out:
push:各种比赛数据推送
*/
func (this *Engine) Reconnect(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := UserManage.GetUser(args.GetMemberID())
	if user == nil {
		return types.Fail(common.CODE_ROOM_INEXIST)
	}
	if user.GetRoom() == nil {
		return types.Fail(common.CODE_ROOM_INEXIST)
	}
	user.Reconnect()
	return types.DONE
}

/*
接收日志
*/
func (this *Engine) ReceLog(ctx context.Context, args types.KVS) {
	if !args.IsNil(frame.MEMBERID) {
		user := UserManage.GetUser(args.GetMemberID())
		if user == nil {
			return
		}
		room := user.GetRoom()
		if room == nil {
			return
		}
		room.Log(fmt.Sprintf("%v", args))
	}
}
