//! Mmap-backed I/O (best for warm data and dev/profiling).
use crate::open::open_readonly;
use memmap2::{Mmap, MmapOptions};
use std::fs::File;
use std::io::{Error, ErrorKind, IoSliceMut, Result};

pub struct MmapIo {
    #[allow(dead_code)]
    file: File,
    map: Mmap,
}

impl MmapIo {
    pub fn open(path: &str) -> Result<Self> {
        let file = open_readonly(path, false)?;
        // SAFETY: file is not mutated; map read-only
        let map = unsafe { MmapOptions::new().map(&file)? };
        Ok(Self { file, map })
    }
}

impl crate::Io for MmapIo {
    fn read_at(&self, dst: &mut [u8], offset: u64) -> Result<()> {
        let off = offset as usize;
        let end = off.checked_add(dst.len()).ok_or_else(|| {
            Error::new(ErrorKind::InvalidInput, "offset + len overflow")
        })?;
        if end > self.map.len() {
            return Err(Error::new(ErrorKind::UnexpectedEof, "read past EOF"));
        }
        dst.copy_from_slice(&self.map[off..end]);
        Ok(())
    }

    fn readv_at(&self, iovecs: &mut [IoSliceMut<'_>], mut offset: u64) -> Result<usize> {
        let mut total = 0usize;
        for iov in iovecs.iter_mut() {
            self.read_at(iov, offset)?;
            offset += iov.len() as u64;
            total += iov.len();
        }
        Ok(total)
    }

    fn prefetch(&self, offset: u64, len: usize) {
        #[cfg(target_os = "linux")]
        unsafe {
            // Hint kernel to read-ahead this range
            let start = self.map.as_ptr().add(offset as usize);
            let _ = libc::madvise(start as *mut _, len, libc::MADV_WILLNEED);
        }
    }

    fn alignment(&self) -> usize { 1 }
}
