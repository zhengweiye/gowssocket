package gowssocket

import "time"

type TimerFunc func()

/*
* delay  首次延迟
* period  间隔
* fun  定时执行的方法
* param  方法的参数
 */

func timer(delay, period time.Duration, fun TimerFunc) {
	go func() {
		if fun == nil {
			return
		}
		t := time.NewTimer(delay)
		for {
			select {
			case <-t.C:
				fun()
				t.Reset(period)
			}
		}
	}()
}
