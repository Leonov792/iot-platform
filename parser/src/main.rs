use std::io::{self, Read, Write};
use std::time::{SystemTime, UNIX_EPOCH};

const MAGIC: [u8; 2] = [0xAB, 0xCD];
const DEVICE_ID_LEN: usize = 16;
const KIND_TELEMETRY: u8 = 0x01;

// формат кадра (бинарный):
//   [magic:2][ver:1][device_id:16][kind:1][payload_len:2 BE][payload:N][crc:1]
// crc — тупой xor по всем байтам от ver до конца payload. надёжность так себе,
// но для демо хватает. TODO: crc32, когда будет время
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
                // одна строка json на кадр — эликсиру удобно читать по строчке
                let _ = writeln!(w, "{json}");
                let _ = w.flush();
            }
            Err(e) => eprintln!("parser: {e}"),
        }
    }
}

enum ReadErr {
    Eof,
    Bad(String),
}

// читает ровно один кадр со stdin. границы кадра известны из заголовка, так что
// внешний length-prefix не нужен
fn read_frame<R: Read>(r: &mut R) -> Result<Vec<u8>, ReadErr> {
    let mut magic = [0u8; 2];
    match r.read_exact(&mut magic) {
        Ok(_) => {}
        // чистый eof на первом же чтении — поток закрыли, выходим
        Err(ref e) if e.kind() == io::ErrorKind::UnexpectedEof => return Err(ReadErr::Eof),
        Err(e) => return Err(ReadErr::Bad(format!("магик: {e}"))),
    }
    if magic != MAGIC {
        return Err(ReadErr::Bad("магик не сошёлся".into()));
    }

    // версия(1) + device_id(16) + kind(1) + payload_len(2)
    let mut header = [0u8; 1 + DEVICE_ID_LEN + 1 + 2];
    if let Err(e) = r.read_exact(&mut header) {
        return Err(ReadErr::Bad(format!("заголовок: {e}")));
    }
    let payload_len =
        u16::from_be_bytes([header[1 + DEVICE_ID_LEN + 1], header[1 + DEVICE_ID_LEN + 2]]) as usize;

    let mut payload = vec![0u8; payload_len];
    if let Err(e) = r.read_exact(&mut payload) {
        return Err(ReadErr::Bad(format!("payload: {e}")));
    }

    let mut crc = [0u8; 1];
    if let Err(e) = r.read_exact(&mut crc) {
        return Err(ReadErr::Bad(format!("crc: {e}")));
    }

    let mut frame = Vec::with_capacity(2 + header.len() + payload.len() + 1);
    frame.extend_from_slice(&magic);
    frame.extend_from_slice(&header);
    frame.extend_from_slice(&payload);
    frame.extend_from_slice(&crc);

    Ok(frame)
}

fn parse_frame(frame: &[u8]) -> Result<String, String> {
    const HDR_END: usize = 2 + 1 + DEVICE_ID_LEN + 1 + 2; // всё до payload
    if frame.len() < HDR_END + 1 {
        return Err("кадр короче заголовка".into());
    }
    if frame[0..2] != MAGIC {
        return Err("магик не сошёлся".into());
    }

    let ver = frame[2];
    if ver != 1 {
        return Err(format!("неизвестная версия протокола {ver}"));
    }

    let id_start = 3;
    let id_end = id_start + DEVICE_ID_LEN;
    let device_id = String::from_utf8_lossy(&frame[id_start..id_end])
        .trim_end_matches('\0')
        .trim()
        .to_string();
    if device_id.is_empty() {
        return Err("пустой device_id".into());
    }

    let kind = frame[id_end];
    let payload_len = u16::from_be_bytes([frame[id_end + 1], frame[id_end + 2]]) as usize;
    let payload_start = id_end + 3;
    let payload_end = payload_start + payload_len;
    if frame.len() < payload_end + 1 {
        return Err("payload не влез в кадр".into());
    }

    let payload = &frame[payload_start..payload_end];
    let crc = frame[payload_end];

    let mut sum = 0u8;
    for &b in &frame[2..payload_end] {
        sum ^= b;
    }
    if sum != crc {
        return Err(format!("crc не сошёлся: ждём {sum:#x}, пришло {crc:#x}"));
    }

    match kind {
        KIND_TELEMETRY => parse_telemetry(&device_id, payload),
        _ => Err(format!("неизвестный kind {kind:#x}")),
    }
}

