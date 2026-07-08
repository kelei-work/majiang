package engine

import (
	"time"

	"combine.com/utils/delay"
)

//房间守护线程开启
func (this *Room) roomDaemonThreadStart() {
	f := func() {
	}
	this.daemonThread = &delay.Task{
		Key: "roomDaemonThread",
		Exec: func() {
			f()
		},
		SurplusTime: time.Second * 10,
		CycleMode:   delay.CYCLEMODE_FOREVER,
	}
	this.daemonThread.Start()
}
