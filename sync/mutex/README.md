## Mutex

[Home](https://github.com/luk4z7/go-concurrency-guide)<br/>
[Sync](https://github.com/luk4z7/go-concurrency-guide/tree/main/sync)

Here we can take some of the many examples of how to make `memory access synchonization` safely to avoid data race errors and race conditions.

In Go we can use the `sync` package to protect critical sections like the example [complete code](https://github.com/luk4z7/go-concurrency-guide/blob/main/sync/mutex/main.go)

```go
var count int
var lock sync.Mutex

func main() {
    increment := func() {
        lock.Lock()
        defer lock.Unlock()
        count++
        fmt.Printf("Incrementing: %d\n", count)
    }
```


Another view from other languages

Few languages tell us this at compile time like `rust` which prevents us from accessing the same variable on multiple threads, this code causes an error because at compile time a data race is detected, it blocked us from writing to the same variable simultaneously, [mutex code](https://github.com/luk4z7/go-concurrency-guide/blob/main/sync/mutex/rust/mutex/src/main.rs)

This example return the error, [complete code](https://github.com/luk4z7/go-concurrency-guide/blob/main/sync/mutex/rust/dataracefree/src/main.rs)
```rust
// error -> closure may outlive the current function, but it borrows `vec`, which is owned by the current function
fn main() {
    let mut vec: Vec<i64> = Vec::new();

    thread::spawn(|| {
        add_vec(&mut vec);
    });

    vec.push(34)
}

fn add_vec(vec: &mut Vec<i64>) {
    vec.push(42);
}
```

C Example using C11 `threads.h` implementation over POSIX threads from [c11threads](https://github.com/jtsiomb/c11threads)
```c++
int increment_the_counter() {
    int r = mtx_lock(&mtx);
    if (r != 0) return r;

    // With the mutex locked we can safely poke the counter.
    the_counter++;

    return mtx_unlock(&mtx);
}

int read_the_counter(int *value) {
    int r = mtx_lock(&mtx);
    if (r != 0) return r;

    // With the mutex locked we can safely read the counter.
    *value = the_counter;

    return mtx_unlock(&mtx);
}
```

[complete c code](https://github.com/luk4z7/go-concurrency-guide/tree/main/sync/mutex/c)
