package main

import (
	"fmt"
	"github.com/jiaxwu/gcache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	gcache.NewGroup("scores", 2<<10, gcache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s does not exist", key)
	}))
	addr := "localhost:9999"
	peers := gcache.NewHTTPPool(addr)
	log.Printf("gcache is running at %s \n", addr)
	log.Fatalln(http.ListenAndServe(addr, peers))
}
