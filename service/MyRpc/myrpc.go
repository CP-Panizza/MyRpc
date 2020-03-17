package MyRpc

import (
	"reflect"
	"net/rpc"
	"net"
	"strconv"
	)

type MyRpc struct {
	infos []string
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



func (this *MyRpc)sendDataToRegistCenter(){

}


func (this *MyRpc)StartServer(port int){
	listen, err := net.Listen("tcp", ":" + strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	rpc.Accept(listen)
}
