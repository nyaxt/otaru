import {$} from './domhelper.js';
import {formatBlobSize} from './format.js';
import {previewFile, previewFileUrl} from './api.js';

const kCursorClass = 'content__entry--cursor';

const innerHTMLSource =
  `<div class="preview__header">
    <div class="preview__title"></div>
    <div class="button preview__close">✘</div>
  </div>
  <div class="preview__content">
  </div>
  <div class="preview__imgcont hidden">
    <img class="preview__img" src="//:0">
  </div>
  `;

const innerHTMLTable = `<table class="content__table"><tbody></tbody></table>`;

class OtaruPreview extends HTMLElement {
  constructor() {
    super();

    this.cursorIndex_ = -1;
    this.tbody_ = null;
    this.cursorRow_ = null;
  }

  connectedCallback() {
    this.innerHTML = innerHTMLSource;
    this.classList.add('preview');
    this.classList.add('hidden');

    this.closeBtn_ = this.querySelector('.preview__close');
    this.titleDiv_ = this.querySelector('.preview__title');
    this.contentDiv_ = this.querySelector('.preview__content');
    this.imgcontDiv_ = this.querySelector('.preview__imgcont');
    this.img_ = this.querySelector('.preview__img');

    this.closeBtn_.addEventListener('click', () => {
      this.close();
    });
  }

  get isOpen() {
    return !this.classList.contains('hidden');
  }

  async open(opath) {
    this.opath_ = opath;
    this.classList.remove('hidden');
    this.contentDiv_.innerText = "loading";
    this.titleDiv_.innerText = opath;

    const resp = await previewFile(opath);
    if (Array.isArray(resp)) {
      this.contentDiv_.innerHTML = innerHTMLTable;
      const tbody = this.contentDiv_.querySelector('tbody');
      this.tbody_ = tbody;

      resp.forEach((e, idx) => {
        const tr = document.createElement('tr');
        tr.classList.add('content__entry');
        tr.classList.add('preview__entry');
        tr.data = e;

        const tdName = document.createElement('td');
        tdName.classList.add('content__cell');
        tdName.classList.add('preview__cell--name');
        tdName.innerText = e['name'];
        tr.appendChild(tdName);

        const tdSize = document.createElement('td');
        tdSize.classList.add('content__cell');
        tdSize.classList.add('preview__cell--size');
        tdSize.innerText = formatBlobSize(e['size']);
        tr.appendChild(tdSize);

        tr.addEventListener('click', ev => {
          this.cursorIndex = idx;
        });

        tbody.appendChild(tr);
      });

      this.cursorIndex = 0;
    } else {
      let pre = document.createElement('pre');
      pre.textContent = resp;

      this.contentDiv_.innerHTML = '';
      this.contentDiv_.appendChild(pre);
      this.tbody_ = null;
      this.cursorRow_ = null;
      this.clearCursor();
    }
  }

  close() {
    this.hideImgCont_();
    this.classList.add('hidden');
  }

  get cursorIndex() {
    return this.cursorIndex_;
  }

  set cursorIndex(val) {
    if (this.cursorIndex_ == val)
      return;

    this.cursorIndex_ = val;
    this.updateCursor();
  }

  clearCursor() {
    this.cursorIndex = -1;
  }

  updateCursor() {
    const rows = this.tbody_.querySelectorAll('tr');
    for (let tr of rows) {
      tr.classList.remove(kCursorClass);
    }

    if (this.cursorIndex_ < 0)
      return;

    if (this.cursorIndex_ >= rows.length) {
      this.cursorIndex_ = rows.length - 1;
    } else if (this.cursorIndex < 0) {
      this.cursorRow_ = null;
      return;
    }

    if (this.imgcontVisible) {
      this.img_.src = previewFileUrl(this.opath_, this.cursorIndex_);
    }

    let cr = rows[this.cursorIndex_];
    this.cursorRow_ = cr;
    if (cr) {
      cr.classList.add(kCursorClass);

      let crRect = cr.getBoundingClientRect();
      let divRect = this.contentDiv_.getBoundingClientRect();

      if (crRect.bottom > divRect.bottom) {
        cr.scrollIntoView({block: 'end'});
      } else if (crRect.top < divRect.top) {
        cr.scrollIntoView({block: 'start'});
      }
    }
  }

  hideImgCont_() {
    this.contentDiv_.classList.remove('hidden');
    this.imgcontDiv_.classList.add('hidden');
    this.img_.src = '//:0';
  }

  get imgcontVisible() {
    return !this.imgcontDiv_.classList.contains('hidden');
  }

  toggleView() {
    if (this.imgcontVisible) {
      this.hideImgCont_();
    } else if (this.tbody_) {
      this.contentDiv_.classList.add('hidden');
      this.imgcontDiv_.classList.remove('hidden');
    }
    this.updateCursor();
  }
};

window.customElements.define('otaru-preview', OtaruPreview);

const preview = $('#preview');

export {preview};
