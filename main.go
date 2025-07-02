package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/", serveHTML)
	http.ListenAndServe(":8080", nil)
}

func serveHTML(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "index.html")
}
