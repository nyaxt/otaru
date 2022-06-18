import {$} from './domhelper.js';
import {infobar} from './infobar.js';

const kHostLocal = '[local]';
const kHostNoProxy = '[noproxy]';
const kCopy = 'copy';
const kMove = 'move';

const apiprefix = `${window.document.location.origin}/`;

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

  const fetchOpts = {mode: 'cors', cache: 'reload'};
  const headers = new Headers();
  for (let k of propagateKeys) {
    if (opts[k] === undefined)
      continue;

    if (k === 'headers') {
      const pheaders = opts[k];
      for (let hk of pheaders) {
        headers.appemd(hk, pheaders.get(hk));
      }
      continue;
    }

    fetchOpts[k] = opts[k];
  }
  fetchOpts['headers'] = headers;
  if (opts['body'] !== undefined) {
    const jsonStr = JSON.stringify(opts['body']);
    fetchOpts.body = new Blob([jsonStr], {type: 'application/json'});
  }
  if (opts['rawBody'] !== undefined) {
    fetchOpts.body = opts['rawBody'];
  }

  const resp = await window.fetch(url, fetchOpts);
  if (!resp.ok) {
    if (resp.status === 401) {
      infobar.showMessage(`Failed to authenticate with the API server.`);
      throw new Error(`fetch failed: Unauthorized.`);
    } else if (resp.status === 403) {
      infobar.showMessage(`Request forbidden.`);
      throw new Error(`fetch failed: Forbidden.`);
    } else {
      throw new Error(`fetch failed: ${resp.status}.`);
    }
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
  if (host === kHostNoProxy)
    return `api/v1/filesystem/${op}`;
  else if (host === kHostLocal)
    return `api/v1/fe/local/${op}`;

  return `apigw/${host}/api/v1/filesystem/${op}`;
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
    return result['listing'][0]['entry'] || [];

  return [];
};

const fsMkdir = async (opath) => {
  const {host, path} = parseOtaruPath(opath);
  if (host === kHostLocal) {
    return await rpc('api/v1/fe/local/mkdir', {
      method: 'POST',
      body: {path}
    });
  }

  return await rpc(fsopEndpoint('node', host), {
    method: 'POST',
    body: {
      dir_id: "0",
      name: path,
      uid: 0,
      gid: 0,
      perm_mode: 0o755,
      modified_time: "0",
      type: "DIR",
    }
  });
};

const fsRm = async (opath) => {
  const {host, path} = parseOtaruPath(opath);
  if (host === kHostLocal) {
    return await rpc('api/v1/fe/local/rm', {
      method: 'POST',
      body: {path}
    });
  }

  return await rpc(fsopEndpoint('node/rm', host), {
    method: 'POST',
    body: {
      dir_id: "0",
      name: path,
    }
  });
};

const fsMoveOrCopy = async (moveOrCopy, src, dest) => {
  const {host: hostSrc, path: pathSrc} = parseOtaruPath(src);
  const {host: hostDest, path: pathDest} = parseOtaruPath(dest);

  // local  -> local: /api/v1/fe/local/mv
  // local  -> remote: /api/v1/fe/local/upload
  // remote -> local: /api/v1/fe/local/download
  // remote -> remote: /api/v1/fe/remove_mv

  console.log(`mv "${pathSrc}" -> "${pathDest}"`);

  if (hostSrc === kHostLocal) {
    if (hostDest === kHostLocal) {
      const op = moveOrCopy === kCopy ? 'cp' : 'mv';
      return await rpc(`api/v1/fe/local/${op}`, {
        method: 'POST',
        body: {pathSrc, pathDest},
      });
    }

    await rpc ('api/v1/fe/local/upload', {
      method: 'POST',
      body: {pathSrc, opathDest: dest, allowOverwrite: false},
    });
    if (moveOrCopy === kMove) {
      await fsRm(src);
    }
  } else {
    if (hostDest === kHostLocal) {
      return await rpc('api/v1/fe/local/download', {
        method: 'POST',
        body: {opathSrc: src, pathDest, allowOverwrite: false},
      });
      if (moveOrCopy === kMove) {
        await fsRm(src);
      }
    }

    if (hostSrc === hostDest) {
      if (moveOrCopy === kCopy) {
        throw "not implemented";
      } else {
        return await rpc(fsopEndpoint('node/rename', hostSrc), {
          method: 'POST',
          body: {pathSrc, pathDest},
        });
      }
    }

    const op = moveOrCopy === kCopy ? 'remote_cp' : 'remote_mv';
    return await rpc(`api/v1/fe/local/${op}`, {
      method: 'POST',
      body: {pathSrc, pathDest},
    });
  }
};

const previewFileUrl = (opath, idx) => {
  return rpcUrl('preview', {args: {opath: opath, i: idx}});
};

const previewFile = async (opath) => {
  return await rpc('preview', {args: {opath}});
};

const downloadFile = (host, id, filename) => {
  var ep;
  if (host === kHostNoProxy) {
    ep = `file/${id}/${encodeURIComponent(filename)}`;
  } else if (host === kHostLocal) {
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
  fsMoveOrCopy,
  fsRm,
  kCopy,
  kHostLocal,
  kHostNoProxy,
  kMove,
  parseOtaruPath,
  previewFile,
  previewFileUrl,
  rpc,
};
