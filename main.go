package main

func main() {
    a := New()

    panic(a.r.Run(":8080"))
}
