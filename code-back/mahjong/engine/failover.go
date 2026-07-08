/*
容错处理
因为前台会传入一些引擎中的方法（切后台或者刷新游戏）
*/

package engine

import (
	"context"

	"combine.com/utils/types"
)

func (this *Engine) ExitRoom(ctx context.Context, args types.KVS) (reply types.KVS) {
	return
}
