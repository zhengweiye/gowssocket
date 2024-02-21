package gowssocket

import (
	"os"
	"os/signal"
	"syscall"
)

var wsSignalChan = make(chan os.Signal, 1)

/**
 * 一个服务端存在多个ws的url,但是这些ws的url只共享一个信号量
 */

func Init() {
	signal.Notify(wsSignalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGILL,
		syscall.SIGTRAP,
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		syscall.SIGKILL,
		syscall.SIGSEGV,
		syscall.SIGPIPE,
		syscall.SIGALRM,
		syscall.SIGTERM, //terminated ==> docker重启项目会触发该信号
	)
}
