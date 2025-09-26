package main

import (
    "net/http"
)


func main() {
    serve := http.NewServeMux()
    serve.HandleFunc("/healthz",readinessHandler)
    fileServeHandle := http.StripPrefix("/app",http.FileServer(http.Dir(".")))
    serve.Handle("/app/",fileServeHandle)
    serve.Handle("/assets", http.FileServer(http.Dir("./assets")))
    server := http.Server{
        Handler: serve,
        Addr: ":8080",
    }

    server.ListenAndServe()
}

func readinessHandler(resW http.ResponseWriter,req *http.Request) {
    resW.Header().Add("Content-Type","text/plain; charset=utf-8")
    resW.WriteHeader(200)
    resW.Write([]byte("OK"))
}

