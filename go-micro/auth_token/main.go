package main

import (
	"context"
	"fmt"

	"go-micro.dev/v4"
)

type Greeter struct{}

type HelloRequest struct {
}

type HelloResponse struct {
}

func (g *Greeter) Hello(ctx context.Context, req *HelloRequest, rsp *HelloResponse) error {
	fmt.Println("Hello service was called")
	// Business logic goes here...
	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("hello"),
		micro.WrapHandler(AuthMiddleware()),
	)

	service.Init()

	if err := micro.RegisterHandler(service.Server(), new(Greeter)); err != nil {
		fmt.Println(err)
		return
	}

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
