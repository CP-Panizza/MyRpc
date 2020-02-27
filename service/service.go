package main

import (
	"fmt"
		"net"
	"net/rpc"
	)

type User struct {
	Name string
	Age  int
}

type HelleService struct {

}

func (h *HelleService) Hello(req string, resp *User) error {
	fmt.Println("req:", req)
	(*resp).Name = "Hello"
	(*resp).Age = 100
	return nil
}

func (h *HelleService) Say(req string, resp *string) error {
	fmt.Println("req:", req)
	*resp = "return cmj"
	return nil
}


func main() {

	err := rpc.Register(new(HelleService))
	if err != nil {
		panic(err)
	}

	listen, err := net.Listen("tcp", ":8888")
	if err != nil {
		panic(err)
	}

	rpc.Accept(listen)
}