fn parse_telemetry(device_id: &str, payload: &[u8]) -> Result<String, String> {
    // temp: f32 LE, humidity: f32 LE, battery: u8
    if payload.len() < 9 {
        return Err("короткий payload телеметрии".into());
    }
    let temp = f32::from_le_bytes(payload[0..4].try_into().unwrap());
    let humidity = f32::from_le_bytes(payload[4..8].try_into().unwrap());
    let battery = payload[8];

    // времени в пакете нет, ставим время получения
    let ts = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis())
        .unwrap_or(0);

    // форматим json руками, чтобы не тащить serde ради одной строчки
    Ok(format!(
        r#"{{"device_id":"{device_id}","ts":{ts},"temp":{temp},"humidity":{humidity},"battery":{battery}}}"#
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn build_frame(device_id: &str, temp: f32, humidity: f32, battery: u8) -> Vec<u8> {
        let mut id = [0u8; DEVICE_ID_LEN];
        let b = device_id.as_bytes();
        id[..b.len()].copy_from_slice(b);

        let mut payload = Vec::new();
        payload.extend_from_slice(&temp.to_le_bytes());
        payload.extend_from_slice(&humidity.to_le_bytes());
        payload.push(battery);

        let mut f = Vec::new();
        f.extend_from_slice(&MAGIC);
        f.push(1); // ver
        f.extend_from_slice(&id);
        f.push(KIND_TELEMETRY);
        f.extend_from_slice(&(payload.len() as u16).to_be_bytes());
        f.extend_from_slice(&payload);

        let mut sum = 0u8;
        for &b in &f[2..] {
            sum ^= b;
        }
        f.push(sum);

        f
    }

    #[test]
    fn ok_telemetry() {
        let f = build_frame("sensor-1", 21.5, 40.0, 87);
        let s = parse_frame(&f).expect("кадр должен парситься");
        assert!(s.contains("\"device_id\":\"sensor-1\""));
        assert!(s.contains("\"temp\":21.5"));
        assert!(s.contains("\"battery\":87"));
    }

    #[test]
    fn bad_crc() {
        let mut f = build_frame("sensor-1", 21.5, 40.0, 87);
        let n = f.len();
        f[n - 1] ^= 0xFF; // портим последний байт (crc)
        assert!(parse_frame(&f).is_err());
    }

    #[test]
    fn short_frame() {
        assert!(parse_frame(&[0xAB, 0xCD]).is_err());
    }

    #[test]
    fn unknown_kind() {
        let mut f = build_frame("sensor-1", 21.5, 40.0, 87);
        // меняем kind на что-то незнакомое и пересчитываем crc
        f[3 + DEVICE_ID_LEN] = 0x7F;
        let n = f.len();
        let mut sum = 0u8;
        for &b in &f[2..n - 1] {
            sum ^= b;
        }
        f[n - 1] = sum;
        assert!(parse_frame(&f).is_err());
    }

    fn frame_with(id: &str, kind: u8, payload: &[u8]) -> Vec<u8> {
        let mut idb = [0u8; DEVICE_ID_LEN];
        let b = id.as_bytes();
        idb[..b.len()].copy_from_slice(b);

        let mut f = Vec::new();
        f.extend_from_slice(&MAGIC);
        f.push(1);
        f.extend_from_slice(&idb);
        f.push(kind);
        f.extend_from_slice(&(payload.len() as u16).to_be_bytes());
        f.extend_from_slice(payload);

        let mut sum = 0u8;
        for &b in &f[2..] {
            sum ^= b;
        }
        f.push(sum);
        f
    }

    #[test]
    fn zero_payload_telemetry() {
        let f = frame_with("sensor-1", KIND_TELEMETRY, &[]);
        assert!(parse_frame(&f).is_err());
    }

    #[test]
    fn empty_device_id() {
        let f = frame_with("", KIND_TELEMETRY, &[0u8; 9]);
        assert!(parse_frame(&f).is_err());
    }
}
