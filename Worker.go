package gowssocket

import (
	"fmt"
	"runtime"
)

type Job func()

type Worker struct {
	JobQueue chan Job
	quitChan chan bool
}

func newWorker() Worker {
	return Worker{
		JobQueue: make(chan Job, 100),
		quitChan: make(chan bool, 1),
	}
}

func (w *Worker) run() {
	go func() {
		for {
			select {
			case job := <-w.JobQueue:
				job()
			case <-w.quitChan:
				fmt.Println("worker quitChan select is quit.")
				return
			}
		}
	}()
}

type Pool struct {
	requestQueue chan Job
	workers      []Worker
	balance      Balance
	quitChan     chan bool
}

var deferRequestQueueSize int = 1000

func newPool(maxWorkers int) Pool {
	if maxWorkers == 0 {
		maxWorkers = runtime.NumCPU() << 2
	}

	pool := Pool{
		requestQueue: make(chan Job, deferRequestQueueSize),
		workers:      make([]Worker, maxWorkers),
		balance:      newBalance(),
		quitChan:     make(chan bool, 1),
	}

	pool.run()

	return pool
}

func (p *Pool) run() {
	// 创建Worker（真正执行任务的）
	for i := 0; i < len(p.workers); i++ {
		// 创建线程
		worker := newWorker()
		// 启动线程监听任务
		worker.run()
		// 把worker的chan放入WorkerQueue
		p.workers[i] = worker
	}

	// 监听线程池是否有任务
	go func() {
		for {
			select {
			case request := <-p.requestQueue:
				//TODO #issue,如何并发很高时，jobQueue满了，会不会导致数据丢失？==>一直堵塞在这里
				p.workers[p.balance.Get(uint32(len(p.workers)))].JobQueue <- request
			case <-p.quitChan:
				fmt.Println("pool quitChan select is quit.")
				return
			}
		}
	}()
}

func (p *Pool) execTask(job Job) {
	p.requestQueue <- job
}

func (p *Pool) shutdown() {
	for _, worker := range p.workers {
		close(worker.quitChan)
		close(worker.JobQueue)
	}
	close(p.quitChan)
	close(p.requestQueue)
}
