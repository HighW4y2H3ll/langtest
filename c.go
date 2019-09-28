package main

import (
    "fmt"
    "os"
    "bufio"
    "sync"
    "encoding/hex"
    "crypto/md5"
    "crypto/sha1"
    "crypto/sha256"
)

func loadFile(f string) map[string]string {
    fd, _ := os.Open(f)
    defer fd.Close()

    m := make(map[string]string)
    s := bufio.NewScanner(fd)
    for s.Scan() {
        m[s.Text()] = ""
    }
    return m
}

func loadDictFile(algo string, f string) map[string]string {
    var hash func(string) string
    switch algo {
    case "md5":
        hash = func(h string) string {
            x := md5.Sum([]byte(h))
            return hex.EncodeToString(x[:])
        }
        break
    case "sha1":
        hash = func(h string) string {
            x := sha1.Sum([]byte(h))
            return hex.EncodeToString(x[:])
        }
        break
    case "sha256":
        hash = func(h string) string {
            x := sha256.Sum256([]byte(h))
            return hex.EncodeToString(x[:])
        }
        break
    }

    fd, _ := os.Open(f)
    defer fd.Close()

    m := make(map[string]string)
    s := bufio.NewScanner(fd)
    for s.Scan() {
        m[hash(s.Text())] = s.Text()
    }
    return m
}


var THREADS = 2048

type flagty   int
const (
    no_crack flagty = iota
    found
    update
)
type hash_pw struct {
    hash string
    pw  string
    ty  flagty
}

func main() {
    algo := os.Args[1]

    pmap := loadFile(os.Args[2])
    dmap := loadDictFile(algo, os.Args[3])

    cache_update := make(chan hash_pw, 1024)
    defer close(cache_update)
    result_update := make(chan hash_pw, 1024)
    defer close(result_update)

    var wg sync.WaitGroup
    res := make(map[string]string)
    counter := 0
    for k, _ := range pmap {
        counter++
        wg.Add(1)
        go func(w *sync.WaitGroup, h string, r chan hash_pw, q chan hash_pw) {
            defer w.Done()
            if val, ok := dmap[h]; ok {
                r <- hash_pw{h, val, found}
            }
        }(&wg, k, result_update, cache_update)
        if counter == THREADS {
            wg.Wait()
            for {
                select {
                case c, _ := <-result_update:
                    res[c.hash] = c.pw
                //case _, _ := <-cache_update:
                default:
                    goto BEXIT
                }
            }

            BEXIT:
            // Write Partial Results
            fd, _ := os.Create(os.Args[2] + ".result")
            defer fd.Close()
            for k, v := range res {
                fd.WriteString(k + " " + v + "\n")
            }

            counter = 0
        }
    }

    fmt.Println("Done")
}
