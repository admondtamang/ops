package main

import (
	"fmt"
	"net/http"
)

func main() {
	version := 4
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello from k8s - ", version)
	})
	fmt.Println("server is running -", version)
	http.ListenAndServe(":8080", nil)
}
