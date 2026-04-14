package main

import (
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    switch os.Args[1] {
    case "run":
        fmt.Println("run not implemented")
    default:
        printUsage()
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Println("usage: agent-eval run")
}
