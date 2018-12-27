import {$} from './domhelper.js';

const kHiddenClass = 'hidden';
const kInnerHTMLSource =
  `
  <div class="section__header infobar">
    <div class="infobar__text">hogefuga dayo-</div>
    <div class="infobar__button button">✗</div>
  </div>
  `;

// TODO: multi-line support
class Infobar extends HTMLElement {
  constructor() {
    super();
  }

  connectedCallback() {
    this.hide();
    this.innerHTML = kInnerHTMLSource;

    this.textDiv_ = this.querySelector('.infobar__text');
    this.closeBtn_ = this.querySelector('.infobar__button');
    this.closeBtn_.addEventListener('click', _ => {
      this.hide();
    });
  }

  showMessage(msg) {
    this.textDiv_.innerText = msg;
    this.classList.remove(kHiddenClass);
  }

  hide() {
    this.classList.add(kHiddenClass);
  }
};
window.customElements.define('otaru-infobar', Infobar);

const infobar = $('otaru-infobar');
export {infobar};
