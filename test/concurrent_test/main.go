package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	n := 5
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			rsp, err := http.Get("http://localhost:9999/api?key=Tom")
			if err == nil {
				bytes, err := io.ReadAll(rsp.Body)
				if err == nil {
					fmt.Println(string(bytes))
				}
			}
		}()
	}
	wg.Wait()
}
