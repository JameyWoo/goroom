package main

import (
	"fmt"
	"net"
	"os"
)

// 不错的博客: https://blog.csdn.net/yjp19871013/article/details/82711237

func main() {
	// 命令行参数
	if len(os.Args) != 2 {
		fmt.Printf("Usage : %s <port>\n", os.Args[0])
		os.Exit(1)
	}
	// 监听端口, 使用select处理
	listener, err := net.Listen("tcp", "localhost:6666")
	if err != nil {
		panic(err)
	}
	// 一个广播协程, 专门用来给所有客户端通信的
	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		// 一个处理conn连接的协程
		go handleConn(conn)
	}
}

func broadcaster() {

}

func handleConn(conn net.Conn)  {

}