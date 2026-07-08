/*
	保皇引擎
*/
package main

import (
	"context"
	"flag"
	"fmt"

	"combine.com/mahjong/cmds"
	"combine.com/mahjong/engine"
	"combine.com/utils/frame"
	"combine.com/utils/logger"
	"combine.com/utils/mysql"
	"combine.com/utils/redis"
)

var (
	ctx      = context.Background()
	gameName = frame.GetServerName(engine.PLATFORM)
)

var (
	addr           = flag.String("addr", "127.0.0.1:11010", "服务地址")
	basePath       = flag.String("base", "/rpcx", "rpcx前缀")
	etcdAddr       = flag.String("etcdAddr", "127.0.0.1:2879", "etcd地址")
	etcdAddrBll    = flag.String("etcdAddrBll", "127.0.0.1:2041", "bllEtcd地址")
	etcdAddrSettle = flag.String("etcdAddrSettle", "127.0.0.1:2031", "结算etcd地址")
	etcdAddrBuild  = flag.String("etcdAddrBuild", "127.0.0.1:2011", "组建etcd地址")
	memberDB       = flag.String("memberDB", "member,root,DWLT28102810,127.0.0.1:3306,gouji_nmmember", "")
	memberRedis    = flag.String("memberRedis", "127.0.0.1:9400", "")
	buildRedis     = flag.String("buildRedis", "127.0.0.1:9500", "")
	logLevel       = flag.Int("logLevel", logger.DebugLevel, "日志等级")
)

func main() {
	defer func() {
		if p := recover(); p != nil {
			//退出信号
			exitSignal()
			//退出
			logger.Fatalf(fmt.Sprintf("doudizhu crash<%s>:%v", *addr, p))
		}
	}()
	//解析参数
	flag.Parse()
	//启动服务
	args := frame.Args{}
	args.ServerName = gameName
	args.Commands = cmds.GetCmds()
	//日志等级
	args.LogLevel = *logLevel
	//性能
	args.PProf = &frame.PProf{Port: 6060}
	//gate的服务端
	args.RpcxServer = &frame.RpcxServer{frame.Discovery_Etcd, frame.Rpcx{Addr: addr, EtcdAddr: etcdAddr, BasePath: basePath, Username: "root", Password: "dwlt2810"}, []interface{}{new(frame.Rpcs)}}
	//rpc客户端
	args.MapRpcxClient = map[string]*frame.RpcxClient{
		frame.RPC_SETTLE: &frame.RpcxClient{frame.Discovery_Etcd, frame.Rpcx{EtcdAddr: etcdAddrSettle, BasePath: basePath, Username: "root", Password: "dwlt2810"}, "Rpcs", frame.UNIDIRECTIONAL},
		frame.RPC_BUILD:  &frame.RpcxClient{frame.Discovery_Etcd, frame.Rpcx{EtcdAddr: etcdAddrBuild, BasePath: basePath, Username: "root", Password: "dwlt2810"}, "Rpcs", frame.BIDIRECTIONAL},
		frame.RPC_BLL:    &frame.RpcxClient{frame.Discovery_Etcd, frame.Rpcx{EtcdAddr: etcdAddrBll, BasePath: basePath, Username: "root", Password: "dwlt2810"}, "Rpcs", frame.UNIDIRECTIONAL},
	}
	//redis
	redisDSNs := []*redis.DSN{}
	redisDSNs = append(redisDSNs, &redis.DSN{"member", *memberRedis, "]rds#dwlt#2209["})
	redisDSNs = append(redisDSNs, &redis.DSN{"build", *buildRedis, "]rds#dwlt#2209["})
	args.Redis = &frame.Redis{redisDSNs}
	//mysql
	sqlDSNs := []*mysql.DSN{}
	sqlDSNs = append(sqlDSNs, mysql.AnalysisFlag2DSN(memberDB))
	args.Sql = &frame.Sql{sqlDSNs}
	//退出信号
	args.ExitSignal = exitSignal
	//框架启动完毕后执行的方法
	args.Loaded = start
	//通过参数启动框架
	frame.Load(args)
}

func start() {
	//生成引擎
	engine := engine.New()
	//将引擎注入到各模块
	inject(engine)
	// test()
}

