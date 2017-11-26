import {contentSection, isSectionSelected} from './nav.js';
import {rpc, fillRemoteContent} from './api.js'; 
import {$, $$} from './util.js';

const updateInterval = 3000;

(() => {
  const triggerUpdate = async () => {
    if (!isSectionSelected('browsefs'))
      return;

    try {
      const result = await rpc("/v1/filesystem/ls?path=/");
      console.dir(result);
      for (let e of result.entry) {
        console.dir(e);
      }
    } catch (e) {
      console.log(e);
    }
  }
  contentSection('browsefs').addEventListener('shown', e => {
    triggerUpdate();
  });
  contentSection('browsefs').addEventListener('hidden', e => {
    // $('.browsefs__entry').removeChild();
  });
})();

(() => {
  const triggerUpdate = async () => {
    if (!isSectionSelected('blobstore'))
      return;

    try {
      await fillRemoteContent("/v1/blobstore/config", "#blobstore-", [
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
      await fillRemoteContent("/v1/system/info", "#settings-", [
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
