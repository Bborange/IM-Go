package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	conn   net.Conn
	server *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C:    make(chan string),
		conn: conn,
		
		server: server,
	}

	go user.ListenMessage()

	return user
}

//监听当前User channel的方法，一旦有消息，就发给conn
func (this *User) ListenMessage() {
	for {
		msg := <-this.C

		this.conn.Write([]byte(msg + "\n"))
	}
}

//上线
func (this *User) Online() {
	//用户上线，添加到OnlineMap
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()

	//广播上线消息
	this.server.BroadCast(this, "上线")
}

//下线
func (this *User) Offline() {
	//用户下线，删除map中的用户
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap,this.Name) 
	this.server.mapLock.Unlock()

	//广播下线消息
	this.server.BroadCast(this, "下线啦")
}

func (this *User)SendMsg(msg string){
	this.conn.Write([]byte(msg+"\n"))
}
//用户处理消息
func (this *User) DoMessage(msg string) {
	if msg == "who"{
		//查询有谁在线
		this.server.mapLock.Lock()
		for _,user := range this.server.OnlineMap{
			onlineMsg := "["+user.Addr+"]"+user.Name+":在线...\n"
			this.SendMsg(onlineMsg)
		}
		this.server.mapLock.Unlock()
	}else if len(msg)>7 && msg[:7] == "rename|"{
		//rename|zhangsan
		newName:= strings.Split(msg,"|")[1]

		//判断name是否存在
		_,ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前用户名已被使用\n")
		}else{
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap,this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()
			this.Name = newName

			this.SendMsg("修改成功:"+newName+"\n")
		}
	}else if len(msg)>4 && msg[0:3] == "to|" {
		//私聊消息

		//获取对方名字
		toName := strings.Split(msg, "|")[1]
		content := strings.Split(msg, "|")[2]

		//查询是否存在
		toUser, ok := this.server.OnlineMap[toName]
		if !ok {
			this.SendMsg("该用户名不存在\n")
			return
		}
		//给对方发消息
		if content == "" {
			this.SendMsg("消息不能为空\n")
			return
		}
		toUser.SendMsg(this.Name + ":msg" + content)
	}else{
		this.server.BroadCast(this,msg)
	}
}