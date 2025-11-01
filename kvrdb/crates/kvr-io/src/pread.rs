use crate::align::AlignedBuf;
use crate::open::open_readonly;
use crate::util::{eof, io_other};
use std::fs::File;
use std::io::{ErrorKind, IoSliceMut, Result};
use std::os::fd::AsRawFd;

pub struct PreadIo {
    file: File,
    direct: bool,
    align: usize,
}

impl PreadIo {
    /// `use_direct=true` requests O_DIRECT (Linux). Ensure offsets/lengths/buffers align to 4096.
    pub fn open(path: &str, use_direct: bool) -> Result<Self> {
        let file = open_readonly(path, use_direct)?;
        let align = if use_direct { 4096 } else { 1 };
        Ok(Self { file, direct: use_direct, align })
    }

    #[inline] pub fn alignment(&self) -> usize { self.align }

    pub fn alloc_block(&self, len: usize) -> Result<AlignedBuf> {
        if self.direct {
            AlignedBuf::new(len, self.align)
        } else {
            // still page-align for consistency
            AlignedBuf::new(len, 4096)
        }
    }
}

impl crate::Io for PreadIo {
    fn read_at(&self, dst: &mut [u8], offset: u64) -> Result<()> {
        let fd = self.file.as_raw_fd();
        let mut done = 0usize;
        while done < dst.len() {
            let res = unsafe {
                libc::pread(
                    fd,
                    dst[done..].as_mut_ptr() as *mut _,
                    (dst.len() - done) as libc::size_t,
                    (offset + done as u64) as libc::off_t,
                )
            };
            if res == 0 {
                return Err(eof());
            }
            if res < 0 {
                let err = std::io::Error::last_os_error();
                if err.kind() == ErrorKind::Interrupted { continue; }
                return Err(err);
            }
            done += res as usize;
        }
        Ok(())
    }

    fn readv_at(&self, _iovecs: &mut [IoSliceMut<'_>], _offset: u64) -> Result<usize> {
        // If you want Linux-only scatter reads, switch to libc::preadv here with cfg(target_os="linux").
        Err(io_other("readv_at unsupported in PreadIo (use coalescing/batching above Io layer)"))
    }

    fn alignment(&self) -> usize { self.align }
}
