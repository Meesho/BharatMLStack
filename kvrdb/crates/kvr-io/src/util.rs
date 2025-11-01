use std::io::{Error, ErrorKind};

pub fn eof() -> Error {
    Error::new(ErrorKind::UnexpectedEof, "unexpected EOF")
}

pub fn io_other(msg: impl Into<String>) -> Error {
    Error::new(ErrorKind::Other, msg.into())
}
