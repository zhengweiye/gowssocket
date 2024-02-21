package main

import (
	"fmt"
	"time"
)

func main() {
	valueChan := make(chan int)
	closeChan := make(chan bool)
	go test(valueChan, closeChan)

	time.Sleep(5 * time.Second)
	valueChan <- 5

	time.Sleep(5 * time.Second)

	//TODO 必须先关闭closeChan,才能关闭valueChan
	close(closeChan)
	close(valueChan)

	fmt.Println("停止服务了")
	time.Sleep(10 * time.Second)
}

func test(c chan int, quit chan bool) {
	//TODO 取完chan的值，会一直取零值===>需要退出监听
	defer fmt.Println("退出select监听")
	for {
		select {
		case v := <-c:
			fmt.Println("value=", v)
		case <-quit:
			return
		}
	}

	//TODO 取完chan的值，不会取零值
	/*for v := range c {
		fmt.Println("value=", v)
	}*/
}
