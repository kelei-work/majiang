/*
牌类
*/

package engine

import (
	"combine.com/utils/frame"
)

var (
	//游戏规则
	rule = frame.NewRule(1, frame.GetPCount(PLATFORM), 13)
)

const (
	CardType_Myriad = 1 //万
	CardType_Cake   = 2 //饼
	CardType_Strip  = 3 //条
	CardType_Wind   = 4 //风
	CardType_Arrow  = 5 //箭
	CardType_Flower = 6 //花
	CardType_Mix    = 7 //混
)

const (
	DONG  = 41 //东
	NAN   = 42 //南
	XI    = 43 //西
	BEI   = 44 //北
	ZHONG = 51 //中
	FA    = 52 //发
	BAI   = 53 //白
	HUN   = 71 //混
)

type Multiple struct {
	Name     string
	Multiple int
}

type MultipleList []*Multiple

func (list MultipleList) Len() int {
	return len(list)
}

func (list MultipleList) Less(i, j int) bool {
	iV := list[i].Multiple
	jV := list[j].Multiple
	if iV > jV {
		return true
	} else {
		return false
	}
}

func (list MultipleList) Swap(i, j int) {
	var temp *Multiple = list[i]
	list[i] = list[j]
	list[j] = temp
}

//牌型
const (
	CardTypeGroup_DDJ   = iota //单调将
	CardTypeGroup_QYM          //缺一门
	CardTypeGroup_MG           //明杠
	CardTypeGroup_AG           //暗杠
	CardTypeGroup_SAK          //双暗刻
	CardTypeGroup_STK          //双同刻
	CardTypeGroup_MQQ          //门前清
	CardTypeGroup_ST           //听牌
	CardTypeGroup_BQR          //不求人
	CardTypeGroup_SMG          //双明杠
	CardTypeGroup_PPH          //碰碰胡
	CardTypeGroup_HYS          //混一色
	CardTypeGroup_QQR          //全求人
	CardTypeGroup_HDLY         //海底捞月
	CardTypeGroup_GSKH         //杠上开花
	CardTypeGroup_QL           //青龙
	CardTypeGroup_RH           //人胡
	CardTypeGroup_TT           //天听
	CardTypeGroup_SSBK         //十三不靠
	CardTypeGroup_QYS          //清一色
	CardTypeGroup_QD           //七对
	CardTypeGroup_DH           //地胡
	CardTypeGroup_SG           //三杠
	CardTypeGroup_TH           //天胡
	CardTypeGroup_ZYS          //字一色
	CardTypeGroup_SH           //四混
	CardTypeGroup_YSSTS        //一色三同顺
	CardTypeGroup_SSSTS        //三色三同顺
)

var (
	CardTypeGroup_Names = []string{
		"单调将", "缺一门", "明杠", "暗杠", "双暗刻", "双同刻", "门前清", "听牌", "不求人", "双明杠",
		"碰碰胡", "混一色", "全求人", "海底捞月", "杠上开花", "青龙", "人胡", "天听", "十三不靠", "清一色",
		"七对", "地胡", "三杠", "天胡", "字一色", "四混", "一色三同顺", "三色三同顺",
	}
	CardTypeGroup_Multiples = []int{
		4, 0, 0, 0, 0, 0, 0, 4, 0, 0,
		16, 16, 0, 24, 24, 48, 0, 40, 48, 40,
		48, 64, 0, 128, 0, 64, 48, 16,
	}
	//4|0|0|0|0|0|0|4|0|0|16|16|0|24|24|48|0|40|48|40|48|64|0|128|0|64|48|16
)

// var (
// 	myriads = []int{11, 12, 13, 14, 15, 16, 17, 18, 19}
// 	cakes   = []int{21, 22, 23, 24, 25, 26, 27, 28, 29}
// 	strips  = []int{31, 32, 33, 34, 35, 36, 37, 38, 39}
// 	winds   = []int{41, 42, 43, 44}
// 	arrows  = []int{51, 52, 53}
// 	flowers = []int{61, 62, 63, 64, 65, 66, 67, 68}
// )

// var (
// 	myriads = []int{11, 12, 13, 14, 15, 16, 17, 18, 19}
// 	cakes   = []int{}
// 	strips  = []int{}
// 	winds   = []int{}
// 	arrows  = []int{51, 52, 53}
// 	flowers = []int{61, 62, 63, 64}
// )

// var (
// 	myriads = []int{11, 12, 13, 14, 15, 16, 17, 18, 19}
// 	cakes   = []int{21, 22, 23, 24, 25, 26, 27, 28, 29}
// 	strips  = []int{31, 32, 33, 34, 35, 36, 37, 38, 39}
// 	winds   = []int{41, 42, 43, 44}
// 	arrows  = []int{51, 52, 53}
// 	flowers = []int{61, 62, 63, 64}
// )

var (
	myriads            = []int{11, 12, 13, 14, 15, 16, 17, 18, 19}
	cakes              = []int{21, 22, 23, 24, 25, 26, 27, 28, 29}
	strips             = []int{31, 32, 33, 34, 35, 36, 37, 38, 39}
	winds              = []int{41, 42, 43, 44}
	arrows             = []int{52, 53}
	arrows_without_mix = []int{51, 52, 53}
	flowers            = []int{61, 62, 63, 64}
	mixs               = []int{71}
)

