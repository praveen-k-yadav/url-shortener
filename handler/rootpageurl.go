package handler

import (
	"fmt"
	"net/http"
)

func RootPageUrlhandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}
