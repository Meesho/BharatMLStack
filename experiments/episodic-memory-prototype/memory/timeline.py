"""Append-only timeline store backed by SQLite.

One table: timeline_entries.  Entries are written once and never mutated —
no UPDATE, no DELETE, ever.  The timeline is the source-of-truth log from
which episodes, facts, and embeddings are derived.
"""

from __future__ import annotations

import json
import sqlite3
from datetime import datetime, timezone
from typing import Sequence

from memory.models import CONTENT_TEXT, ROLE_USER, TimelineEntry, _uid, _now

_SCHEMA = """\
CREATE TABLE IF NOT EXISTS timeline_entries (
    entry_id          TEXT    PRIMARY KEY,
    sequence_id       INTEGER UNIQUE NOT NULL,
    timestamp         TEXT    NOT NULL,
    role              TEXT    NOT NULL DEFAULT 'user',
    content_type      TEXT    NOT NULL DEFAULT 'text',
    raw_content       TEXT    NOT NULL DEFAULT '',
    semantic_embedding TEXT,
    state_embedding    TEXT,
    agent_id          TEXT    DEFAULT '',
    session_id        TEXT    DEFAULT '',
    state_label       TEXT    DEFAULT '',
    tags              TEXT    DEFAULT '[]'
)"""

_INSERT = """\
INSERT INTO timeline_entries (
    entry_id, sequence_id, timestamp, role, content_type, raw_content,
    semantic_embedding, state_embedding,
    agent_id, session_id, state_label, tags
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"""

_SELECT_ALL = """\
SELECT entry_id, sequence_id, timestamp, role, content_type, raw_content,
       semantic_embedding, state_embedding,
       agent_id, session_id, state_label, tags
FROM timeline_entries ORDER BY sequence_id"""


