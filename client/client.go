/*
协议:
用户可以执行命令, 以  %<command> args... 的形式
如:
	%set-name 姬小野
	%get-file 十万个为什么.pdf
可以直接输入消息, 也就是除合法命令格式以外的消息, 都作为发送到chatroom的消息来发送

由于tcp传输字节流, 因此在发送每个消息之前, 用一个 4字节 的数据声明要发送的消息的长度, 避免粘包.

*/

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func main() {
	// 需要指定server的 ip 端口号
	if len(os.Args) != 3 {
		fmt.Printf("Usage : %s <ip> <port>\n", os.Args[0])
		os.Exit(1)
	}
	conn, err := net.Dial("tcp", os.Args[1]+":"+os.Args[2])
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	done := make(chan struct{})
	go func() {
		// 将 conn 中的数据拷贝到 os.Stdout. 文件的拷贝也可以这样实现.
		// 这是将 conn读取的数据输出的意思.
		// ! 并且这个协程应该会一直读取, 而不会终止
		_, err := io.Copy(os.Stdout, conn)
		if err != nil {
			panic(err)
		}
		// 服务器断开连接的时候, 才会往下执行
		done <- struct{}{}
	}()
	// 这里不能使用 io.Copy 函数, 因为需要解析命令
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		inputStr := input.Text()
		// 将输入解析, 如果是命令的格式, 那么以命令的方式传递
		subInput := strings.Fields(inputStr)
		switch subInput[0] {
		// 在这里只需要 upload-file, get-file 是需要上下文操作的, 其他的命令只需要传递过去让server处理
		case "%upload-file": // 上传文件

		case "%get-file": // 下载文件

		case "%exit": // 退出聊天室
			return

		case "%set-name": // 设置用户的名字
			fallthrough
		case "%ls": // 列出聊天室中所有的文件
			fallthrough
		default:
			// 先计算数据长度, 然后拼接
			length := len([]byte(inputStr))
			preSend := BytesCombine(BytesCombine(IntToBytes(length)), []byte(inputStr))
			_, err := conn.Write(preSend)
			if err != nil {
				panic(err)
			}
		}
	}
	//if _, err := io.Copy(conn, os.Stdin); err != nil {
	//	panic(err)
	//}
	<-done
}

// 合并两个 []byte
func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

//整形转换成字节
func IntToBytes(n int) []byte {
	x := int32(n)

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

//字节转换成整形
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}