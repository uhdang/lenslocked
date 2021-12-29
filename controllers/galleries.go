package controllers

import (
	"github.com/uhdang/lenslocked/views"
	"net/http"
)

func NewGalleries() *Galleries {
	return &Galleries{
		NewView: views.NewView("bootstrap", "galleries/new"),
	}
}

type Galleries struct {
	NewView *views.View
}

func (g *Galleries) New(w http.ResponseWriter, r *http.Request) {
	if err := g.NewView.Render(w, nil); err != nil {
		panic(err)
	}
}
