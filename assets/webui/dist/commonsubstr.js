function findCommonSubStrs2(a, b) {
  let lastcol = null;
  let best = new Array(a.length);
  for (let i = 0; i < a.length; ++ i) {
    let col = new Array(b.length);
    best[i] = 0;
    for (let j = 0; j < b.length; ++ j) {
      if (a[i] === b[j]) {
        const lu = (i > 0 && j > 0) ? lastcol[j-1] : 0;
        col[j] = lu + 1;
        const off = i - col[j] + 1;
        if (col[j] > best[off]) {
          best[off] = col[j];
        }
      } else {
        col[j] = 0;
      }
    }
    lastcol = col;
  }

  let substrs = [];
  for (let i = 0; i < a.length; ++ i) {
    if (best[i] >= 2) {
      substrs.push(a.substr(i, best[i]));
    }   
  }
  console.log(substrs);
  return substrs;
}

function findCommonSubStrs(ss) {
  if (ss.length < 2)
    return ss;

  const substrs = findCommonSubStrs2(ss[0], ss[1]);
  for (let i = 2; i < ss.length; ++ i) {
    let newsubstrs = [];
    for (let s of substrs) {
      newsubstrs = newsubstrs.concat(findCommonSubStrs2(s, ss[i]));
    }
    substrs = newsubstrs;
  }
  return substrs;
}

function findLongestCommonSubStr(ss) {
  const substrs = findCommonSubStrs(ss);

  let longest = "";
  for (let s of substrs) {
    if (s.length > longest.length)
      longest = s;
  }
  return longest;
}

export {findCommonSubStrs, findLongestCommonSubStr};
