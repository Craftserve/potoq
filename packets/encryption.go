package packets

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
)

// AES CFB-8, version from stdlib is not working

type cfb8 struct {
	c                cipher.Block
	blockSize        int
	iv, iv_real, tmp []byte
	de               bool
}

func newCFB8(c cipher.Block, iv []byte, decrypt bool) *cfb8 {
	if len(iv) != 16 {
		panic("bad iv length!")
	}
	cp := make([]byte, 256)
	copy(cp, iv)
	return &cfb8{
		c:         c,
		blockSize: c.BlockSize(),
		iv:        cp[:16],
		iv_real:   cp,
		tmp:       make([]byte, 16),
		de:        decrypt,
	}
}

func (cf *cfb8) XORKeyStream(dst, src []byte) {
	for i := 0; i < len(src); i++ {
		val := src[i]
		cf.c.Encrypt(cf.tmp, cf.iv)
		val = val ^ cf.tmp[0]

		if cap(cf.iv) >= 17 {
			cf.iv = cf.iv[1:17]
		} else {
			copy(cf.iv_real, cf.iv[1:])
			cf.iv = cf.iv_real[:16]
		}

		if cf.de {
			cf.iv[15] = src[i]
		} else {
			cf.iv[15] = val
		}
		dst[i] = val
	}
}

// > io.Reader

type encryptedReader struct {
	// implements io.Reader with minecraft protocol decryption
	r         io.Reader
	decrypter *cfb8
}

func NewEncryptedReader(r io.Reader, secret []byte) (reader io.Reader, err error) {
	var block cipher.Block
	block, err = aes.NewCipher(secret)
	if err != nil {
		return
	}
	decrypter := newCFB8(block, secret, true)
	reader = &encryptedReader{r, decrypter}
	return
}

func (reader *encryptedReader) Read(p []byte) (n int, err error) {
	n, err = reader.r.Read(p)
	if err != nil {
		return
	}
	reader.decrypter.XORKeyStream(p[:n], p[:n])
	return
}

// > io.Writer

type encryptedWriter struct {
	// implements io.Writer with minecraft protocol decryption
	w         io.Writer
	encrypter *cfb8
}

func NewEncryptedWriter(w io.Writer, secret []byte) (writer io.Writer, err error) {
	var block cipher.Block
	block, err = aes.NewCipher(secret)
	if err != nil {
		return
	}
	encrypter := newCFB8(block, secret, false)
	writer = &encryptedWriter{w, encrypter}
	return
}

func (writer *encryptedWriter) Write(p []byte) (n int, err error) {
	writer.encrypter.XORKeyStream(p, p)
	n, err = writer.w.Write(p)
	return
}
