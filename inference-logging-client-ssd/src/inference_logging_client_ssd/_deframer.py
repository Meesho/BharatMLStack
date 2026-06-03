"""Binary frame parsing for asyncloguploader .log files.

On-disk format written by asyncloguploader (Go):

  File = Frame* (frames are contiguous, each capacity bytes in total)

  Frame layout (capacity bytes):
    [0:4]  capacity       uint32 LE  — total frame size in bytes (incl. this header)
    [4:8]  validDataBytes uint32 LE  — valid bytes in the data section
    [8:]   data section   bytes      — capacity - 8 bytes; only first validDataBytes are used

  Record layout (within data section, back-to-back, no inter-record padding):
    [0:4]  length    uint32 LE  — total record payload size = TIMESTAMP_SIZE + len(proto)
    [4:12] timestamp uint64 LE  — Unix nanoseconds
    [12:]  proto      bytes      — serialised MPLog protobuf (length - 8 bytes)

Files are written with Direct I/O so the data section may be zero-padded to a
4 KiB boundary; the validDataBytes field marks where real records end.
"""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Generator

logger = logging.getLogger(__name__)

_FRAME_HEADER_SIZE = 8   # capacity (4B) + validDataBytes (4B)
_RECORD_LEN_SIZE = 4     # length prefix per record
_TIMESTAMP_SIZE = 8      # UnixNano uint64 LE
_MAX_FRAME_SIZE = 500 * 1024 * 1024  # 500 MB sanity cap
_RECOVERY_SCAN_LIMIT = 10 * 1024 * 1024  # 10 MB forward scan on corrupt frame
_RECOVERY_ALIGNMENT = 4096  # Direct I/O alignment used by the uploader


def deframe(path: str | Path) -> list[tuple[int, bytes]]:
    """Parse a .log file and return ``(timestamp_ns, proto_bytes)`` pairs.

    Corrupt or empty frames are skipped with a WARNING log; parse errors
    within individual records are skipped with a DEBUG log.

    Args:
        path: Path to the asyncloguploader ``.log`` file.

    Returns:
        List of ``(timestamp_ns, proto_bytes)`` tuples, one per record.
    """
    path = Path(path)
    file_size = path.stat().st_size
    records: list[tuple[int, bytes]] = []
    frame_num = 0

    with open(path, "rb") as f:
        while f.tell() < file_size:
            frame_start = f.tell()
            header = f.read(_FRAME_HEADER_SIZE)

            if len(header) < _FRAME_HEADER_SIZE:
                break  # trailing bytes < 8 — end of file

            capacity = int.from_bytes(header[0:4], "little")
            valid_data_bytes = int.from_bytes(header[4:8], "little")
            frame_num += 1

            if not _valid_frame_header(capacity, valid_data_bytes, file_size - frame_start):
                logger.warning(
                    "Frame %d at offset %d: invalid header "
                    "(capacity=%d, validDataBytes=%d) — scanning for next frame",
                    frame_num, frame_start, capacity, valid_data_bytes,
                )
                next_pos = _find_next_frame(f, frame_start, file_size)
                if next_pos is None:
                    logger.warning(
                        "Could not recover after corrupt frame %d — stopping deframe",
                        frame_num,
                    )
                    break
                f.seek(next_pos)
                continue

            if valid_data_bytes == 0:
                logger.debug("Frame %d: empty (capacity=%d) — skipping", frame_num, capacity)
                f.seek(frame_start + capacity)
                continue

            # Read the data section (capacity already accounts for the 8-byte header)
            data = f.read(capacity - _FRAME_HEADER_SIZE)
            if len(data) < valid_data_bytes:
                logger.warning(
                    "Frame %d: truncated read (got %d bytes, need %d) — skipping",
                    frame_num, len(data), valid_data_bytes,
                )
                # f is at EOF; nothing more to read
                break

            before = len(records)
            for record in _extract_records(data[:valid_data_bytes], frame_num):
                records.append(record)

            logger.debug(
                "Frame %d: +%d records (running total %d)",
                frame_num, len(records) - before, len(records),
            )

            # Advance to the start of the next frame
            f.seek(frame_start + capacity)

    logger.info("Deframed %s — %d frames, %d records", path.name, frame_num, len(records))
    return records


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------

def _valid_frame_header(capacity: int, valid_data_bytes: int, remaining: int) -> bool:
    """Return True if the header values are self-consistent and fit in the file."""
    data_section = capacity - _FRAME_HEADER_SIZE
    return (
        capacity >= _FRAME_HEADER_SIZE
        and capacity <= _MAX_FRAME_SIZE
        and data_section >= 0
        and valid_data_bytes <= data_section
        and capacity <= remaining
    )


def _find_next_frame(f, corrupt_start: int, file_size: int) -> int | None:
    """Scan forward at 4 KiB-aligned offsets looking for a valid frame header.

    Returns the file offset of the next valid frame, or None if not found.
    """
    limit = min(corrupt_start + _RECOVERY_SCAN_LIMIT, file_size)
    for offset in range(_RECOVERY_ALIGNMENT, limit - corrupt_start, _RECOVERY_ALIGNMENT):
        pos = corrupt_start + offset
        f.seek(pos)
        header = f.read(_FRAME_HEADER_SIZE)
        if len(header) < _FRAME_HEADER_SIZE:
            break
        capacity = int.from_bytes(header[0:4], "little")
        valid_data = int.from_bytes(header[4:8], "little")
        if _valid_frame_header(capacity, valid_data, file_size - pos):
            logger.debug("Recovery: found valid frame at offset %d (+%d)", pos, offset)
            return pos
    return None


def _extract_records(
    data: bytes, frame_num: int
) -> Generator[tuple[int, bytes], None, None]:
    """Yield ``(timestamp_ns, proto_bytes)`` from a frame's valid data block."""
    offset = 0
    length = len(data)

    while offset < length:
        # Need at least 4 bytes for the length prefix
        if offset + _RECORD_LEN_SIZE > length:
            break

        record_len = int.from_bytes(data[offset : offset + _RECORD_LEN_SIZE], "little")
        offset += _RECORD_LEN_SIZE

        if record_len == 0:
            # Zero-length record is used as padding; continue scanning
            continue

        if offset + record_len > length:
            logger.debug(
                "Frame %d: record at byte %d claims %d bytes but only %d remain — stopping",
                frame_num, offset, record_len, length - offset,
            )
            break

        if record_len < _TIMESTAMP_SIZE:
            logger.debug(
                "Frame %d: record too short to hold timestamp (%d bytes) — skipping",
                frame_num, record_len,
            )
            offset += record_len
            continue

        timestamp_ns = int.from_bytes(data[offset : offset + _TIMESTAMP_SIZE], "little")
        proto_bytes = data[offset + _TIMESTAMP_SIZE : offset + record_len]
        yield timestamp_ns, proto_bytes

        offset += record_len
