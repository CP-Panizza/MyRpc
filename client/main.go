package main

import (
	"reflect"
	"net/rpc"
	"fmt"
)


//定义传输数据格式
type User struct {
	Name string
	Age  int
}

//定义rpc调用接口，通过tag定义接口中函数对应的远程服务名
type HelleServiceInterface struct {
	Hello func(string,*User) string `service:"HelleService.Hello"`
	Say func(string,*string) string `service:"HelleService.Say"`
}


func main() {
	service := HelleServiceInterface{}
	clinet := MyRpcClient{}
	clinet.Implement(&service)
	u := new(User)
	err := service.Hello("cmj", u)
	if err != "nil" {
		println(err)
	}
	fmt.Println(u)
	var str string
	service.Say("dasfaf", &str)
	fmt.Println(str)

}

//构造MyRpcClient
func NewMyRpcClient() *MyRpcClient{
	clinet := new(MyRpcClient)
	clinet.serviceMap = map[string]*[]string{}
	return clinet
}


type MyRpcClient struct {
	serviceMap map[string]*[]string
}

//注册服务，传入服务名和服务所在服务器ip
func regiestService(serviceName string, serviceIp string)error{
	return nil
}

//负载均衡算法，通过传入服务名称，负载均衡算法计算得出对应的服务器IP地址
func (this *MyRpcClient)load_balance(serviceName string)(string, error){
	return "127.0.0.1:8888", nil
}

//rpc调用远程服务
func (this *MyRpcClient)call(serverName string, serverIp string, args []reflect.Value) string {
	clinet, err := rpc.Dial("tcp",serverIp)
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
func (this *MyRpcClient)Implement(i interface{})  {
	val := reflect.ValueOf(i).Elem()
	typ := reflect.TypeOf(i).Elem()
	for index:=0;index< val.NumField(); index++ {
		funcType := val.Field(index).Type()
		server := typ.Field(index).Tag.Get("service")
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