package main

import (
	"fmt"
	socketUtils "github.com/JameyWoo/goroom/socketUtils"
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
	entering    = make(chan client)
	leaving     = make(chan client)
	messages    = make(chan string)     // all incoming client messages
	downloading = make(map[client]bool) // 下载状态不接收信息
)

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

// 广播的协程, 处理三种信号: 广播, 进入, 离开
func broadcaster() {
	// 遍历 clients 的话, 以 client 也就是 chan client 为键
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// ! 注意, 这里msg是字符串, 而不是 chan !!! 所以可以给每个 cli 赋值!!!
			// 打印服务端信息
			fmt.Println(msg)
			for cli := range clients {
				// ! 会继续执行阻塞的 clientWriter 函数
				// 设置一个下载状态
				if !downloading[cli] {
					cli <- msg
				} else {
					fmt.Println("file downloading")
				}
			}
		// 有客户端进入
		// ! 注意, 这里的键是 entering 传递来的, 而 entering 是从 handleConn中声明的 ch传递过来的
		// ! chan 是一个引用类型, 所以把那个ch传递来, 在上面改编了cli的话, 就会继续执行clientWriter里的循环输出
		case cli := <-entering:
			clients[cli] = true
			downloading[cli] = false
		// 有客户端离开
		case cli := <-leaving:
			delete(clients, cli)
			delete(downloading, cli)
			close(cli)
		}
	}
}

// 处理每个客户端连接的协程
func handleConn(conn net.Conn) {
	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	var alias string // 用户自定义的别名
	ch <- "You are " + who
	// * 要被广播的message, broadcaster 会捕捉
	messages <- who + " has arrived"
	entering <- ch // 表示新的用户进入了

	for {
		inputByte := socketUtils.ReceiveBytesFromConn(conn)
		inputStr := string(inputByte)
		// 将输入解析, 如果是命令的格式, 那么以命令的方式传递
		subInput := strings.Fields(inputStr)
		if len(subInput) < 1 {
			// 用break而不是continue, continue 的话即使client断开了连接, 资源也不会释放
			break
		}

		// 分别处理每一个命令
		switch subInput[0] {
		// 在这里只需要 upload, download  是需要上下文操作的, 其他的命令只需要传递过去让server处理
		case "%upload": // 上传文件
			HandleUploadServer(subInput, conn, ch)

		case "%download": // 下载文件
			ch <- inputStr
			handleDownloadServer(subInput, conn, ch, downloading)

		case "%set-name": // 设置用户的名字
			if len(subInput) >= 2 { // 取 %set-name 之后的第一个字符串为名字
				alias = subInput[1]
				ch <- "设置用户名 \"" + alias + "\" 成功!"
			} else {
				ch <- "设置用户名失败! 请输入合法的用户名!"
			}
		case "%ls": // 列出聊天室中所有的文件
			// TODO: 要注意这里如果在其他目录运行server.go, 这个目录还是否有效
			if !socketUtils.Exists("./disk") {
				err := os.Mkdir("disk", os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
			}
			files, err := ioutil.ReadDir("./disk")
			if err != nil {
				panic(err)
			}
			var filesStr string
			for _, file := range files {
				filesStr += file.Name() + "\t"
			}
			// 这里是单独给当前客户端发送信息
			ch <- filesStr
		default:
			// 广播信息
			if len(alias) > 0 {
				messages <- who + "(" + alias + ")" + ": " + inputStr
			} else {
				messages <- who + ": " + inputStr
			}
		}
	}
	// ! .Scan() 为false之后, 不再输入, 说明socket断开连接, 告知离开
	leaving <- ch
	messages <- who + "(" + alias + ")" + " has left"
	conn.Close()
}

func HandleUploadServer(subInput []string, conn net.Conn, ch client) {
	if len(subInput) >= 2 {
		for _, filename := range subInput[1:] {
			// 首先判断文件是否存在, 如果存在那么无法写入
			// filename 需要经过解析. 以 " / " 作为分隔符
			subFilename := strings.Split(filename, "/")
			newFilename := "./disk/" + subFilename[len(subFilename)-1]
			if !socketUtils.Exists("./disk") {
				// 如果 disk文件夹不存在, 那么创建
				err := os.Mkdir("disk", os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
			}
			if socketUtils.Exists(newFilename) {
				// 如果存在, 那么取消上传该文件并通报
				ch <- "服务器上存在同名文件 \"" + subFilename[len(subFilename)-1] + " \", 将覆盖该文件!"
				err := os.Remove(newFilename)
				if err != nil {
					log.Fatal(err)
				}
			}
			// 打开文件, 计算字节
			fileByte := socketUtils.ReceiveBytesFromConn(conn)
			newFile, err := os.Create(newFilename)
			if err != nil {
				log.Fatal(err)
			}
			_, err = newFile.Write(fileByte)
			if err != nil {
				log.Fatal(err)
			} else {
				ch <- "upload " + filename + " successed!"
			}
			newFile.Close()
		}
	}
}

func handleDownloadServer(subInput []string, conn net.Conn, ch client, downloading map[client]bool) {
	if len(subInput) >= 2 {
		downloading[ch] = true
		// TODO: 有的文件是不存在的, 需要加一个检测, 否则会终止程序
		for _, filename := range subInput[1:] {
			// 打开文件, 计算字节
			newFilename := "disk/" + filename
			fileByte, err := ioutil.ReadFile(newFilename)
			if err != nil {
				ch <- "file not exist"
				continue
			}
			fileByteLen := len(fileByte)
			preSend := socketUtils.BytesCombine(socketUtils.IntToBytes(fileByteLen), fileByte)
			_, err = conn.Write(preSend)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		ch <- "文件下载失败, 请给出文件名, 可同时下载多个文件"
	}
	downloading[ch] = false // 下载结束, 可以接收信息
}

// 向客户端写入数据的协程
// ! ch循环阻塞, 知道ch被传入了数据, 这个协程不会直接终止
func clientWriter(conn net.Conn, ch <-chan string) {
	// * range 遍历, 当 ch 为空的时候, 这个语句会阻塞. 当 ch 得到了值, 他又会醒过来
	for msg := range ch {
		msgByte := []byte(msg)
		msgByteNew := socketUtils.BytesCombine(socketUtils.IntToBytes(len(msgByte)), msgByte)
		conn.Write(msgByteNew)
	}
}
