import {$} from './domhelper.js';

let apiprefix = `${window.document.location.origin}/`;
(() => {
  const apiprefix_input = $("#apiprefix");
  apiprefix_input.value = apiprefix;
  apiprefix_input.addEventListener("change", ev => {
    apiprefix = ev.value;
  });
})();

const propagateKeys = ['method']
const rpc = async (endpoint, opts = {}) => {
  const url = new URL(endpoint, apiprefix);
  const args = opts['args'] || {};
  for (let k in args) {
    url.searchParams.set(k, args[k]);
  }

  const fetchOpts = {mode: 'cors', cache: 'reload'}
  for (let k of propagateKeys) {
    if (opts[k] !== undefined)
      fetchOpts[k] = opts[k];
  }
  if (opts['body'] !== undefined) {
    const jsonStr = JSON.stringify(opts['body']);
    fetchOpts.body = new Blob([jsonStr], {type: 'application/json'});
  }

  const response = await window.fetch(url, fetchOpts);
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

const downloadFile = (id, filename) => {
  const url = new URL(`file/${id}/${encodeURIComponent(filename)}`, apiprefix);
  window.location = url;
}

export {rpc, fillRemoteContent, downloadFile};
