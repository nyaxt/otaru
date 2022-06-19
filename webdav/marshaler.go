package webdav

import (
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/nyaxt/otaru/cli"
	"go.uber.org/zap"
)

type Marshaler interface {
	WriteResponse(w http.ResponseWriter, basepath string, entry *Entry, listing Listing)
}

type PropStatMarshaler struct{}

type collection struct{}
type davresp struct {
	XMLName      xml.Name    `xml:"D:response"`
	Href         string      `xml:"D:href"`
	DispName     string      `xml:"D:propstat>D:prop>D:displayname"`
	CreationDate string      `xml:"D:propstat>D:prop>D:creationdate"`
	ContentType  string      `xml:"D:propstat>D:prop>D:getcontenttype"`
	Size         int64       `xml:"D:propstat>D:prop>D:getcontentlength"`
	Collection   *collection `xml:"D:propstat>D:prop>D:resourcetype>D:collection,omitempty"`
	LastModified string      `xml:"D:propstat>D:prop>D:getlastmodified"`
	Status       string      `xml:"D:propstat>D:status"`
}

func entryToResp(basepath string, entry *Entry, toplevel bool) *davresp {
	href := basepath
	if !toplevel {
		href = filepath.Join(href, url.PathEscape(entry.Name))
	}

	utc := entry.ModifiedTime.UTC()
	r := &davresp{
		Href:         href,
		DispName:     entry.Name,
		CreationDate: utc.Format(time.RFC3339),
		Size:         entry.Size,
		LastModified: utc.Format(time.RFC1123),
		Status:       StatusOk,
	}
	if entry.IsDir {
		r.Collection = &collection{}
	} else {
		r.ContentType = mime.TypeByExtension(filepath.Ext(entry.Name))
	}

	return r
}

type multistatus struct {
	XMLName xml.Name `xml:"D:multistatus"`
	D       string   `xml:"xmlns:D,attr"`
	Rs      []*davresp
}

const StatusOk = "HTTP/1.1 200 OK"

func (m PropStatMarshaler) WriteResponse(w http.ResponseWriter, basepath string, entry *Entry, listing Listing) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		zap.S().Infof("Error while writing XML header: %v", err)
		return
	}

	enc := xml.NewEncoder(w)

	rs := make([]*davresp, 0, len(listing)+1)
	rs = append(rs, entryToResp(basepath, entry, true))
	for _, e := range listing {
		rs = append(rs, entryToResp(basepath, e, false))
	}

	ms := multistatus{D: "DAV:", Rs: rs}
	if err := enc.Encode(ms); err != nil {
		zap.S().Infof("Error while writing XML: %v", err)
	}
}

type HtmlMarshaler struct{}

func (m HtmlMarshaler) WriteResponse(w http.ResponseWriter, basepath string, entry *Entry, listing Listing) {
	// FIXME: implement html marshal
	(PropStatMarshaler{}).WriteResponse(w, basepath, entry, listing)
}

type ContentServerMarshaler struct {
	OrigReq *http.Request
	CInfo   *cli.ConnectionInfo
}

func copyHeader(d, s http.Header) {
	for k, v := range s {
		d[k] = v
	}
}

func (m ContentServerMarshaler) WriteResponse(w http.ResponseWriter, basepath string, entry *Entry, listing Listing) {
	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: m.CInfo.TLSConfig,
		},
	}
	url := &url.URL{
		Scheme: "https",
		Host:   m.CInfo.ApiEndpoint,
		Path:   fmt.Sprintf("/file/%d/bin", entry.Id),
	}
	zap.S().Debugf("requrl %v", url.String())
	req := &http.Request{
		Method: "GET",
		Header: map[string][]string{},
		URL:    url,
	}
	req.Header.Add("Range", m.OrigReq.Header.Get("Range"))

	resp, err := cli.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to issue HTTP GET request to backend: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		http.Error(w, fmt.Sprintf("Backend respond with error: %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	copyHeader(w.Header(), resp.Header)
	io.Copy(w, resp.Body)
}
