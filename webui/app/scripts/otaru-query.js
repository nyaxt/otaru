"use strict";

class OtaruQuery {
  constructor(opts) {
    if (opts.endpointURL === undefined) { throw "endpointURL required!"; }
    if (opts.onData === undefined) { throw "onData required!"; }

    this.endpointURL = opts.endpointURL;
    this.onData = opts.onData;
    this.onError = opts.onError || this._defaultOnError;

    this.shouldFetch = false; 
    this.state = 'inactive';
    this.timer = null;
  }

  start() {
    this.shouldFetch = true; 
    this._requestIfNeeded();
  }

  stop() {
    this.shouldFetch = false;
  }

  _waitAndRequestIfNeeded() {
    if (this.shouldFetch) {
      this.state = 'wait';
      this.timer = setTimeout(() => this._requestIfNeeded(), 3000);
    }
  }

  _requestIfNeeded() {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    if (this.shouldFetch) {
      fetch(this.endpointURL)
        .catch(this._onError.bind(this))
        .then(this._onResponse.bind(this));
      this.state = 'inflight';
    } else {
      this.state = 'inactive';
    }
  }

  _onResponse(res) {
    if (res === undefined) return;

    res.text().then(function(resText) {
      this.onData(resText);
      this._waitAndRequestIfNeeded();
    }.bind(this));
  }

  _onError(err) {
    this.onError(err);
    this._waitAndRequestIfNeeded();
  }

  _defaultOnError(err) {
    // do nothing
  }
}
