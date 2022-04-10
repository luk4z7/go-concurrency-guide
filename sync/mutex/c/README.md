```bash
make
```

output:
```bash
cc -std=gnu99 -pedantic -Wall -g   -c -o mutex.o mutex.c
cc -o mutex mutex.o -lpthread && make run
./mutex && make clean
start thread test
hello from thread 0
hello from thread 1
the_counter: 2
hello from thread 2
hello from thread 3
the_counter: 4
the_counter: 4
the_counter: 2
thread 1 done
thread 0 done
thread 3 done
thread 2 done
end thread test

rm -f mutex.o mutex
```