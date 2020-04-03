package main

import (
	"reflect"
	"net/rpc"
	"fmt"
	"net"
	"encoding/json"
	"sync"
	"time"
	"github.com/pkg/errors"
	"math/rand"
	"log"
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
	client := NewMyRpcClient("127.0.0.1")
	client.Implement(&service)
	err := client.Init()
	if err != nil {
		panic(err)
	}
	client.StartPull(time.Second * 5,func(err error) {
		if err != nil {
			log.Println(err)
		}
	})

	for i := 5; i <= 10; i++ {
		go func() {
			for {
				var u int
				err := service.Hello(10, &u)
				if err != "nil" {
					println(err)
				}
				fmt.Println(u)
				var str string
				err = service.Say("dasfaf", &str)
				if err != "nil" {
					println(err)
				}
				fmt.Println(str)
				time.Sleep(time.Second * 2)
			}
		}()
	}

	for {
		time.Sleep(time.Minute)
	}
}

//构造MyRpcClient
func NewMyRpcClient(ip string) *MyRpcClient {
	clinet := new(MyRpcClient)
	clinet.serviceMap = map[string][]*CircuitBreaker{}
	clinet.RegisterCenterIp = ip
	return clinet
}

type MyRpcClient struct {
	mutex            sync.RWMutex
	RegisterCenterIp string
	serviceList      []string
	serviceMap       map[string][]*CircuitBreaker
}

const (
	HALF_OPEN int = 0
	CLOSED    int = 1
	OPEN      int = 2
)

//熔断器
type CircuitBreaker struct {
	Proportion       int   			//占比
	failuerThreshold int           //故障次数阈值
	retryTimePeriod  time.Duration //失败后从新尝试的时间间隔 纳秒
	lastFailureTime  time.Duration
	failureCount     int
	state            int
	ServiceIp        string
	Mutex            sync.Mutex
}

func NewCircuitBreaker(serviceIp string,proportion int ,failuerThreshold int, retryTimePeriod time.Duration) *CircuitBreaker {
	cb := new(CircuitBreaker)
	cb.Proportion = proportion
	cb.ServiceIp = serviceIp
	cb.failuerThreshold = failuerThreshold
	cb.failureCount = 0
	cb.retryTimePeriod = retryTimePeriod
	cb.lastFailureTime = 0
	cb.state = CLOSED
	return cb
}

func (this *CircuitBreaker) GetState() int {
	return this.state
}

func (this *CircuitBreaker) ReSet() {
	this.failureCount = 0
	this.lastFailureTime = 0
	this.state = CLOSED
}

func (this *CircuitBreaker) SetState() {
	if this.failureCount > this.failuerThreshold {
		if (time.Now().Nanosecond() - int(this.lastFailureTime)) > int(this.retryTimePeriod) {
			this.state = HALF_OPEN
		} else {
			this.state = OPEN
		}
	} else {
		this.state = CLOSED
	}
}

func (this *CircuitBreaker) RecordFailure() {
	this.failureCount += 1
	this.lastFailureTime = time.Duration(time.Now().Nanosecond())
}

type In_data struct {
	Ip         string
	Proportion int
}

type pullRecvData struct {
	Ok   bool                       `json:"ok"`
	Msg  string                     `json:"msg"`
	Data [](map[string]([]In_data)) `json:"data"`
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

	fmt.Printf("resave: %s\n", data[:index])

	recvData := pullRecvData{}
	err = json.Unmarshal(data[:index], &recvData)
	if err != nil {
		return err
	}

	tempList := recvData.Data

	for _, ipsMap := range tempList {
		for serverName, ipsList := range ipsMap {
			if len(ipsList) != 0 {
				temp := new([]*CircuitBreaker)
				for _, ip := range ipsList {
					cb := NewCircuitBreaker(ip.Ip, ip.Proportion , 5, time.Minute*1)
					*temp = append(*temp, cb)
				}
				this.mutex.Lock()
				this.serviceMap[serverName] = *temp
				this.mutex.Unlock()
			} else {
				this.mutex.Lock()
				delete(this.serviceMap, serverName)
				this.mutex.Unlock()
			}
		}
	}

	conn.Close()
	return nil
}

//func count(in []*CircuitBreaker, target string) bool {
//	if len(in) == 0 {
//		return false
//	}
//	for _, v := range in {
//		if v.ServiceIp == target {
//			return true
//		}
//	}
//	return false
//}

//负载均衡算法，通过传入服务名称，负载均衡算法计算得出对应的服务器IP地址--多线程环境
func (this *MyRpcClient) load_balance(serviceName string) (*CircuitBreaker, error) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	cbs, ok := this.serviceMap[serviceName];
	if !ok {
		return nil, errors.New("CAN NOT FIND '" + serviceName + "' IN LOCAL SERVER TABLE")
	}
	rand.Seed(time.Now().Unix())
	return cbs[rand.Intn(len(cbs))], nil
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

func (this *MyRpcClient) Init() error {
	return this.pullService()
}

//每隔一段时间去注册中心拉取一下服务
func (this *MyRpcClient) StartPull(timer time.Duration ,callBack func(err error)) {
	ticker := time.NewTicker(timer)
	go func() {
		for _ = range ticker.C {
			callBack(this.pullService())
		}
	}()
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
				serverCB, err := this.load_balance(server)
				if err != nil {
					return []reflect.Value{reflect.ValueOf(err.Error())}
				}
				serverCB.Mutex.Lock()
				defer serverCB.Mutex.Unlock()
				serverCB.SetState()
				switch serverCB.GetState() {
				case OPEN:
					return []reflect.Value{reflect.ValueOf("CIRCUITBREAKER IS OPEN!")}
				case CLOSED, HALF_OPEN:
					errMsg := this.call(server, serverCB.ServiceIp, args)
					if errMsg == "nil" {
						serverCB.ReSet()
					} else {
						serverCB.RecordFailure()
					}
					return []reflect.Value{reflect.ValueOf(errMsg)}
				}
				return []reflect.Value{reflect.ValueOf("nil")}
			})
			val.Field(index).Set(proxyFunc)
		}
	}
}
