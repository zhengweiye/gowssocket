package gowssocket

import "github.com/zhengweiye/gopool"

var pool *gopool.Pool

func getPool() *gopool.Pool {
	return gopool.NewPool(1000, 1000)
}
