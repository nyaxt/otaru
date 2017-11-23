import './nav.js';
import {$, $$} from './util.js';

let apiprefix = `${window.document.location.origin}/api`;
(() => {
  const apiprefix_input = $("#apiprefix");
  apiprefix_input.value = apiprefix;
  apiprefix_input.addEventListener("change", ev => {
    apiprefix = ev.value;
  });
})();

const rpc = async (endpoint) => {
  const response = await window.fetch(
      apiprefix + endpoint,
      {mode: 'cors', cache: 'reload'});
  if (!response.ok) {
    throw new Error(`fetch failed: ${response.status}`);
  }
  return await response.json();
};

const updateInterval = 3000;
const triggerUpdate = async () => {
  try {
    const result = await rpc("/v1/system/info");

    const fillKeys = ['go_version', 'os', 'arch', 'num_goroutine', 'hostname', 'pid', 'uid', 'mem_alloc', 'mem_sys', 'num_gc', 'num_fds'];
    for (let k of fillKeys) {
      $(`#settings-${k}`).textContent = result[k];
    }
  } catch (e) {
    console.log(e);
  }
  if ($('.section--settings').classList.contains('section--selected'))
    window.setTimeout(triggerUpdate, updateInterval);
}
$('.section--settings').addEventListener('shown', e => {
  triggerUpdate();
});
