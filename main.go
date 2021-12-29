package main

import (
	"github.com/gorilla/mux"
	"github.com/uhdang/lenslocked/controllers"
	"net/http"
)

func main() {
	staticC := controllers.NewStatic()
	usersC := controllers.NewUsers()
	galleriesC := controllers.NewGalleries()

	r := mux.NewRouter()
	r.Handle("/", staticC.Home).Methods("GET")
	r.Handle("/contact", staticC.Contact).Methods("GET")
	r.Handle("/faq", staticC.Faq).Methods("GET")
	r.HandleFunc("/signup", usersC.New).Methods("GET")
	r.HandleFunc("/signup", usersC.Create).Methods("POST")
	r.HandleFunc("/galleries/new", galleriesC.New).Methods("GET")

	http.ListenAndServe(":3000", r)
}
