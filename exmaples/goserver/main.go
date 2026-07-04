package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello from k8s -v2")
	})
	fmt.Println("server is running")
	http.ListenAndServe(":8080", nil)
}
