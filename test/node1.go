package main

import (
   "github.com/vtphan/disq"
   // "fmt"
)

func main() {
   node := disq.NewNode("127.0.0.1:5000")
   node.Run("")
}