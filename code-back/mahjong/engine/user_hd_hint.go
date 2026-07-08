/*
玩家-操作-提示
*/

package engine

import (
	"context"
	"fmt"

	"combine.com/utils/types"

	"combine.com/utils/logger"
)

func init() {
	//	cards := []Card{Card{Priority: 8}, Card{Priority: 9}, Card{Priority: 10}}
	//	u := &User{}
	//	cardType, _ := u.getCardType(cards)
	//	fmt.Println(1111, cardType)
}

/*
Hint(提示)
result:
	有能压过的牌:index|index|index
	没有:-2
*/
func (this *Engine) Hint(ctx context.Context, args types.KVS) (reply types.KVS) {
	user := GetUser(args)
	res := fmt.Sprintf("%d", HintWithUser(user))
	reply = types.Json(res)
	return
}

func HintWithUser(user *User) int {
	defer func() {
		if p := recover(); p != nil {
			logger.Warnf("HintWithUser:%v", p)
		}
	}()
	index := 0 //压不过
	return index
}
