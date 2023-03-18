package main

import (
	"fmt"
	"io"
	"net/http"
	"unit.nginx.org/go"
	"os"
	"strings"
)

func handler(w http.ResponseWriter, r *http.Request) {
	args := strings.Join(os.Args[1:], ",")

	w.Header().Add("X-Arg-0", fmt.Sprintf("%v", os.Args[0]))
	w.Header().Add("Content-Length", fmt.Sprintf("%v", len(args)))
	io.WriteString(w, args)
}

func main() {
	http.HandleFunc("/", handler)
	unit.ListenAndServe(":7080", nil)
}

// getArguments returns the command-line arguments passed to the program.
func getArguments() []string {
	return os.Args[1:]
}

// getArgumentAtIndex returns the command-line argument at the given index.
func getArgumentAtIndex(index int) string {
	args := getArguments()

	if index < len(args) {
		return args[index]
	}

	return ""
}

// getNumberOfArguments returns the number of command-line arguments passed to the program.
func getNumberOfArguments() int {
	return len(getArguments())
}