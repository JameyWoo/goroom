// 不错的博客: https://blog.csdn.net/yjp19871013/article/details/82711237
// TODO: 发现一个bug, 用 panic 会造成server终止, 当client的连接中断时, 应该忽略错误
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

// 控制客户端输出的channel
type client chan<- string

// 三个值, 分别是 进入, 离开, 要广播的消息. <-messages 是个 string 类型
var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string) // all incoming client messages
)

// 广播的协程, 处理三种信号: 广播, 进入, 离开
func broadcaster() {
	// 遍历 clients 的话, 以 client 也就是 chan client 为键
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			// ! 注意, 这里msg是字符串, 而不是 chan !!! 所以可以给每个 cli 赋值!!!
			// 打印服务端信息
			fmt.Println(msg)
			for cli := range clients {
				// ! 会继续执行阻塞的 clientWriter 函数
				cli <- msg
			}
		// 有客户端进入
		// ! 注意, 这里的键是 entering 传递来的, 而 entering 是从 handleConn中声明的 ch传递过来的
		// ! chan 是一个引用类型, 所以把那个ch传递来, 在上面改编了cli的话, 就会继续执行clientWriter里的循环输出
		case cli := <-entering:
			clients[cli] = true
		// 有客户端离开
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

// 处理每个客户端连接的协程
func handleConn(conn net.Conn) {
	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	var alias string  // 用户自定义的别名
	ch <- "You are " + who
	// * 要被广播的message, broadcaster 会捕捉
	messages <- who + " has arrived"
	entering <- ch // 表示新的用户进入了

	for {
		// 先读取 4 字节, 作为长度
		lengthByte := make([]byte, 4)
		length, _ := conn.Read(lengthByte)  // 忽略错误
		if length != 4 {  // 读取 int 类型的长度
			continue
		}
		length = BytesToInt(lengthByte)  // int 类型的长度
		inputByte := make([]byte, length) // 输入命令
		length, _ = conn.Read(inputByte)  // 忽略错误
		inputStr := string(inputByte)
		// 将输入解析, 如果是命令的格式, 那么以命令的方式传递
		subInput := strings.Fields(inputStr)

		// 分别处理每一个命令
		switch subInput[0] {
		// 在这里只需要 upload-file, get-file 是需要上下文操作的, 其他的命令只需要传递过去让server处理
		case "%upload-file": // 上传文件

		case "%get-file": // 下载文件

		case "%set-name": // 设置用户的名字
			if len(subInput) >= 2 {  // 取 %set-name 之后的第一个字符串为名字
				alias = subInput[1]
				ch <-"设置用户名 \"" + alias + " \"成功!"
			} else {
				ch <-"设置用户名失败! 请输入合法的用户名!"
			}
		case "%ls": // 列出聊天室中所有的文件
			// TODO: 要注意这里如果在其他目录运行server.go, 这个目录还是否有效
			files, err := ioutil.ReadDir("./disk")
			if err != nil {
				panic(err)
			}
			var filesStr string
			for _, file := range files {
				filesStr += file.Name() + "\t"
			}
			// 这里是单独给当前客户端发送信息
			ch <-filesStr
		default:
			// 广播信息
			if len(alias) > 0 {
				messages <- who + "(" + alias + ")" +  ": " + inputStr
			} else {
				messages <- who + ": " + inputStr
			}
		}
	}

	//// ! 读取网络输入, 然后传递给message用以广播
	//input := bufio.NewScanner(conn)
	//// .Scan() 函数为true时, 代表在输入, 否则代表不输入了(应该是文件关闭了)
	//// ! 这里实现的是服务端读取客户端的网络输入, 并将其传递给messages用以广播
	//for input.Scan() {
	//
	//	messages <- who + ": " + input.Text()
	//}
	// NOTE: ignoring potential errors from input.Err()

	// ! .Scan() 为false之后, 不再输入, 说明socket断开连接, 告知离开
	leaving <- ch
	messages <- who + alias + " has left"
	conn.Close()
}

// 向客户端写入数据的协程
// ! ch循环阻塞, 知道ch被传入了数据, 这个协程不会直接终止
func clientWriter(conn net.Conn, ch <-chan string) {
	// * range 遍历, 当 ch 为空的时候, 这个语句会阻塞. 当 ch 得到了值, 他又会醒过来
	for msg := range ch {
		// 输出到客户端
		fmt.Fprintln(conn, msg)
	}
}

func main() {
	// 命令行参数, 指定运行端口
	if len(os.Args) != 2 {
		fmt.Printf("Usage : %s <port>\n", os.Args[0])
		os.Exit(1)
	}
	listener, err := net.Listen("tcp", "localhost:"+os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
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