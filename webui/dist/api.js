import {$} from './domhelper.js';

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

const fillRemoteContent = async (endpoint, prefix, fillKeys) => {
  const result = await rpc(endpoint);

  for (let k of fillKeys) {
    $(`${prefix}${k}`).textContent = result[k];
  }
};

export {rpc, fillRemoteContent};
