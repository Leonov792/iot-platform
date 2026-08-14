use std::io::{self, Write};

use gateway_parser::{parse_frame, read_frame, ReadErr};

// standalone-бинарь: читает бинарные кадры со stdin, печатает json в stdout.
// оставлен для обратной совместимости (эмулятор, офлайн-демо, ручной запуск).
fn main() {
    let stdin = io::stdin();
    let mut r = stdin.lock();
    let stdout = io::stdout();
    let mut w = stdout.lock();

    loop {
        let frame = match read_frame(&mut r) {
            Ok(f) => f,
            Err(ReadErr::Eof) => break,
            Err(ReadErr::Bad(e)) => {
                eprintln!("parser: {e}");
                // не падаем целиком — вдруг следующий кадр норм
                continue;
            }
        };

        match parse_frame(&frame) {
            Ok(json) => {
                // одна строка json на кадр — удобно читать по строчке
                let _ = writeln!(w, "{json}");
                let _ = w.flush();
            }
            Err(e) => eprintln!("parser: {e}"),
        }
    }
}
