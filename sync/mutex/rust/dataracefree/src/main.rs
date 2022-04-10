use std::thread;

fn main() {
    let mut vec: Vec<i64> = Vec::new();

    // ------- value moved into closure here
    thread::spawn(move || {
        // --- variable moved due to use in closure
        add_vec(&mut vec);
    });

    // vec.push(34)
    // value borrowed here after move
}

fn add_vec(vec: &mut Vec<i64>) {
    vec.push(42);
}
