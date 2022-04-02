package main

import (
    "io/ioutil"
    "net"
    "testing"
)

func init() {
    daemonStarted1 := startNetworkDaemon1()
    daemonStarted1.Wait()

    daemonStarted2 := startNetworkDaemon2()
    daemonStarted2.Wait()
}

func BenchmarkNetworkRequest1(b *testing.B) {
    for i := 0; i < b.N; i++ {
        conn, err := net.Dial("tcp", "localhost:8080")
        if err != nil {
            b.Fatalf("cannot dial host: %v", err)
        }

        if _, err := ioutil.ReadAll(conn); err != nil {
            b.Fatalf("cannot read: %v", err)
        }

        conn.Close()
    }
}

func BenchmarkNetworkRequest2(b *testing.B) {
    for i := 0; i < b.N; i++ {
        conn, err := net.Dial("tcp", "localhost:8081")
        if err != nil {
            b.Fatalf("cannot dial host: %v", err)
        }

        if _, err := ioutil.ReadAll(conn); err != nil {
            b.Fatalf("cannot read: %v", err)
        }

        conn.Close()
    }
}