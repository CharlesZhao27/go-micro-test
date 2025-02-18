package main

import (
	"context"
	"fmt"

	"go-micro.dev/v4"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *interface{}, rsp *interface{}) error {
	fmt.Println("Service A was called")
	return nil
}

func main() {

	// New service
	service := micro.NewService(micro.Name("serviceA"))

	service.Init()

	micro.RegisterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