class Timeline:
    """Ordered, append-only timeline backed by a single SQLite table."""

    def __init__(self, db_path: str = ":memory:") -> None:
        self._conn = sqlite3.connect(db_path, check_same_thread=False)
        self._conn.execute("PRAGMA journal_mode=WAL")
        self._conn.execute(_SCHEMA)
        self._conn.commit()

    # -- internal helpers ------------------------------------------------------

    def _next_seq(self) -> int:
        row = self._conn.execute(
            "SELECT MAX(sequence_id) FROM timeline_entries"
        ).fetchone()
        return 0 if row[0] is None else row[0] + 1

    @staticmethod
    def _encode_embedding(emb: list[float] | None) -> str | None:
        return json.dumps(emb) if emb is not None else None

    @staticmethod
    def _decode_embedding(val: str | None) -> list[float] | None:
        return json.loads(val) if val else None

    def _entry_to_row(self, e: TimelineEntry) -> tuple:
        return (
            e.entry_id,
            e.sequence_id,
            e.timestamp.isoformat(),
            e.role,
            e.content_type,
            e.raw_content,
            self._encode_embedding(e.semantic_embedding),
            self._encode_embedding(e.state_embedding),
            e.agent_id,
            e.session_id,
            e.state_label,
            json.dumps(e.tags),
        )

    def _row_to_entry(self, row: tuple) -> TimelineEntry:
        return TimelineEntry(
            entry_id=row[0],
            sequence_id=row[1],
            timestamp=datetime.fromisoformat(row[2]),
            role=row[3],
            content_type=row[4],
            raw_content=row[5],
            semantic_embedding=self._decode_embedding(row[6]),
            state_embedding=self._decode_embedding(row[7]),
            agent_id=row[8],
            session_id=row[9],
            state_label=row[10],
            tags=json.loads(row[11]) if row[11] else [],
        )

    # -- writes (append-only, never update / delete) ---------------------------

    def append(
        self,
        role: str,
        content: str,
        content_type: str = CONTENT_TEXT,
        state_label: str = "",
        tags: list[str] | None = None,
    ) -> TimelineEntry:
        """Create a TimelineEntry from arguments and persist it."""
        entry = TimelineEntry(
            role=role,
            raw_content=content,
            content_type=content_type,
            state_label=state_label,
            sequence_id=self._next_seq(),
            tags=tags or [],
        )
        self._conn.execute(_INSERT, self._entry_to_row(entry))
        self._conn.commit()
        return entry

    def append_entry(self, entry: TimelineEntry) -> TimelineEntry:
        """Persist a pre-built TimelineEntry, assigning sequence_id."""
        entry.sequence_id = self._next_seq()
        self._conn.execute(_INSERT, self._entry_to_row(entry))
        self._conn.commit()
        return entry

    # -- reads -----------------------------------------------------------------

    def get_range(self, start_seq: int, end_seq: int) -> list[TimelineEntry]:
        """Return entries whose sequence_id is in [start_seq, end_seq]."""
        rows = self._conn.execute(
            "SELECT entry_id, sequence_id, timestamp, role, content_type, "
            "raw_content, semantic_embedding, state_embedding, "
            "agent_id, session_id, state_label, tags "
            "FROM timeline_entries "
            "WHERE sequence_id >= ? AND sequence_id <= ? "
            "ORDER BY sequence_id",
            (start_seq, end_seq),
        ).fetchall()
        return [self._row_to_entry(r) for r in rows]

    def get_recent(self, n: int) -> list[TimelineEntry]:
        """Return the last *n* entries by sequence order."""
        rows = self._conn.execute(
            "SELECT entry_id, sequence_id, timestamp, role, content_type, "
            "raw_content, semantic_embedding, state_embedding, "
            "agent_id, session_id, state_label, tags "
            "FROM timeline_entries ORDER BY sequence_id DESC LIMIT ?",
            (n,),
        ).fetchall()
        return [self._row_to_entry(r) for r in reversed(rows)]

    @property
    def entries(self) -> list[TimelineEntry]:
        """All entries, ordered by sequence_id."""
        rows = self._conn.execute(_SELECT_ALL).fetchall()
        return [self._row_to_entry(r) for r in rows]

    def last(self, n: int = 1) -> list[TimelineEntry]:
        return self.get_recent(n)

    def by_ids(self, ids: Sequence[str]) -> list[TimelineEntry]:
        if not ids:
            return []
        rows = self._conn.execute(
            "SELECT entry_id, sequence_id, timestamp, role, content_type, "
            "raw_content, semantic_embedding, state_embedding, "
            "agent_id, session_id, state_label, tags "
            "FROM timeline_entries "
            "WHERE entry_id IN (SELECT value FROM json_each(?)) "
            "ORDER BY sequence_id",
            (json.dumps(list(ids)),),
        ).fetchall()
        return [self._row_to_entry(r) for r in rows]

    def slice(
        self,
        start: datetime | None = None,
        end: datetime | None = None,
    ) -> list[TimelineEntry]:
        if start and end:
            rows = self._conn.execute(
                "SELECT entry_id, sequence_id, timestamp, role, content_type, "
                "raw_content, semantic_embedding, state_embedding, "
                "agent_id, session_id, state_label, tags "
                "FROM timeline_entries "
                "WHERE timestamp >= ? AND timestamp <= ? "
                "ORDER BY sequence_id",
                (start.isoformat(), end.isoformat()),
            ).fetchall()
        elif start:
            rows = self._conn.execute(
                "SELECT entry_id, sequence_id, timestamp, role, content_type, "
                "raw_content, semantic_embedding, state_embedding, "
                "agent_id, session_id, state_label, tags "
                "FROM timeline_entries WHERE timestamp >= ? "
                "ORDER BY sequence_id",
                (start.isoformat(),),
            ).fetchall()
        elif end:
            rows = self._conn.execute(
                "SELECT entry_id, sequence_id, timestamp, role, content_type, "
                "raw_content, semantic_embedding, state_embedding, "
                "agent_id, session_id, state_label, tags "
                "FROM timeline_entries WHERE timestamp <= ? "
                "ORDER BY sequence_id",
                (end.isoformat(),),
            ).fetchall()
        else:
            return self.entries
        return [self._row_to_entry(r) for r in rows]

    def __len__(self) -> int:
        row = self._conn.execute(
            "SELECT COUNT(*) FROM timeline_entries"
        ).fetchone()
        return row[0]
