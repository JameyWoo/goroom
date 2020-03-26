package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	// 需要指定server的 ip 端口号, 以及自己的用户名
	if len(os.Args) != 4 {
		fmt.Printf("Usage : %s <ip> <port> <user_name>\n", os.Args[0])
		os.Exit(1)
	}
	conn, err := net.Dial("tcp", os.Args[1] + ":" + os.Args[2])
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	done := make(chan struct{})
	go func() {
		// 将 conn 中的数据拷贝到 os.Stdout. 文件的拷贝也可以这样实现.
		// 这是将 conn读取的数据输出的意思.
		// ! 并且这个协程应该会一直读取, 而不会终止
		io.Copy(os.Stdout, conn)
		// 服务器断开连接的时候, 才会往下执行
		done <- struct{}{}
	}()
	if _, err := io.Copy(conn, os.Stdin); err != nil {
		panic(err)
	}
	<-done
}