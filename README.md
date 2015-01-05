workqueue-go
============

a sample workqueue for go, similar to workqueue in Linux kernel 
usage
-----
get source:<br>

    go get github.com/Ishukone/workqueue-go
code example：
```go
package main

import (
        "fmt"
        "github.com/Ishukone/workqueue-go"
)

func greeting(work *workqueue.Work) int {
        fmt.Printf("hello, %s\n", work.Data)
}

func main() {
        wq := workqueue.CreateWorkQueue(4)
        var greet string

        for {
                fmt.Scan(&greet)

                work := new(workqueue.Work)
                work.Data = greet
                work.Action = greeting
                wq.ScheduleWork(work)
        }
}
```
build:<br>

    go build
test:<br>
run build result in bash. you can input many names split by space and type Enter.every name will be greeted by work.
