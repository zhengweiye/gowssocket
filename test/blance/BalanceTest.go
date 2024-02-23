package main

import (
	"fmt"
	"github.com/zhengweiye/gowssocket"
	"sync"
)

func main() {
	balance := gowssocket.RoundRobinBalance{}
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			value := balance.Get(5)
			fmt.Println("index=", index, ", 拿到的值=", value)
		}(i)
	}
	wg.Wait()
}
