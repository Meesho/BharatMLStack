//! fr-io: FlashRing I/O backends with a tiny trait surface.
//!
//! Backends:
//! - [`MmapIo`]: great for warm dev/profiling (page cache).
//! - [`PreadIo`]: portable baseline; optional O_DIRECT + aligned buffers.
//! - [`UringIo`]: (feature "uring") placeholder delegating to PreadIo (swap in real uring later).
//!
//! Notes:
//! - Keep block size a multiple of 4 KiB if using O_DIRECT (common FS requirement).
//! - Use `alignment()` to learn backend alignment constraints.

use std::io::{IoSliceMut, Result as IoResult};

pub mod align;
pub mod mmap;
pub mod open;
pub mod pread;
pub mod util;

#[cfg(all(feature = "uring", target_os = "linux"))]
pub mod uring;

pub use mmap::MmapIo;
pub use pread::PreadIo;

#[cfg(all(feature = "uring", target_os = "linux"))]
pub use uring::UringIo;

/// Minimal I/O trait for block-centric reads.
///
/// Implementations should fill the entire `dst` slice or return an error.
/// `readv_at` is optional; callers should handle `Unsupported` gracefully.
pub trait Io: Send + Sync + 'static {
    /// Read exactly dst.len() bytes starting at absolute file offset.
    fn read_at(&self, dst: &mut [u8], offset: u64) -> IoResult<()>;

    /// Optional scatter read-at. If unsupported, return `ErrorKind::Unsupported`.
    fn readv_at(&self, _iovecs: &mut [IoSliceMut<'_>], _offset: u64) -> IoResult<usize> {
        Err(std::io::Error::new(std::io::ErrorKind::Unsupported, "readv_at"))
    }

    /// Optional prefetch hint for a contiguous region. May be a no-op.
    fn prefetch(&self, _offset: u64, _len: usize) {}

    /// Preferred alignment for O_DIRECT (e.g., 4096). `1` if unconstrained.
    fn alignment(&self) -> usize {
        1
    }
}