type Card struct {
	//类型(1:万 2:饼 3:条 4:风 5:箭 6:花)
	Type int
	/*
	  万(11:一 12:二 13:三 14:四 15:五 16:六 17:七 18:八 19:九)
	  饼(21:一 22:二 23:三 24:四 25:五 26:六 27:七 28:八 29:九)
	  条(31:一 32:二 33:三 34:四 35:五 36:六 37:七 38:八 39:九)
	  风(41:东 42:南 43:西 44:北)
	  箭(51:中 52:发 53:白)
	  花(61:春 62:夏 63:秋 64:冬 65:梅 66:兰 67:竹 68:菊)
	*/
	ID int
	//一套牌中的索引
	Index int
	//是否是混子
	IsMix int
}

//是万饼条
func isWBT(cardid int) bool {
	if cardid/10 == CardType_Myriad || cardid/10 == CardType_Cake || cardid/10 == CardType_Strip {
		return true
	}
	return false
}

//是万饼条
func isMyriadCakeStrip(card *Card) bool {
	if card.Type == CardType_Myriad || card.Type == CardType_Cake || card.Type == CardType_Strip {
		return true
	}
	return false
}

//是风箭
func isFJ(cardid int) bool {
	if cardid/10 == CardType_Arrow || cardid/10 == CardType_Wind {
		return true
	}
	return false
}

//是风箭
func isWindArrow(card *Card) bool {
	if card.Type == CardType_Wind || card.Type == CardType_Arrow {
		return true
	}
	return false
}

//是混子
func isHZ(cardid int) bool {
	if cardid/10 == CardType_Mix {
		return true
	}
	return false
}

//是混子
func isMix(card *Card) bool {
	if card.Type == CardType_Mix {
		return true
	}
	return false
}

func NewCard(id int) *Card {
	var card Card
	if id == 11 { //==================万
		card = Card{1, 11, 0, 0}
	} else if id == 12 {
		card = Card{1, 12, 0, 0}
	} else if id == 13 {
		card = Card{1, 13, 0, 0}
	} else if id == 14 {
		card = Card{1, 14, 0, 0}
	} else if id == 15 {
		card = Card{1, 15, 0, 0}
	} else if id == 16 {
		card = Card{1, 16, 0, 0}
	} else if id == 17 {
		card = Card{1, 17, 0, 0}
	} else if id == 18 {
		card = Card{1, 18, 0, 0}
	} else if id == 19 {
		card = Card{1, 19, 0, 0}
	} else if id == 21 { //==================饼
		card = Card{2, 21, 0, 0}
	} else if id == 22 {
		card = Card{2, 22, 0, 0}
	} else if id == 23 {
		card = Card{2, 23, 0, 0}
	} else if id == 24 {
		card = Card{2, 24, 0, 0}
	} else if id == 25 {
		card = Card{2, 25, 0, 0}
	} else if id == 26 {
		card = Card{2, 26, 0, 0}
	} else if id == 27 {
		card = Card{2, 27, 0, 0}
	} else if id == 28 {
		card = Card{2, 28, 0, 0}
	} else if id == 29 {
		card = Card{2, 29, 0, 0}
	} else if id == 31 { //==================条
		card = Card{3, 31, 0, 0}
	} else if id == 32 {
		card = Card{3, 32, 0, 0}
	} else if id == 33 {
		card = Card{3, 33, 0, 0}
	} else if id == 34 {
		card = Card{3, 34, 0, 0}
	} else if id == 35 {
		card = Card{3, 35, 0, 0}
	} else if id == 36 {
		card = Card{3, 36, 0, 0}
	} else if id == 37 {
		card = Card{3, 37, 0, 0}
	} else if id == 38 {
		card = Card{3, 38, 0, 0}
	} else if id == 39 {
		card = Card{3, 39, 0, 0}
	} else if id == 41 { //==================风
		card = Card{4, 41, 0, 0}
	} else if id == 42 {
		card = Card{4, 42, 0, 0}
	} else if id == 43 {
		card = Card{4, 43, 0, 0}
	} else if id == 44 {
		card = Card{4, 44, 0, 0}
	} else if id == 51 { //==================箭
		card = Card{5, 51, 0, 0}
	} else if id == 52 {
		card = Card{5, 52, 0, 0}
	} else if id == 53 {
		card = Card{5, 53, 0, 0}
	} else if id == 61 { //==================花
		card = Card{6, 61, 0, 0}
	} else if id == 62 {
		card = Card{6, 62, 0, 0}
	} else if id == 63 {
		card = Card{6, 63, 0, 0}
	} else if id == 64 {
		card = Card{6, 64, 0, 0}
	} else if id == 65 {
		card = Card{6, 65, 0, 0}
	} else if id == 66 {
		card = Card{6, 66, 0, 0}
	} else if id == 67 {
		card = Card{6, 67, 0, 0}
	} else if id == 68 {
		card = Card{6, 68, 0, 0}
	} else if id == 71 { //混子
		card = Card{7, 71, 0, 1}
	}
	return &card
}

func (this *Card) clone() *Card {
	return &Card{this.Type, this.ID, this.Index, this.IsMix}
}
