/*
协议:
用户可以执行命令, 以  %<command> args... 的形式
如:
	%ls                         // 查看server中的文件
	%exit                       // 推出客户端
	%set-name 姬小野             // 设置用户名
	%download 十万个为什么.pdf    // 下载文件
	%upload test.go             // 上传文件

可以直接输入消息, 也就是除合法命令格式以外的消息, 都作为发送到chatroom的消息来发送

由于tcp传输字节流, 因此在发送每个消息之前, 用一个 4字节 的数据声明要发送的消息的长度, 避免粘包.
*/

package main

import (
	"bufio"
	socketUtils "chatroom/socketUtils"
	"fmt"
	"io/ioutil"
	"log"
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
	go HandleOutput(conn, done) // 处理输出, 包括文件输出(下载)

	// 这里不能使用 io.Copy 函数, 因为需要解析命令
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		inputStr := input.Text()
		// 将输入解析, 如果是命令的格式, 那么以命令的方式传递
		subInput := strings.Fields(inputStr)
		if len(subInput) < 1 {
			continue
		}
		switch subInput[0] {
		// 在这里只需要 upload, download 是需要上下文操作的, 其他的命令只需要传递过去让server处理
		case "%upload": // 上传文件
			HandleUpload(conn, inputStr, subInput)

		case "%download": // 下载文件
			// 尝试
			socketUtils.SendBytesToConn(conn, []byte(inputStr))

		case "%exit": // 退出聊天室
			return
		case "%set-name": // 设置用户的名字
			fallthrough
		case "%ls": // 列出聊天室中所有的文件
			fallthrough
		default:
			// 发送命令
			socketUtils.SendBytesToConn(conn, []byte(inputStr))
		}
	}
	<-done
}

// 处理输出. 包括标准输出和文件输出
func HandleOutput(conn net.Conn, done chan struct{}) {
	for {
		inputByte := socketUtils.ReceiveBytesFromConn(conn)
		inputStr := string(inputByte)
		if len(inputStr) < 9 {
			fmt.Println(inputStr)
		} else {
			if inputStr[:9] == "%download" {
				HandleDownload(inputStr, conn)
			} else {
				fmt.Println(inputStr)
			}
		}
		fmt.Print("$ ")
	}
	// 服务器断开连接的时候, 才会往下执行
	done <- struct{}{}
}

func HandleDownload(inputStr string, conn net.Conn) {
	subInput := strings.Fields(inputStr)
	if len(subInput) >= 2 {
		for _, filename := range subInput[1:] {
			fmt.Println("downloading " + filename + " ...")
			// 首先判断文件是否存在, 如果存在那么无法写入
			newFilename := "./disk/" + filename
			if !socketUtils.Exists("./disk") {
				// 如果 disk文件夹不存在, 那么创建
				err := os.Mkdir("disk", os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
			}
			// 打开文件, 计算字节
			fileByte := socketUtils.ReceiveBytesFromConn(conn)
			if len(fileByte) == 14 && string(fileByte) == "file not exist" {
				fmt.Println("file \"" + filename + "\" not exist")
			} else {
				if socketUtils.Exists(newFilename) {
					// 如果存在, 那么取消上传该文件并通报
					fmt.Println("客户端上存在同名文件 \"" + filename + " \", 继续将覆盖该文件!")
					err := os.Remove(newFilename)
					if err != nil {
						log.Fatal(err)
					}
				}
				newFile, err := os.Create(newFilename)
				if err != nil {
					log.Fatal(err)
				}
				_, err = newFile.Write(fileByte)
				if err != nil {
					log.Fatal(err)
				} else {
					fmt.Println("download " + filename + " successed!")
				}
				newFile.Close()
			}
		}
	}
}

func HandleUpload(conn net.Conn, inputStr string, subInput []string) {
	// 先发送命令
	socketUtils.SendBytesToConn(conn, []byte(inputStr))
	// 然后上传文件
	// 可以同时上传多个文件
	if len(subInput) >= 2 {
		// TODO: 有的文件是不存在的, 需要加一个检测, 否则会终止程序
		for _, filename := range subInput[1:] {
			// 打开文件, 计算字节
			fileByte, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatal(err)
			}
			fileByteLen := len(fileByte)
			preSend := socketUtils.BytesCombine(socketUtils.IntToBytes(fileByteLen), fileByte)
			_, err = conn.Write(preSend)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		fmt.Println("文件上传失败, 请给出文件名, 可同时上传多个文件")
	}
}
