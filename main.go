package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/pyrox777/NaW/handlers"
)

const version = "0.4.0"

func main() {
	serverAddr := flag.String("server", "127.0.0.1:8081", "server ip and port")
	flag.Parse()

	http.HandleFunc("/", handlers.Root)
	http.HandleFunc("/view/", handlers.Make(handlers.View))
	http.HandleFunc("/edit/", handlers.Make(handlers.Edit))
	http.HandleFunc("/save/", handlers.Make(handlers.Save))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	log.Println("NaW (Not another Wiki) v" + version + " starts server on " + *serverAddr)
	log.Fatal(http.ListenAndServe(*serverAddr, nil))
}
