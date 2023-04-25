Here we can explore many cenarious that we can make mistakes when use the sync process to protect our data of data races

### Using reference to loop iterator variable

by design Go transparently access free variables by references in goroutines, that we can take a look in next examples, in this example we can take an data race because the index of the loop is
update by the for/range and is passing the same time by referencing into the goroutine

```go
numbers := []int{1, 2, 3, 4, 5, 6, 7}
for _, i := range numbers {
    go func() {
        fmt.Println(i)
    }()
}

// Output:
// WARNING: DATA RACE
// Read at 0x00c00001a118 by goroutine 24:
//   main.main.func4()
//      main.go:71 +0x3a
```

Go recommends to copy a variable into a iterate to another one, like this

```go
for _, i := range numbers {
    i := i

    go func() {
        fmt.Println(i)
    }()
}

// Output:
// 1
// 3
// 2
// ...
```

Or another approache is using the closures for evaluate each iteration and placed on the stack for the goroutine, so each slice element is available to the goroutine when it is eventually executed.

```go
for _, i := range numbers {
    go func(d int) {
        fmt.Println(i)
    }(i)
}

// Output:
// 1
// 3
// 2
// ...
```

the same is when use the `errogroup` because the function there is not param option

```go
var g errgroup.Group

for i := 0; i < 10; i++ {
    // if we don't made this "i := i" the goroutines is
    // initiated after the for is completed and gets only the last number
    // i := i

    g.Go(func() error {
        return foo(i)
    })
}

// Output:
// WARNING: DATA RACE
// Read at 0x00c0000be038 by goroutine 7:
//  main.main.func1()
//     main.go:34 +0x34
```

To resolve this or using the `i := i` or closure approache

```go
data := []string{"a", "b", "c", "d"}

for _, d := range data {
    func(value string) {
        g.Go(func() error {
            return bar(value)
        })
    }
}

// Output:
// c
// a
// b
// d
```


### References:


[Using reference to loop iterator variable](https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable)

https://eng.uber.com/dynamic-data-race-detection-in-go-code/

https://eng.uber.com/data-race-patterns-in-go/

[Lecture 2: RPC and Threads](https://www.youtube.com/watch?v=gA4YXUJX7t8)
