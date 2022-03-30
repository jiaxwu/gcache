package main

import (
	"context"
	"fmt"
	"github.com/jiaxwu/gcache/registry"
	"log"
)

func main() {
	n, err := registry.New("gcache/", []string{"49.233.30.197:2379"})
	if err != nil {
		log.Fatalln(err)
	}
	watch := n.Watch(context.Background())
	fmt.Println(n.GetAddrs(context.Background()))
	if err := n.Register(context.Background(), "localhost:8081"); err != nil {
		log.Fatalln(err)
	}
	for event := range watch {
		fmt.Println(event)
	}
}
