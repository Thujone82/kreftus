use std::path::Path;
use std::process::Command;

fn main() {
    let exe = if Path::new(".\\bin\\win\\x64\\sudokusolve.exe").exists() {
        ".\\bin\\win\\x64\\sudokusolve.exe"
    } else if Path::new(".\\bin\\win\\x86\\sudokusolve.exe").exists() {
        ".\\bin\\win\\x86\\sudokusolve.exe"
    } else {
        ".\\sudokusolve.exe"
    };
    let args: Vec<String> = std::env::args().skip(1).collect();
    let mut cmd = Command::new(exe);
    cmd.args(&args);
    let status = cmd.status().expect("Failed to execute sudokusolve.exe");
    std::process::exit(status.code().unwrap_or(0));
}