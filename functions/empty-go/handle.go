package function

import (
	"fmt"
	"net/http"
	"strconv"
)

var isCold = true

func Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", strconv.FormatBool(isCold))
	isCold = false
}
