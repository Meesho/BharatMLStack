use kvr_io::{Io, PreadIo};
use std::io::Write;

#[test]
fn pread_smoke() {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("data.bin");
    let mut f = std::fs::File::create(&path).unwrap();
    f.write_all(&[0u8; 8192]).unwrap();
    f.write_all(b"hello world").unwrap();

    let io = PreadIo::open(path.to_str().unwrap(), false).unwrap();
    let mut buf = vec![0u8; 11];
    io.read_at(&mut buf, 8192).unwrap();
    assert_eq!(&buf, b"hello world");
}
