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

const requestInteval = 3000;
const triggerUpdate = async () => {
  try {
    const response = await window.fetch(
        apiprefix + "/v1/system/info",
        {mode: 'cors', cache: 'reload'});
    if (!response.ok) {
      throw new Error(`fetch failed: ${response.status}`);
    }
    const json = await response.json();

    const fillKeys = ['go_version', 'os', 'arch', 'num_goroutine', 'hostname', 'pid', 'uid', 'mem_alloc', 'mem_sys', 'num_gc', 'num_fds'];
    for (let k of fillKeys) {
      $(`#settings-${k}`).textContent = json[k];
    }
  } catch (e) {
    console.log(e);
  }
  window.setTimeout(triggerUpdate, requestInteval);
}
triggerUpdate();