func test() {
	// intArr := []int{
	// 	20, 20, 20, 50, 50, 50, 50, 20, 100, 100,
	// 	200, 200, 200, 300, 300, 400, 400, 400, 500, 500,
	// 	500, 500, 700, 1000, 1000,
	// }
	// strArr := common.IntArrToStrArr(intArr)
	// fmt.Println(strings.Join(strArr, "|"))
	user := &engine.User{}
	//设置玩家牌
	cards := engine.CardList{}
	// cardsid := []int{11, 11, 11, 12, 12, 12, 13, 13, 13, 15, 16, 17, 15, 15}
	// cardsid := []int{11, 21, 41, 41, 42, 42, 43, 43, 43, 51, 51, 52, 52, 53}
	// cardsid := []int{11, 11, 11, 21, 21, 21, 13, 14, 15, 15, 16, 17, 15, 15}
	// cardsid := []int{11, 11, 12, 12, 13, 13, 21, 21, 22, 22, 23, 23, 41, 41}
	// cardsid := []int{41, 42, 43, 44, 51, 52, 53, 11, 13, 15, 17, 22, 24, 26}
	// cardsid := []int{11, 12, 13, 14, 15, 16, 17, 18, 19, 21, 22, 23, 33, 33}
	// cardsid := []int{11, 12, 13, 14, 15, 16, 17, 18, 19, 11, 12, 13, 52, 52}
	// cardsid := []int{12, 13, 14, 19, 19, 33, 33, 33, 34, 35, 36}
	// cardsid := []int{33, 34, 35, 35, 35, 36, 37, 37, 38, 38, 39}
	// cardsid := []int{11, 11, 23, 26, 26, 29, 29, 31, 34, 38, 39, 39, 53, 53}
	// cardsid := []int{33, 34, 35, 35, 35, 36, 37, 37, 38, 38, 39}
	// cardsid := []int{11, 11, 11, 22, 22, 22, 41, 41, 41, 51, 51}
	// cardsid := []int{}
	// cardsid := []int{41, 41, 42, 42, 43, 43, 44, 44, 44, 53, 53}
	// cardsid := []int{11, 11, 11, 12, 13, 13, 14, 15, 16, 17, 17, 18, 18, 19}
	// cardsid := []int{11, 11, 11, 21, 21, 21, 14, 15, 16, 17, 18, 19, 31, 31}
	// cardsid := []int{24, 24, 24, 25, 26, 27, 28, 28, 29, 29, 29, 51, 51, 51}
	// cardsid := []int{31, 34, 37, 22, 25, 28, 13, 16, 19, 41, 42, 43, 44, 51}
	// cardsid := []int{17, 18, 19, 21, 21, 22, 23, 24, 24, 25, 26, 27, 28, 29}
	// cardsid := []int{26, 26, 26, 27, 27, 27, 43, 43}
	cardsid := []int{71, 17, 18, 19, 32, 33, 34, 32}
	for _, cardid := range cardsid {
		cards = append(cards, engine.NewCard(cardid))
	}
	user.SetCards(cards)
	user.SetDisplayArea("16|16|16$15|15|15|15#1")
	// user.AddSendCardCount()
	// user.AddAnGangCount()
	// user.AddAnGangCount()
	// user.AddMingGangCount()
	// user.AddChiCount()
	room := &engine.Room{}
	room.SetGangBloom(true)
	// room.SetJiangCard(engine.NewCard(52))
	user.SetRoom(room)
	//设置牌池
	roomDeck := []engine.Card{}
	roomDeck = append(roomDeck, engine.GetDeck()...)
	room.SetDeck(roomDeck)
	room.SetMatchResult(engine.MatchResult_DianPao)
	//设置操作的最后一张牌
	// room.SetHandleLastCard(engine.NewCard(11))
	//测试
	b := room.HuCheck(cards)
	fmt.Println("是否可胡牌:", b)
	if b {
		user.Test_multiple_cardType(cardsid)
	}
	// go func() {
	// 	defer func() {
	// 		if p := recover(); p != nil {
	// 			logger.Errorf("[recovery] test err : %v", p)
	// 		}
	// 	}()
	// 	time.Sleep(time.Second * 1)
	// 	//		engine.Test_checkMatchingOver()
	// 	//		engine.PlayVideo([]string{""})
	// }()
}

func inject(engine *engine.Engine) {
	frame.Inject(engine)
	cmds.Inject(engine)
}

//退出信号
func exitSignal() {
	engine.ShutDown()
}
