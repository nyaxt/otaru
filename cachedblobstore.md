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
  -> Clean or WriteInProgress or Closing
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
