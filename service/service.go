package main

import (
	"fmt"
	"./MyRpc"
)

type User struct {
	Name string
	Age  int
}

type HelleService struct {

}

func (h *HelleService) Hello(req int, resp *int) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	*resp = 10/req;

	return nil
}


func (h *HelleService) Say(req string, resp *string) error {
	fmt.Println("req:", req)
	*resp = "return cmj"
	return nil
}

type MyService struct {

}


func (h *MyService) Do(req string, resp *string) error {
	fmt.Println("req:", req)
	*resp = "return cmj"
	return nil
}


func main() {
	myrpc := MyRpc.NewMyRpc("127.0.0.1")
	myrpc.Register(new(struct{
		HelleService
		MyService
	}))
	myrpc.StartServer(8888)
}
