use std::thread;
use std::time::Duration;

fn main() {
    let mut vec: Vec<i64> = Vec::new();

    println!("value {:?}: ", vec);
    // error -> closure may outlive the current function, but it borrows `vec`, which is owned by the current function
    // to force the closure to take ownership of `vec` (and any other referenced variables), use the `move` keyword
    let handle = thread::spawn(move || {
        add_vec(&mut vec);
        println!("value {:?}: ", vec);

        thread::sleep(Duration::from_millis(5));
    });

    handle.join().unwrap()
}

fn add_vec(vec: &mut Vec<i64>) {
    vec.push(42);
}
