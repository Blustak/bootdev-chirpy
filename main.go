package main

import (
    "net/http"
)

func main() {
    serve := http.NewServeMux()
    server := http.Server{
        Handler: serve,
        Addr: ":8080",
    }

    server.ListenAndServe()
}
