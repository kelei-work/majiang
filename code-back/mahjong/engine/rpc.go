package engine

import (
	"combine.com/utils/frame"
	"combine.com/utils/types"
)

//推送匹配信息
func matchingPush(roomid string, memberid int) {
	kvs := types.KVS{}
	kvs.Set("func", "MatchingPush")
	kvs.Set(frame.ROOMID, roomid)
	kvs.Set(frame.MEMBERID, memberid)
	frame.RpcCall(frame.GetRpcxClient(frame.RPC_BUILD), kvs)
}

//开赛-结算
func matchBegin(roomid string) types.KVS {
	kvs := types.KVS{}
	kvs.Set("func", "MatchBegin")
	kvs.Set(frame.ROOMID, roomid)
	reply := frame.RpcCall(frame.GetRpcxClient(frame.RPC_SETTLE), kvs)
	return reply
}

//完赛-结算
func matchEnd(roomid string) types.KVS {
	kvs := types.KVS{}
	kvs.Set("func", "MatchEnd")
	kvs.Set(frame.ROOMID, roomid)
	reply := frame.RpcCall(frame.GetRpcxClient(frame.RPC_SETTLE), kvs)
	return reply
}

func MatchEndBuild(kvs types.KVS) {
	matchEndBuild(kvs)
}

//比赛结束调用build服务
func matchEndBuild(kvs types.KVS) {
	kvs.Set("func", "MatchEnd")
	frame.RpcCall(frame.GetRpcxClient(frame.RPC_BUILD), kvs)
}

// 获取用户数据
func getUserInfo(memberid int) types.KVS {
	kvs := types.KVS{}
	kvs.Set("func", "GetUserInfo")
	kvs.Set(frame.MEMBERID, memberid)
	reply := frame.RpcCall(frame.GetRpcxClient(frame.RPC_BLL), kvs)
	return reply
}
