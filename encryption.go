package potoq

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"fmt"
	"io"
	"time"

	"github.com/Craftserve/potoq/packets"
)

func (handler *Handler) establishEncryptionAsServer() error {
	auth := handler.Authenticator
	// S->C Encryption Request
	original_token := make([]byte, 4)
	_, err := io.ReadFull(rand.Reader, original_token)
	if err != nil {
		return err
	}

	keyrequest := &packets.EncryptionRequestPacket{
		ServerID:  auth.ServerID,
		PublicKey: auth.PublicKey,
		Token:     original_token,
	}
	err = handler.DownstreamW.WritePacket(keyrequest, true)
	if err != nil {
		return err
	}

	// C->S Encryption Response
	var keyresponse packets.EncryptionResponsePacket
	_, err = packets.ParsePackets(handler.DownstreamR, &keyresponse)
	if err != nil {
		return err
	}

	decrypted_token, err := rsa.DecryptPKCS1v15(rand.Reader, auth.ServerKey, keyresponse.Token)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(decrypted_token, original_token) != 1 {
		return fmt.Errorf("Client returned invalid encrypted token!")
	}

	symmetric_key, err := rsa.DecryptPKCS1v15(rand.Reader, auth.ServerKey, keyresponse.Secret)
	if err != nil {
		return err
	}

	// Set up encrypted reader and writer
	reader, err := packets.NewEncryptedReader(handler.DownstreamC, symmetric_key)
	if err != nil {
		return err
	}
	writer, err := packets.NewEncryptedWriter(handler.DownstreamC, symmetric_key)
	if err != nil {
		return err
	}
	handler.DownstreamR = packets.NewPacketReader(bufio.NewReaderSize(reader, 128*1024), handler.CompressThreshold)
	handler.DownstreamW = packets.NewPacketWriter(bufio.NewWriterSize(writer, 128*1024), handler.CompressThreshold)

	// Verify player with session.minecraft.net
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return auth.HasJoined(ctx, handler, symmetric_key)
}
