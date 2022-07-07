package main

import (
    "fmt"
    "os"
)

func main() {
    a := New()

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080" // Default port if not specified
    }

    panic(a.r.Run(fmt.Sprintf(":%s", port)))
}
