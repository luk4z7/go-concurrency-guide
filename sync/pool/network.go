package main

import (
    "fmt"
    "log"
    "net"
    "sync"
    "time"
)

func connectToService() interface{} {
    time.Sleep(1 * time.Second)

    return struct{}{}
}

func warmServiceConnCache() *sync.Pool {
    p := &sync.Pool{
        New: connectToService,
    }

    for i := 0; i < 10; i++ {
        p.Put(p.New())
    }

    return p
}

func startNetworkDaemon2() *sync.WaitGroup {
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        connPool := warmServiceConnCache()

        server, err := net.Listen("tcp", "localhost:8081")
        if err != nil {
            log.Fatalf("cannot listen: %v", err)
        }
        defer server.Close()

        wg.Done()

        for {
            conn, err := server.Accept()
            if err != nil {
                log.Printf("cannot accept connection: %v", err)
                continue
            }

            svcConn := connPool.Get()
            fmt.Fprintln(conn, "")
            connPool.Put(svcConn)
            conn.Close()
        }
    }()

    return &wg
}

func startNetworkDaemon1() *sync.WaitGroup {
    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        server, err := net.Listen("tcp", "localhost:8080")
        if err != nil {
            log.Fatalf("cannot listen: %v", err)
        }
        defer server.Close()

        wg.Done()

        for {
            conn, err := server.Accept()
            if err != nil {
                log.Printf("cannot accept connection: %v", err)
                continue
            }

            connectToService()
            fmt.Fprintln(conn, "")
            conn.Close()
        }
    }()

    return &wg
}