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
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

var (
	downloading bool
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
		//_, err := io.Copy(os.Stdout, conn)
		//if err != nil {
		//	panic(err)
		//}
		//input := bufio.NewScanner(conn)
		//for input.Scan() {
		//	fmt.Println(input.Text())
		//}
		for {
			inputByte := ReceiveByteFromClient(conn)
			inputStr := string(inputByte)
			if inputStr[:9] == "%get-file" {
				subInput := strings.Fields(inputStr)
				if len(subInput) >= 2 {
					for _, filename := range subInput[1:] {
						fmt.Println("downloading " + filename + " ...")
						// 首先判断文件是否存在, 如果存在那么无法写入
						newFilename := "./disk/" + filename
						if !Exists("./disk") {
							// 如果 disk文件夹不存在, 那么创建
							err := os.Mkdir("disk", os.ModePerm)
							if err != nil {
								log.Fatal(err)
							}
						}
						if Exists(newFilename) {
							// 如果存在, 那么取消上传该文件并通报
							fmt.Println("服务器上存在同名文件 \"" + filename + " \", 将覆盖该文件!")
							err := os.Remove(newFilename)
							if err != nil {
								log.Fatal(err)
							}
						}
						// 打开文件, 计算字节
						fileByte := ReceiveByteFromClient(conn)
						fmt.Println("fileByte len:", len(fileByte))
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
			} else {
				fmt.Println(inputStr)
			}
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
		if len(subInput) < 1 {
			continue
		}
		switch subInput[0] {
		// 在这里只需要 upload-file, get-file 是需要上下文操作的, 其他的命令只需要传递过去让server处理
		case "%upload-file": // 上传文件
			HandleUploadFileClient(conn, inputStr, subInput)

		case "%get-file": // 下载文件
			// 尝试
			SendBytesToServer(conn, []byte(inputStr))

		case "%exit": // 退出聊天室
			return

		case "%set-name": // 设置用户的名字
			fallthrough
		case "%ls": // 列出聊天室中所有的文件
			fallthrough
		default:
			// 发送命令
			SendBytesToServer(conn, []byte(inputStr))
		}
	}
	//if _, err := io.Copy(conn, os.Stdin); err != nil {
	//	panic(err)
	//}
	<-done
}

// 从客户端读取字节流
func ReceiveByteFromClient(conn net.Conn) []byte {
	// 先读取 4 字节, 作为长度
	lengthByte := make([]byte, 4)
	conn.Read(lengthByte)            // 忽略错误
	length := BytesToInt(lengthByte) // int 类型的长度
	// TODO: 注意这里如果是传输比较大的文件的话, 是否需要拆分成小的段?
	inputByte := make([]byte, length) // 输入命令
	length, _ = conn.Read(inputByte)  // 忽略错误
	return inputByte
}

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func HandleUploadFileClient(conn net.Conn, inputStr string, subInput []string) {
	// 先发送命令
	SendBytesToServer(conn, []byte(inputStr))
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
			preSend := BytesCombine(BytesCombine(IntToBytes(fileByteLen)), fileByte)
			_, err = conn.Write(preSend)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		fmt.Println("文件上传失败, 请给出文件名, 可同时上传多个文件")
	}
}

// 向服务器发送命令
// 先计算数据长度, 然后拼接
func SendBytesToServer(conn net.Conn, inputStrByte []byte) {
	length := len(inputStrByte)
	preSend := BytesCombine(IntToBytes(length), inputStrByte)
	_, err := conn.Write(preSend)
	if err != nil {
		panic(err)
	}
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