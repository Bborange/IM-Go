package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//在线用户列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	//消息广播的channel
	Message chan string
}

//创建一个server接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}

	return server
}

//监听Message广播channel的goroutine,有消息就发给全部User
func (this *Server) ListenMessager() {
	for {
		//获取数据
		msg := <-this.Message

		//遍历发送
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

//广播方法
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	this.Message <- sendMsg
}
func (this *Server) Handler(conn net.Conn) {
	//当前连接的业务
	// fmt.Println("连接建立成功")

	user := NewUser(conn, this)

	user.Online()

	//监听用户是否活跃的channel
	isLive := make(chan bool)

	//接受客户发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				return
			}

			//提取消息 取出\n
			msg := string(buf[:n-1])

			user.DoMessage(msg)

			isLive <- true
		}
	}()

	//当前handler阻塞
	for {
		select {

			//用户活跃重置定时器
		case <-time.After(time.Second * 100):
			//已经超时
			//强制关闭当前User

			user.SendMsg("你被踢了")

			//销毁资源
			close(user.C)
			conn.Close()
			//退出当前handler
			return
		case <-isLive:
		}

	}
}

//启动服务器
func (this *Server) Start() {
	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.listen err:", err)
		return
	}
	defer listener.Close()

	go this.ListenMessager()
	for {
		//  conn  <- accpet
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept error:", err)
			continue
		}

		//do handler
		go this.Handler(conn)

	}

}
