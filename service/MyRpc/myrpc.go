package MyRpc

import (
	"reflect"
	"net/rpc"
	"net"
	"strconv"
		"encoding/json"
	"fmt"
	"errors"
)

type MyRpc struct {
	infos []string
	RegisterCenterIp string
}


func NewMyRpc(ip string) *MyRpc{
	if ip == "" {
		panic(errors.New("SET INVALID IP"))
	}
	return &MyRpc{RegisterCenterIp:ip}
}


//传入结构体指针
func(this *MyRpc)Register(components interface{}){
	t := reflect.TypeOf(components).Elem()
	v := reflect.ValueOf(components).Elem()
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i).Type
		newInstance := reflect.New(fieldType).Interface()
		this.getInfo(newInstance)
		err := rpc.Register(newInstance)
		if err != nil {
			panic(err)
		}
	}
}


func (this *MyRpc)getInfo(instance interface{}){
	v := reflect.ValueOf(instance)
	t := reflect.TypeOf(instance)
	structName := v.Elem().Type().Name()
	for i := 0; i < v.NumMethod(); i++ {
		name := t.Method(i).Name
		serviceName := structName + "." + name
		this.infos = append(this.infos, serviceName)
	}
}



func (this *MyRpc)sendDataToRegistCenter(port int){
	type RegiestServiceData struct {
		Op          string
		ServiceList []string
		ServicePort string
	}

	conn, err := net.Dial("tcp", this.RegisterCenterIp + ":8527")
	if err != nil {
		panic(err)
	}

	sendData, err := json.Marshal(RegiestServiceData{
		"REG",
		this.infos,
		":" + strconv.Itoa(port),
	})

	if err != nil {
		panic(err)
	}

	_, err = conn.Write(sendData)
	if err != nil {
		panic(err)
	}

	data := make([]byte, 4096)
	index, err := conn.Read(data)
	if err != nil {
		panic(err)
	}
	fmt.Printf("success regist, resave: %s", data[:index])
	conn.Close()
}

func (this *MyRpc)startHeartCheck(){
	listen, err := net.ListenTCP("tcp", &net.TCPAddr{Port:8528})

	if err != nil {
		panic(err)
	}

	for {
		accept(listen)
	}
}


func accept(listen *net.TCPListener){
	connect, err := listen.Accept()
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 64)
	index, err := connect.Read(buf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("resave: %s\n", buf[:index])
	connect.Write([]byte("hello registerCenter!"))
	connect.Close()
}

func (this *MyRpc)StartServer(port int){
	go this.startHeartCheck() //开启心跳检测
	this.sendDataToRegistCenter(port) //发送数据到注册中心
	listen, err := net.Listen("tcp", ":" + strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	rpc.Accept(listen) //开启服务监听
}
