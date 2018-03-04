package webdav_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bobziuchkovski/digest"

	tu "github.com/nyaxt/otaru/testutils"
	. "github.com/nyaxt/otaru/webdav"
)

func init() { tu.EnsureLogger() }

const testListenAddr = "localhost:20800"
const username = "username"
const password = "password"

func TestServe_Basic(t *testing.T) {
	fs := tu.TestFileSystem()
	if err := fs.WriteFile("/foo.txt", tu.HelloWorld, 0644); err != nil {
		t.Errorf("WriteFile: %v", err)
	}

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		err := Serve(
			FileSystem(fs),
			ListenAddr(testListenAddr),
			CloseChannel(closeC),
		)
		if err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		joinC <- struct{}{}
	}()

	resp, err := http.Get("http://" + testListenAddr + "/foo.txt")
	if err != nil {
		t.Errorf("http.Get: %v", err)
		return
	}
	cont, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ReadAll(http.Get resp.Body): %v", err)
		return
	}
	if !bytes.Equal(cont, tu.HelloWorld) {
		t.Errorf("unexpected content: %v != exp %v", cont, tu.HelloWorld)
	}
	resp.Body.Close()

	close(closeC)
	<-joinC

	// hack: ensure listener socket close before proceeding
	time.Sleep(100 * time.Millisecond)
}

func TestServe_TLS(t *testing.T) {
	otarudir := os.Getenv("OTARUDIR")
	certFile := path.Join(otarudir, "cert.pem")
	keyFile := path.Join(otarudir, "cert-key.pem")

	fs := tu.TestFileSystem()
	if err := fs.WriteFile("/foo.txt", tu.HelloWorld, 0644); err != nil {
		t.Errorf("WriteFile: %v", err)
	}

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		err := Serve(
			FileSystem(fs),
			X509KeyPair(certFile, keyFile),
			ListenAddr(testListenAddr),
			CloseChannel(closeC),
		)
		if err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		joinC <- struct{}{}
	}()

	c, err := tu.TLSHTTPClient(certFile)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	resp, err := c.Get("https://" + testListenAddr + "/foo.txt")
	if err != nil {
		t.Errorf("http.Get: %v", err)
		return
	}
	cont, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ReadAll(http.Get resp.Body): %v", err)
		return
	}
	if !bytes.Equal(cont, tu.HelloWorld) {
		t.Errorf("unexpected content: %v != exp %v", cont, tu.HelloWorld)
	}
	resp.Body.Close()

	close(closeC)
	<-joinC

	// hack: ensure listener socket close before proceeding
	time.Sleep(100 * time.Millisecond)
}

func TestServe_Htdigest(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "htdigest")
	if err != nil {
		t.Errorf("TempFile: %v", err)
		return
	}

	htdigestFilePath := tmpfile.Name()
	defer os.Remove(htdigestFilePath)

	htdigest := "username:otaru webdav:0a61aad0dd78551b25c72fa6ad68a7dc\n"
	if _, err := tmpfile.Write([]byte(htdigest)); err != nil {
		t.Errorf("TempFile write: %v", err)
		return
	}
	if err := tmpfile.Close(); err != nil {
		t.Errorf("TempFile close: %v", err)
		return
	}

	fs := tu.TestFileSystem()
	if err := fs.WriteFile("/foo.txt", tu.HelloWorld, 0644); err != nil {
		t.Errorf("WriteFile: %v", err)
	}

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		err := Serve(
			FileSystem(fs),
			ListenAddr(testListenAddr),
			DigestAuth("otaru webdav", htdigestFilePath),
			CloseChannel(closeC),
		)
		if err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		joinC <- struct{}{}
	}()

	dat := digest.NewTransport(username, password)
	dac, err := dat.Client()
	if err != nil {
		t.Errorf("Client: %v", err)
		return
	}
	resp, err := dac.Get("http://" + testListenAddr + "/foo.txt")
	if err != nil {
		t.Errorf("roundtrip: %v", err)
		return
	}
	cont, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ReadAll(resp.Body): %v", err)
		return
	}
	if !bytes.Equal(cont, tu.HelloWorld) {
		t.Errorf("unexpected content: %v != exp %v", cont, tu.HelloWorld)
	}
	resp.Body.Close()

	c := &http.Client{Transport: &http.Transport{}}

	resp, err = c.Get("http://" + testListenAddr + "/foo.txt")
	if err != nil {
		t.Errorf("http.Get: %v", err)
		return
	}
	cont, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ReadAll(resp.Body): %v", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Unauthorized request success: %v", resp.Status)
	}
	if bytes.Equal(cont, tu.HelloWorld) {
		t.Errorf("Unauthorized data read!: %v", cont)
	}

	close(closeC)
	<-joinC

	// hack: ensure listener socket close before proceeding
	time.Sleep(100 * time.Millisecond)
}
