/*
比赛构建-好友同玩
*/

package engine

import (
	"context"
	"fmt"

	"combine.com/utils/types"
)

/*
获取好友同玩积分
in:{target:目标memberid}
out:memberid,积分
*/
func (this *Engine) GetHYTWIntegral(ctx context.Context, args types.KVS) (reply types.KVS) {
	memberid := args.GetInt("target")
	user := GetUser(args)
	integral := 0
	if user != nil {
		integral = user.getRoundIntegral()
	}
	reply = types.Json(fmt.Sprintf("%d,%d", memberid, integral))
	return
}
