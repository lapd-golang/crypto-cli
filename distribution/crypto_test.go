// Copyright © 2018 SENETAS SECURITY PTY LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package distribution_test

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Senetas/crypto-cli/crypto"
	"github.com/Senetas/crypto-cli/distribution"
	"github.com/google/uuid"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/udhos/equalfile"
)

var (
	passphrase = "196884 = 196883 + 1"
	opts       = &crypto.Opts{
		Salt:    "MgSO4(H2O)x",
		EncType: crypto.Pbkdf2Aes256Gcm,
		Compat:  false,
	}
)

type ConstReader byte

func (r ConstReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = byte(r)
	}
	return len(b), nil
}

func TestCrypto(t *testing.T) {
	opts.SetPassphrase(passphrase)

	c, err := distribution.NewDecrypto(opts)
	if err != nil {
		t.Fatal("could not create decrypto")
	}

	e, err := distribution.EncryptKey(*c, opts)
	if err != nil {
		t.Fatal(err)
	}

	d, err := distribution.DecryptKey(e, opts)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*c, d) {
		t.Fatalf("inversion failed, c = %s, d = %s", c, d)
	}
}

func TestCryptoBlobs(t *testing.T) {
	opts.SetPassphrase(passphrase)

	dir := filepath.Join(os.TempDir(), "com.senetas.crypto", uuid.New().String())
	size, d, fn, err := mkRandFile(t, dir)
	if err != nil {
		t.Error(err)
		return
	}
	encpath := filepath.Join(dir, "enc")
	decpath := filepath.Join(dir, "dec")

	defer func() {
		if err = os.RemoveAll(dir); err != nil {
			t.Logf(err.Error())
		}
	}()

	c, err := distribution.NewDecrypto(opts)
	if err != nil {
		t.Error(err)
		return
	}

	blob := distribution.NewLayer(fn, d, size, c)

	enc, err := blob.EncryptBlob(opts, encpath)
	if err != nil {
		t.Error(err)
		return
	}

	fi, err := os.Stat(encpath)
	if err != nil {
		t.Error(err)
		return
	} else if fi.Size() != enc.GetSize() {
		t.Error(errors.Errorf("encrypted file is incorrect size: %d vs %d", fi.Size(), enc.GetSize()))
	}

	dec, err := enc.DecryptBlob(opts, decpath)
	if err != nil {
		t.Error(err)
		return
	}

	fi, err = os.Stat(decpath)
	if err != nil {
		t.Error(err)
		return
	} else if fi.Size() != dec.GetSize() {
		t.Error(errors.Errorf("decrypted file is incorrect size: %d vs %d", fi.Size(), dec.GetSize()))
	}

	equal, err := equalfile.CompareFile(fn, dec.GetFilename())
	if err != nil {
		t.Error(err)
		return
	}

	if !equal {
		showContents(t, fn, decpath)
		return
	}

	if blob.GetDigest().String() != dec.GetDigest().String() {
		t.Errorf("digests do not match: orig: %s decrypted: %s", blob.GetDigest(), dec.GetDigest())
		return
	}
}

func TestCompressBlobs(t *testing.T) {
	opts.SetPassphrase(passphrase)

	dir := filepath.Join(os.TempDir(), "com.senetas.crypto", uuid.New().String())
	size, d, fn, err := mkConstFile(t, dir)
	if err != nil {
		t.Error(err)
		return
	}
	compath := filepath.Join(dir, "enc.gz")
	decpath := filepath.Join(dir, "dec")

	defer func() {
		if err = os.RemoveAll(dir); err != nil {
			t.Logf(err.Error())
		}
	}()

	blob := distribution.NewPlainLayer(fn, d, size)

	com, err := blob.Compress(compath)
	if err != nil {
		t.Error(err)
		return
	}

	fi, err := os.Stat(compath)
	if err != nil {
		t.Error(err)
		return
	} else if fi.Size() != com.GetSize() {
		t.Error(errors.Errorf("compressed file is incorrect size: %d vs %d", fi.Size(), com.GetSize()))
	}

	dec, err := com.Decompress(decpath)
	if err != nil {
		t.Error(err)
		return
	}

	fi, err = os.Stat(decpath)
	if err != nil {
		t.Error(err)
		return
	} else if fi.Size() != dec.GetSize() {
		t.Error(errors.Errorf("decompressed file is incorrect size: %d vs %d", fi.Size(), dec.GetSize()))
	}

	equal, err := equalfile.CompareFile(fn, dec.GetFilename())
	if err != nil {
		t.Error(err)
		return
	}

	if !equal {
		showContents(t, fn, decpath)
		return
	}

	if blob.GetDigest().String() != dec.GetDigest().String() {
		t.Errorf("digests do not match: orig: %s decrypted: %s", blob.GetDigest(), dec.GetDigest())
		return
	}
}

func mkConstFile(t *testing.T, dir string) (_ int64, _ digest.Digest, _ string, err error) {
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}

	file := filepath.Join(dir, "plain")
	fh, err := os.Create(file)
	if err != nil {
		return
	}
	defer func() {
		if err = fh.Close(); err != nil {
			t.Log(err)
		}
	}()

	digester := digest.Canonical.Digester()
	mw := io.MultiWriter(digester.Hash(), fh)

	z := ConstReader(0)

	n, err := io.CopyN(mw, z, 1024)
	if err != nil {
		return
	}

	return n, digester.Digest(), file, nil
}

func mkRandFile(t *testing.T, dir string) (_ int64, _ digest.Digest, _ string, err error) {
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}

	fn := filepath.Join(dir, "plain")
	fh, err := os.Create(fn)
	if err != nil {
		return
	}
	defer func() {
		if err = fh.Close(); err != nil {
			t.Log(err)
		}
	}()

	digester := digest.Canonical.Digester()
	mw := io.MultiWriter(digester.Hash(), fh)

	r := rand.Reader

	n, err := io.CopyN(mw, r, 1024)
	if err != nil {
		return
	}

	return n, digester.Digest(), fn, nil
}

func showContents(t *testing.T, fn, decpath string) {
	a := readFile(t, fn)
	b := readFile(t, decpath)
	t.Errorf("decryption is not inverting encryption:\nPlaintext: %v\nDecrypted: %v", a, b)
}

func readFile(t *testing.T, filename string) []byte {
	fh, err := os.Open(filename)
	if err != nil {
		return []byte(fmt.Sprintf("[could not read %s]", filename))
	}
	contents, err := ioutil.ReadAll(fh)
	if err != nil {
		return []byte(fmt.Sprintf("[could not read %s]", filename))
	}
	return contents
}
