package function

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var isCold = true

// Handle an HTTP Request.
func Handle(w http.ResponseWriter, r *http.Request) {
	/*
	 * YOUR CODE HERE
	 *
	 * Try running `go test`.  Add more test as you code in `handle_test.go`.
	 */

	time.Sleep(1 * time.Second)
	fmt.Fprintf(w, "%s", strconv.FormatBool(isCold))
	isCold = false
}
