const $ = document.querySelector.bind(document);
const $$ = document.querySelectorAll.bind(document);
const removeAllChildNodes = par => {
  while(par.hasChildNodes())
    par.removeChild(par.lastChild);
};

export {$, $$, removeAllChildNodes};
