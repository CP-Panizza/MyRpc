package main

import (
	"reflect"
	"net/rpc"
	"fmt"
	"net"
	"encoding/json"
	"time"
	"log"
	"sync"
)

//定义传输数据格式
type User struct {
	Name string
	Age  int
}

//定义rpc调用接口，通过tag定义接口中函数对应的远程服务名
type HelleServiceInterface struct {
	Hello func(int, *int) string       `service:"HelleService.Hello"`
	Say   func(string, *string) string `service:"HelleService.Say"`
}

type AddUser struct {
	Add func(int, *int) string `service:"AddUser.Add"`
}

func main() {
	service := HelleServiceInterface{}
	clinet := NewMyRpcClient("127.0.0.1")
	clinet.Implement(&service)
	var u *int
	err := service.Hello(0, u)
	if err != "nil" {
		println(err)
	}
	fmt.Println(u)
	var str string
	service.Say("dasfaf", &str)
	fmt.Println(str)
}

//构造MyRpcClient
func NewMyRpcClient(ip string) *MyRpcClient {
	clinet := new(MyRpcClient)
	clinet.serviceMap = map[string]*[]string{}
	clinet.RegisterCenterIp = ip
	return clinet
}

type MyRpcClient struct {
	mutex            sync.Mutex
	RegisterCenterIp string
	serviceList      []string
	serviceMap       map[string]*[]string
}

type pullRecvData struct {
	Ok   bool                    `json:"ok"`
	Msg  string                  `json:"msg"`
	Data [](map[string][]string) `json:"data"`
}

//从远端服务器拉取服务ip
func (this *MyRpcClient) pullService() error {
	type PullServiceData struct {
		Op          string
		ServiceList []string
	}
	conn, err := net.Dial("tcp", this.RegisterCenterIp+":8527")
	if err != nil {
		return err
	}

	sendData, err := json.Marshal(PullServiceData{
		"PULL",
		this.serviceList,
	})

	if err != nil {
		return err
	}

	_, err = conn.Write(sendData)
	if err != nil {
		return err
	}

	data := make([]byte, 4096)
	index, err := conn.Read(data)
	if err != nil {
		return err
	}

	fmt.Printf("resave: %s", data[:index])

	recvData := pullRecvData{}
	err = json.Unmarshal(data[:index], &recvData)
	if err != nil {
		return err
	}

	conn.Close()
	return nil
}

//负载均衡算法，通过传入服务名称，负载均衡算法计算得出对应的服务器IP地址--多线程环境
func (this *MyRpcClient) load_balance(serviceName string) (string, error) {
	return "127.0.0.1:8888", nil
}

//rpc调用远程服务--多线程环境
func (this *MyRpcClient) call(serverName string, serverIp string, args []reflect.Value) string {
	clinet, err := rpc.Dial("tcp", serverIp)
	if err != nil {
		return err.Error()
	}
	err = clinet.Call(serverName, args[0].Interface(), args[1].Interface())
	if err != nil {
		return err.Error()
	}
	clinet.Close()
	return "nil"
}

//动态对结构体进行代理,实现对应接口
func (this *MyRpcClient) Implement(i ...interface{}) {
	for _, v := range i {
		val := reflect.ValueOf(v).Elem()
		typ := reflect.TypeOf(v).Elem()
		for index := 0; index < val.NumField(); index++ {
			funcType := val.Field(index).Type()
			server := typ.Field(index).Tag.Get("service")
			this.serviceList = append(this.serviceList, server)
			proxyFunc := reflect.MakeFunc(funcType, func(args []reflect.Value) (results []reflect.Value) {
				serverIp, err := this.load_balance(server)
				if err != nil {
					return []reflect.Value{reflect.ValueOf(err.Error())}
				}
				errMsg := this.call(server, serverIp, args)
				return []reflect.Value{reflect.ValueOf(errMsg)}
			})
			val.Field(index).Set(proxyFunc)
		}
	}

	go func() {
		for {
			var sec time.Duration = 20000
			err := this.pullService()
			if err != nil {
				log.Println(err)
				sec = 60000
			}
			time.Sleep(sec)
		}
	}()
}
