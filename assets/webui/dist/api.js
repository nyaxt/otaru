import {$} from './domhelper.js';

let apiprefix = `${window.document.location.origin}/`;
(() => {
  const apiprefix_input = $("#apiprefix");
  apiprefix_input.value = apiprefix;
  apiprefix_input.addEventListener("change", ev => {
    apiprefix = ev.value;
  });
})();

const rpcUrl = (endpoint, opts = {}) => {
  const url = new URL(endpoint, apiprefix);

  const args = opts['args'] || {};
  for (let k in args) {
    url.searchParams.set(k, args[k]);
  }

  return url;
};

const propagateKeys = ['method'];
const rpc = async (endpoint, opts = {}) => {
  const url = rpcUrl(endpoint, opts);

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

  const resp = await window.fetch(url, fetchOpts);
  if (!resp.ok) {
    throw new Error(`fetch failed: ${resp.status}`);
  }
  const ctype = resp.headers.get('Content-Type');
  if (ctype === "application/json") {
    return await resp.json();
  } else if (ctype === "text/plain") {
    return await resp.text();
  } else {
    throw new Error(`fetch resp unknown ctype "${ctype}"`);
  }
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
  const result = await rpc(fsopEndpoint('ls', host), {args: {path}});
  if (result['entry'])
    return result['entry'];

  if (result['listing'])
    return result['listing'][0]['entry'];

  return [];
};

const fsMkdir = async (opath) => {
  const {host, path} = parseOtaruPath(opath);
  return await rpc(fsopEndpoint('mkdir', host), {
    method: 'POST',
    body: {path}
  });
};

const fsRm = async (opath) => {
  const {host, path} = parseOtaruPath(opath);
  return await rpc(fsopEndpoint('rm', host), {
    method: 'POST',
    body: {path}
  });
};

const fsMv = async (src, dest) => {
  const {host: hostSrc, path: pathSrc} = parseOtaruPath(src);
  const {host: hostDest, path: pathDest} = parseOtaruPath(dest);

  if (hostSrc != hostDest) {
    throw new Error(`Unimplemented: src host ${hostSrc} != dest host ${hostDest}`);
  }

  console.log(`mv "${pathSrc}" -> "${pathDest}"`);
  return await rpc (fsopEndpoint('mv', hostSrc), {
    method: 'POST',
    body: {pathSrc, pathDest},
  });
};

const previewFileUrl = (opath, idx) => {
  return rpcUrl('preview', {args: {opath: opath, i: idx}});
};

const previewFile = async (opath) => {
  return await rpc('preview', {args: {opath}});
};

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

export {
  downloadFile,
  fillRemoteContent,
  fsLs,
  fsMkdir,
  fsMv,
  fsRm,
  parseOtaruPath,
  previewFile,
  previewFileUrl,
  rpc,
};
