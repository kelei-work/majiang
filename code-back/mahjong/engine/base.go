package engine

import (
	"fmt"
	"runtime"
	"time"

	"combine.com/utils/delay"
	"combine.com/utils/frame"
)

func init() {
	return
	task := delay.NewTask(time.Second)
	task.CycleMode = delay.CYCLEMODE_FOREVER
	task.Exec = func() {
		fmt.Printf("当前goroutine数量: %d\n", runtime.NumGoroutine())
	}
	task.Start()
}

//获取游戏配置
func getGameConfigInfo(args ...string) ([]interface{}, error) {
	return frame.GetGameConfigInfo(PLATFORM, args...)
}

//获取平台配置
func getConfigInfo(args ...string) ([]interface{}, error) {
	return frame.GetConfigInfo(args...)
}

var (
	generateDeck = func() (deck []Card) {
		deck = []Card{}
		//万
		for _, myriad := range myriads {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Myriad, myriad, 0, 0})
			}
		}
		//饼
		for _, cake := range cakes {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Cake, cake, 0, 0})
			}
		}
		//条
		for _, strip := range strips {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Strip, strip, 0, 0})
			}
		}
		//风
		for _, wind := range winds {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Wind, wind, 0, 0})
			}
		}
		//箭
		for _, arrow := range arrows {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Arrow, arrow, 0, 0})
			}
		}
		//花
		for _, flower := range flowers {
			deck = append(deck, Card{CardType_Flower, flower, 0, 0})
		}
		//混
		for _, mix := range mixs {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Mix, mix, 0, 1})
			}
		}
		// for i, card := range deck {
		// 	println(i, card.ID)
		// }
		return deck
	}
	generateDeck_without_mix = func() (deck []Card) {
		deck = []Card{}
		//万
		for _, myriad := range myriads {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Myriad, myriad, 0, 0})
			}
		}
		//饼
		for _, cake := range cakes {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Cake, cake, 0, 0})
			}
		}
		//条
		for _, strip := range strips {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Strip, strip, 0, 0})
			}
		}
		//风
		for _, wind := range winds {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Wind, wind, 0, 0})
			}
		}
		//箭
		for _, arrow := range arrows_without_mix {
			for i := 0; i < 4; i++ {
				deck = append(deck, Card{CardType_Arrow, arrow, 0, 0})
			}
		}
		//花
		for _, flower := range flowers {
			deck = append(deck, Card{CardType_Flower, flower, 0, 0})
		}
		return deck
	}
)

func GetDeck() []Card {
	return generateDeck()
}

type CardList []*Card

func (list CardList) Len() int {
	return len(list)
}

func (list CardList) Less(i, j int) bool {
	iID := list[i].ID
	jID := list[j].ID
	if iID == HUN {
		return true
	}
	if jID == HUN {
		return false
	}
	if iID < jID {
		return true
	} else {
		return false
	}
}

func (list CardList) Swap(i, j int) {
	var temp *Card = list[i]
	list[i] = list[j]
	list[j] = temp
}
