use std::process::Command;

fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let mut cmd = Command::new(".\\sudoku.exe");
    cmd.args(&args);
    let status = cmd.status().expect("Failed to execute sudoku.exe");
    std::process::exit(status.code().unwrap_or(0));
}
