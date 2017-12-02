import {contentSection, isSectionSelected} from './nav.js';
import {fillRemoteContent} from './api.js';
import './browsefs.js';

const updateInterval = 3000;

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
