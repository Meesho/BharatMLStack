//! Page-aligned buffer for O_DIRECT reads.
//!
//! Provides `AlignedBuf` which owns a page-aligned allocation via `posix_memalign`.

use std::io::{Error, ErrorKind, Result};
use std::mem::ManuallyDrop;
use std::ops::{Deref, DerefMut};
use std::ptr::NonNull;
use std::sync::Mutex;

pub struct AlignedBuf {
    ptr: NonNull<u8>,
    len: usize,
    align: usize,
}

unsafe impl Send for AlignedBuf {}
unsafe impl Sync for AlignedBuf {}

impl AlignedBuf {
    /// Allocate a page-aligned buffer (alignment must be power-of-two, multiple of page size).
    pub fn new(len: usize, align: usize) -> Result<Self> {
        if !align.is_power_of_two() {
            return Err(Error::new(ErrorKind::InvalidInput, "align must be power-of-two"));
        }
        let mut p: *mut std::ffi::c_void = std::ptr::null_mut();
        let rc = unsafe { libc::posix_memalign(&mut p, align, len) };
        if rc != 0 || p.is_null() {
            return Err(Error::new(
                ErrorKind::Other,
                format!("posix_memalign failed rc={rc}"),
            ));
        }
        // SAFETY: posix_memalign returns unique, suitably aligned block
        let ptr = unsafe { NonNull::new_unchecked(p as *mut u8) };
        Ok(Self { ptr, len, align })
    }

    #[inline]
    pub fn as_mut_slice(&mut self) -> &mut [u8] {
        // SAFETY: we own len bytes
        unsafe { std::slice::from_raw_parts_mut(self.ptr.as_ptr(), self.len) }
    }

    #[inline]
    pub fn as_slice(&self) -> &[u8] {
        // SAFETY: we own len bytes
        unsafe { std::slice::from_raw_parts(self.ptr.as_ptr(), self.len) }
    }

    pub fn len(&self) -> usize { self.len }
    pub fn alignment(&self) -> usize { self.align }
}

impl Drop for AlignedBuf {
    fn drop(&mut self) {
        unsafe { libc::free(self.ptr.as_ptr() as *mut _) }
    }
}

impl Deref for AlignedBuf {
    type Target = [u8];
    fn deref(&self) -> &Self::Target { self.as_slice() }
}
impl DerefMut for AlignedBuf {
    fn deref_mut(&mut self) -> &mut Self::Target { self.as_mut_slice() }
}

/// Convert an `AlignedBuf` into a `Box<[u8]>` by copying (if you need heap-owned copies).
pub fn to_boxed_slice(buf: &AlignedBuf) -> Box<[u8]> {
    let mut v = Vec::with_capacity(buf.len());
    v.extend_from_slice(buf.as_slice());
    v.into_boxed_slice()
}


pub struct BufPool {
    align: usize,
    size: usize,
    free: Mutex<Vec<AlignedBuf>>,
}
impl BufPool {
    pub fn new(n: usize, size: usize, align: usize) -> Self {
        let mut v = Vec::with_capacity(n);
        for _ in 0..n { v.push(AlignedBuf::new(size, align).unwrap()); }
        Self { align, size, free: Mutex::new(v) }
    }
    pub fn get(&self) -> AlignedBuf {
        self.free.lock().unwrap().pop().unwrap_or_else(|| AlignedBuf::new(self.size, self.align).unwrap())
    }
    pub fn put(&self, mut b: AlignedBuf) {
        // optional: zero or leave dirty
        self.free.lock().unwrap().push(b);
    }
}
