import {contentSection, isSectionSelected} from './nav.js';
import {fillRemoteContent} from './api.js';
import {formatTimestampRPC} from './format.js';
import './browsefs.js';
import './loglevel.js';

const updateInterval = 3000;

(() => {
  const triggerUpdate = async () => {
    if (!isSectionSelected('inodedbstats'))
      return;

    try {
      await fillRemoteContent('api/v1/inodedb/stats', '#inodedbstats-', {
          'last_sync': formatTimestampRPC,
          'last_tx': formatTimestampRPC,
          'last_id': null,
          'version': null,
          'last_ticket': null,
          'number_of_node_locks': null,
      });
    } catch (e) {
      console.log(e);
    }
    window.setTimeout(triggerUpdate, updateInterval);
  }
  contentSection('inodedbstats').addEventListener('shown', e => {
    triggerUpdate();
  });
})();

(() => {
  const triggerUpdate = async () => {
    if (!isSectionSelected('blobstore'))
      return;

    try {
      await fillRemoteContent('api/v1/blobstore/config', '#blobstore-', [
          'backend_impl_name', 'backend_flags',
          'cache_impl_name', 'cache_flags']);
    } catch (e) {
      console.log(e);
    }
    window.setTimeout(triggerUpdate, updateInterval);
  }
  contentSection('blobstore').addEventListener('shown', e => {
    triggerUpdate();
  });
})();

(() => {
  const triggerUpdate = async () => {
    if (!isSectionSelected('settings'))
      return;

    try {
      await fillRemoteContent("api/v1/system/info", "#settings-", [
          'go_version', 'os', 'arch', 'num_goroutine', 'hostname', 'pid', 'uid',
          'mem_alloc', 'mem_sys', 'num_gc', 'num_fds']);
    } catch (e) {
      console.log(e);
    }
    window.setTimeout(triggerUpdate, updateInterval);
  }
  contentSection('settings').addEventListener('shown', e => {
    triggerUpdate();
  });
})();
