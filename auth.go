package potoq

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	math_rand "math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Craftserve/potoq/packets"
)

var ErrBadLogin = fmt.Errorf("Bad Login!")
var ErrUnauthenticated = fmt.Errorf("Player unauthenticated")
var ErrTooManyRequests = fmt.Errorf("Too many requests, try again later!")

var authenticators []*Authenticator

func init() {
	var publicIPs []net.IP
	interfaces, _ := net.Interfaces()
	for _, intf := range interfaces {
		addrs, _ := intf.Addrs()
		for _, i := range addrs {
			ip := i.(*net.IPNet).IP.To4()
			if ip != nil && ip.IsGlobalUnicast() {
				publicIPs = append(publicIPs, ip)
			}
		}
	}
	for _, ip := range publicIPs {
		a, err := newAuthenticator(ip)
		if err != nil {
			panic(err)
		}
		authenticators = append(authenticators, a)
	}
}

func getRandomAuthenticator() *Authenticator {
	return authenticators[math_rand.Intn(len(authenticators))]
}

type Authenticator struct {
	ServerID   string
	ServerKey  *rsa.PrivateKey
	PublicKey  []byte // encoded
	HttpClient *http.Client
	Limiter    *rate.Limiter
}

func newAuthenticator(ip net.IP) (*Authenticator, error) {
	var buf [8]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		return nil, err
	}
	serverID := hex.EncodeToString(buf[:])

	serverKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	serverKey.Precompute()
	publicKey, err := x509.MarshalPKIXPublicKey(serverKey.Public())
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: time.Second, LocalAddr: &net.TCPAddr{IP: ip}}
	transport := &http.Transport{DialContext: dialer.DialContext}

	return &Authenticator{
		ServerID:  serverID,
		ServerKey: serverKey,
		PublicKey: publicKey,
		HttpClient: &http.Client{
			Transport: transport,
			Timeout:   3 * time.Second,
		},
		Limiter: rate.NewLimiter(200, 1000),
	}, nil
}

func (a *Authenticator) HasJoined(ctx context.Context, handler *Handler, secret []byte) error {
	if !ValidateNickname(handler.Nickname) {
		return fmt.Errorf("Invalid nickname")
	}

	if !a.Limiter.Allow() {
		return ErrTooManyRequests
	}

	var query = make(url.Values)
	query.Set("serverId", authDigest([]byte(a.ServerID), secret, a.PublicKey))
	query.Set("username", handler.Nickname)

	req, err := http.NewRequest("GET", "https://sessionserver.mojang.com/session/minecraft/hasJoined?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := a.HttpClient.Do(req.WithContext(ctx))
	if err != nil {
		handler.Log().WithError(err).Println("HasJoined failed")
		return ErrBadLogin
	}
	defer resp.Body.Close()

	var response struct {
		UUID       uuid.UUID              `json:"id"`
		Name       string                 `json:"name"`
		Properties []packets.AuthProperty `json:"properties"`
		// Legacy     bool                   `json:"legacy"`
		// Demo       bool                   `json:"demo"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return err
	}

	handler.Nickname = response.Name
	handler.UUID = response.UUID
	handler.AuthProperties = response.Properties

	return nil
}

func authDigest(serverID, secret, publicKey []byte) string {
	h := sha1.New()
	h.Write(serverID)
	h.Write(secret)
	h.Write(publicKey)
	hash := h.Sum(nil)

	// Check for negative hashes
	negative := (hash[0] & 0x80) == 0x80
	if negative {
		hash = twosComplement(hash)
	}

	// Trim away zeroes
	res := strings.TrimLeft(fmt.Sprintf("%x", hash), "0")
	if negative {
		res = "-" + res
	}

	return res
}

// little endian
func twosComplement(p []byte) []byte {
	carry := true
	for i := len(p) - 1; i >= 0; i-- {
		p[i] = byte(^p[i])
		if carry {
			carry = p[i] == 0xff
			p[i]++
		}
	}
	return p
}

func (a *Authenticator) HasPaid(ctx context.Context, nickname string) (paid bool, err error) {
	if !ValidateNickname(nickname) {
		return false, fmt.Errorf("Invalid nickname")
	}

	if !a.Limiter.Allow() {
		return false, ErrTooManyRequests
	}

	// Blocked from hetzner: "https://api.mojang.com/users/profiles/minecraft/" + nickname (?)
	req, err := http.NewRequest("GET", "https://api.minetools.eu/uuid/"+nickname, nil)
	if err != nil {
		return false, err
	}
	resp, err := a.HttpClient.Do(req.WithContext(ctx))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var v struct {
		Id string `json:"id"`
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(&v)
		if err != nil {
			return false, err
		}
		return v.Id != "null", nil
	case http.StatusTooManyRequests:
		return false, ErrTooManyRequests
	default:
		return false, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}
}

var nicknameRegexp = regexp.MustCompile("[A-Za-z0-9_]{3,16}")

func ValidateNickname(nickname string) bool {
	return nicknameRegexp.MatchString(nickname)
}

func OfflinePlayerUUID(nickname string) uuid.UUID {
	h := md5.New()
	io.WriteString(h, "OfflinePlayer:"+nickname)
	var uuid, _ = uuid.FromBytes(h.Sum(nil))
	uuid[6] = (uuid[6] & 0x0f) | uint8((3&0xf)<<4)
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // RFC 4122 variant
	return uuid
}
