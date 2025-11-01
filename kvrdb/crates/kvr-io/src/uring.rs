//! io_uring backend (Linux only).
//! Currently delegates to PreadIo for correctness; replace internals with real uring when ready.
#![cfg(all(feature = "uring", target_os = "linux"))]

use crate::pread::PreadIo;
use std::io::{IoSliceMut, Result};

pub struct UringIo {
    inner: PreadIo,
}

impl UringIo {
    pub fn open(path: &str, use_direct: bool, _queue_depth: u32) -> Result<Self> {
        // TODO: replace with real io_uring queue + registered buffers
        let inner = PreadIo::open(path, use_direct)?;
        Ok(Self { inner })
    }
}

impl crate::Io for UringIo {
    fn read_at(&self, dst: &mut [u8], offset: u64) -> Result<()> {
        self.inner.read_at(dst, offset)
    }
    fn readv_at(&self, iovecs: &mut [IoSliceMut<'_>], offset: u64) -> Result<usize> {
        self.inner.readv_at(iovecs, offset)
    }
    fn alignment(&self) -> usize { self.inner.alignment() }
}
