const formatBlobSize = val => {
  if (val === undefined) {
    return '0B';
  } else if (val > 1024 * 1024 * 1024) {
    return (val / (1024 * 1024 * 1024)).toPrecision(2) + 'GiB';
  } else if (val > 1024 * 1024) {
    return (val / (1024 * 1024)).toPrecision(2) + 'MiB';
  } else if (val > 1024) {
    return (val / 1024).toPrecision(2) + 'KiB';
  } else {
    return val + 'B';
  }
}

const formatTimestamp = (t, opts={}) => {
  const diff = new Date() - t;

  const pad = n => (n < 10 ? '0' : '') + n;
  if (opts.relative !== false) {
    const startOfToday = new Date();
    startOfToday.setHours(0);
    startOfToday.setMinutes(0);
    startOfToday.setSeconds(0);
    startOfToday.setMilliseconds(0);

    if (diff < 60 * 1000) {
      return `${(diff / (1000)).toFixed(0)}s ago`;
    } else if (diff < 1 * 60 * 60 * 1000) {
      return `${(diff / (60 * 1000)).toFixed(0)}m ago`;
    } else if (diff < 6 * 60 * 60 * 1000) {
      return `${(diff / (60 * 60 * 1000)).toFixed(0)}h ago`;
    } else if (t > startOfToday) {
      return `${pad(t.getHours())}:${pad(t.getMinutes())}`;
    }
  }
  const ymd = `${pad(t.getFullYear()-2000)}/${pad(t.getMonth()+1)}/${pad(t.getDate())}`;
  if (!opts.full) {
    return ymd; 
  }
  
  return `${ymd} ${pad(t.getHours())}:${pad(t.getMinutes())}:${pad(t.getSeconds())}`
}

const formatTimestampRPC = n => {
  if (n < 0)
    return "-";
  else
    return formatTimestamp(new Date(n*1000));
};

export {formatBlobSize, formatTimestamp, formatTimestampRPC};
