use std::fs::File;
use std::io::{Error, ErrorKind, Result};
use std::os::fd::{FromRawFd, OwnedFd};

#[cfg(unix)]
fn flags_readonly(use_direct: bool) -> libc::c_int {
    let mut flags = libc::O_RDONLY | libc::O_CLOEXEC;

    // Avoid following symlinks (best-effort across Unix)
    #[cfg(any(target_os = "linux", target_os = "android", target_os = "macos", target_os = "freebsd", target_os = "netbsd", target_os = "openbsd"))]
    {
        flags |= libc::O_NOFOLLOW;
    }

    // Linux-only niceties
    #[cfg(target_os = "linux")]
    {
        // Reduce atime updates (ignored if not permitted)
        flags |= libc::O_NOATIME;
        if use_direct {
            flags |= libc::O_DIRECT;
        }
    }

    flags
}

/// Open a file read-only with safe flags.
/// `use_direct` (Linux-only) requests O_DIRECT; ignored elsewhere.
pub fn open_readonly(path: &str, use_direct: bool) -> Result<File> {
    #[cfg(unix)]
    {
        use std::ffi::CString;
        let cpath = CString::new(path).map_err(|_| Error::new(ErrorKind::InvalidInput, "path has NUL"))?;
        let flags = flags_readonly(use_direct);

        // SAFETY: libc::open is called with a valid C string and flags; mode is ignored for O_RDONLY
        let fd = unsafe { libc::open(cpath.as_ptr(), flags, 0) };
        if fd < 0 {
            return Err(Error::last_os_error());
        }
        // SAFETY: fd is uniquely owned; wrap in OwnedFd then File
        let owned = unsafe { OwnedFd::from_raw_fd(fd) };
        return Ok(File::from(owned));
    }

    #[cfg(not(unix))]
    {
        if use_direct {
            return Err(Error::new(ErrorKind::Unsupported, "O_DIRECT not supported on this platform"));
        }
        std::fs::File::open(path)
    }
}
