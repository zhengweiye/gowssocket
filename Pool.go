package gowssocket

import (
	"context"
	"fmt"
	"github.com/zhengweiye/gopool"
	"sync"
)

func getPool(ctx context.Context, wg *sync.WaitGroup) *gopool.Pool {
	pool := gopool.NewPool(1000, 1000, ctx, wg)
	fmt.Printf(">>> [WebSocket] 协程池%p\n", pool)
	return pool
}
