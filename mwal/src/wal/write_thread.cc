// Leader-follower write thread for group commit in mwal.

#include "wal/write_thread.h"

#include <cassert>

namespace mwal {

WriteThread::WriteThread() = default;

bool WriteThread::JoinBatchGroup(Writer* w) {
  assert(w != nullptr);
  std::unique_lock<std::mutex> lock(mu_);

  if (closed_) {
    w->status = Status::Aborted("WAL is shutting down");
    w->done = true;
    return false;
  }

  pending_.push_back(w);

  if (!leader_active_) {
    leader_active_ = true;
    return true;
  }

  w->cv.wait(lock, [w, this] { return w->done || w->is_leader || closed_; });
  if (closed_ && !w->is_leader) {
    if (!w->done) {
      w->status = Status::Aborted("WAL is shutting down");
      w->done = true;
    }
    return false;
  }
  if (w->is_leader) {
    return true;
  }
  return false;
}

bool WriteThread::JoinAndBuildGroup(Writer* w, WriteGroup* group,
                                    size_t max_group_size) {
  assert(w != nullptr);
  assert(group != nullptr);
  std::unique_lock<std::mutex> lock(mu_);

  if (closed_) {
    w->status = Status::Aborted("WAL is shutting down");
    w->done = true;
    return false;
  }

  pending_.push_back(w);

  if (!leader_active_) {
    leader_active_ = true;

    group->leader = nullptr;
    group->size = 0;
    group->need_sync = false;
    group->last_writer = nullptr;

    size_t cap = (max_group_size > 0) ? max_group_size : pending_.size();
    size_t taken = 0;
    Writer* prev = nullptr;
    while (!pending_.empty() && taken < cap) {
      Writer* cur = pending_.front();
      pending_.pop_front();
      cur->link_newer = nullptr;
      if (prev) prev->link_newer = cur;
      if (cur->sync) group->need_sync = true;
      if (!group->leader) group->leader = cur;
      group->last_writer = cur;
      group->size++;
      prev = cur;
      taken++;
    }
    return true;
  }

  w->cv.wait(lock, [w, this] { return w->done || w->is_leader || closed_; });
  if (closed_ && !w->is_leader) {
    if (!w->done) {
      w->status = Status::Aborted("WAL is shutting down");
      w->done = true;
    }
    return false;
  }
  if (w->is_leader) {
    group->leader = nullptr;
    group->size = 0;
    group->need_sync = false;
    group->last_writer = nullptr;

    size_t cap = (max_group_size > 0) ? max_group_size : pending_.size();
    size_t taken = 0;
    Writer* prev = nullptr;
    while (!pending_.empty() && taken < cap) {
      Writer* cur = pending_.front();
      pending_.pop_front();
      cur->link_newer = nullptr;
      if (prev) prev->link_newer = cur;
      if (cur->sync) group->need_sync = true;
      if (!group->leader) group->leader = cur;
      group->last_writer = cur;
      group->size++;
      prev = cur;
      taken++;
    }
    return true;
  }
  return false;
}

size_t WriteThread::EnterAsBatchGroupLeader(Writer* leader, WriteGroup* group,
                                            size_t max_group_size) {
  assert(leader != nullptr);
  assert(group != nullptr);

  std::lock_guard<std::mutex> lock(mu_);

  group->leader = leader;
  group->size = 0;
  group->need_sync = false;
  group->last_writer = nullptr;

  size_t cap = (max_group_size > 0) ? max_group_size : pending_.size();
  size_t taken = 0;

  Writer* prev = nullptr;
  while (!pending_.empty() && taken < cap) {
    Writer* w = pending_.front();
    pending_.pop_front();

    w->link_newer = nullptr;
    if (prev != nullptr) {
      prev->link_newer = w;
    }
    if (w->sync) {
      group->need_sync = true;
    }
    group->last_writer = w;
    group->size++;
    prev = w;
    taken++;
  }

  return group->size;
}

void WriteThread::CompleteAsyncFollowers(WriteGroup& group, Status status) {
  std::lock_guard<std::mutex> lock(mu_);

  Writer* prev = group.leader;
  Writer* w = group.leader->link_newer;
  while (w != nullptr) {
    Writer* next = w->link_newer;
    if (!w->sync) {
      prev->link_newer = next;
      if (w == group.last_writer) {
        group.last_writer = prev;
      }
      group.size--;
      w->status = status;
      w->done = true;
      w->cv.notify_one();
    } else {
      prev = w;
    }
    w = next;
  }
}

void WriteThread::ExitAsBatchGroupLeader(WriteGroup& group, Status status) {
  std::lock_guard<std::mutex> lock(mu_);

  Writer* w = group.leader->link_newer;
  while (w != nullptr) {
    w->status = status;
    w->done = true;
    w->cv.notify_one();
    w = w->link_newer;
  }

  if (!pending_.empty()) {
    Writer* next = pending_.front();
    next->is_leader = true;
    next->cv.notify_one();
  } else {
    leader_active_ = false;
  }
}

void WriteThread::DrainAndClose() {
  std::lock_guard<std::mutex> lock(mu_);
  closed_ = true;

  for (Writer* w : pending_) {
    w->status = Status::Aborted("WAL is shutting down");
    w->done = true;
    w->cv.notify_one();
  }
  pending_.clear();
  leader_active_ = false;
}

}  // namespace mwal
