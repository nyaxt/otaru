import {$} from './domhelper.js';

let apiprefix = `${window.document.location.origin}/`;
(() => {
  const apiprefix_input = $("#apiprefix");
  apiprefix_input.value = apiprefix;
  apiprefix_input.addEventListener("change", ev => {
    apiprefix = ev.value;
  });
})();

const propagateKeys = ['method'];
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
  if (opts['rawBody'] !== undefined) {
    fetchOpts.body = opts['rawBody']; 
  }

  const response = await window.fetch(url, fetchOpts);
  if (!response.ok) {
    throw new Error(`fetch failed: ${response.status}`);
  }
  return await response.json();
};

const fillRemoteContent = async (endpoint, prefix, fillKeys) => {
  const result = await rpc(endpoint);

  if (Array.isArray(fillKeys)) {
    for (let k of fillKeys) {
      $(`${prefix}${k}`).textContent = result[k] || 0;
    }
  } else {
    for (let k in fillKeys) {
      const t = fillKeys[k] || (a => a);
      $(`${prefix}${k}`).textContent = t(result[k] || 0);
    }
  }
};

const fsopEndpoint = (op, host) => {
  if (host === '[noproxy]')
    return `api/v1/filesystem/${op}`;
  else if (host === '[local]')
    return `api/v1/fe/local/${op}`;

  return `proxy/${host}/api/v1/filesystem/${op}`;
};

const reOtaruPath = /^\/\/([\w\[\]]+)(\/.*)$/
const parseOtaruPath = (opath) => {
  const m = opath.match(reOtaruPath);
  if (!m) {
    throw new Error(`Invalid otaru path: ${opath}`)
  }
  const host = m[1];
  const path = m[2];
  return {host, path};
}

const fsLs = async (host, path) => {
  return await rpc(fsopEndpoint('ls', host), {args: {path: path}});
};

const fsMv = async (src, dest) => {
  const {host: hostSrc, path: pathSrc} = parseOtaruPath(src);
  const {host: hostDest, path: pathDest} = parseOtaruPath(dest);

  if (hostSrc != hostDest) {
    throw new Error(`Unimplemented: src host ${hostSrc} != dest host ${hostDest}`);
  }

  return await rpc (fsopEndpoint('mv', hostSrc), {
    method: 'POST',
    body: {pathSrc: pathSrc, pathDest: pathDest},
  });
}

const downloadFile = (host, id, filename) => {
  var ep;
  if (host === '[noproxy]') {
    ep = `file/${id}/${encodeURIComponent(filename)}`;
  } else if (host === '[local]') {
    throw "attempt to download local file";
  } else {
    ep = `proxy/${host}/file/${id}/${encodeURIComponent(filename)}`;
  }

  const url = new URL(ep, apiprefix);
  window.location = url;
}

export {rpc, fillRemoteContent, downloadFile, parseOtaruPath, fsLs, fsMv};
