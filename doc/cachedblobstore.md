# ChunkIO and CachedBlobStore

- ChunkedFileIO.PWrite()
-- cio := NewChunkIO()
--- bh := cbs.Open()
-- cio.PWrite()
--- bh.PWrite()
-- cio.Close() -> cio.Sync()
--- bh.PWrite() (updateHeader)

# State
- Uninitialized
  -> Invalidating or Clean or Errored
- Invalidating
  -> Clean or Errored
- Errored
  -> ErroredClosed
- ErroredClosed
  -> ∅
- Clean
  -> WriteInProgress or Closing
- WriteInProgress
  -> Dirty
- Dirty
  -> Clean or WriteInProgress or DirtyClosing
- DirtyClosing
  -> Closed
- Closing
  -> Closed
- Closed
  -> ∅

writeBackWithLock is called from:
- Sync
- Close

and we have to ensure that:
- forbid new write during writeBack, or at least
- know that new write occured during the writeBack

Which is better?
- allow new write
  - simpler impl
  - less blocking
- forbid new write during writeback
  - ???

Q: Should we really sync during chunk write? Good chance where chunk may become unreadable.
A: Yes
- Supporting transactional IO in cachedblobstore makes it even more complicated.
- ChooseSyncEntry would do best to avoid the situation.
- We should make ChunkedIO tolerant to bad blocks anyway.
