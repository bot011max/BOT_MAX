package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    fmt.Println("Медицинский бот запускается...")
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Привет! Я медицинский бот. Версия 1.0.0")
    })
    
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"status": "ok", "version": "1.0.0"}`)
    })
    
    log.Println("Сервер запущен на порту 8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}