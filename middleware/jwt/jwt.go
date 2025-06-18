package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"hash"
	"strings"
	"time"

	"github.com/juven0/Velocity/types"
)

type LookUpMethode string

const (
	HeaderJWT LookUpMethode = "header:Authorization"
	QueryJWT  LookUpMethode = "query:token"
	CookieJWT LookUpMethode = "cookie:jwt"
)

type SigninMethod string

const (
	HS256 SigninMethod = "HS256"
	HS384 SigninMethod = "HS384"
	HS512 SigninMethod = "HS512"
)

type Header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type Claims struct {
	Issuer         string                 `json:"iss,omitempty"`
	Subject        string                 `json:"sub,omitempty"`
	Audience       string                 `json:"aud,omitempty"`
	ExpirationTime int64                  `json:"exp,omitempty"`
	NotBefore      int64                  `json:"nbf,omitempty"`
	IssuedAt       int64                  `json:"iat,omitempty"`
	JwtID          string                 `json:"jti,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
	Role           string                 `json:"role,omitempty"`
	Username       string                 `json:"username,omitempty"`
	Custom         map[string]interface{} `json:"-"`
}

type Token struct {
	Header    Header
	Claims    Claims
	Signature string
	Raw       string
	Valid     bool
}

type Config struct {
	SecretKey        string
	TokenLookUp      LookUpMethode
	SigninMethode    SigninMethod
	ContextKey       string
	ErrorHandler     func(*types.Context, error) error
	SuccessHandler   func(*types.Context) error
	TokenExpiration  time.Duration
	RefreshThreshold time.Duration
	Issuer           string
}

var DefaultConfog = Config{
	SecretKey:        "",
	TokenLookUp:      HeaderJWT,
	SigninMethode:    HS256,
	ContextKey:       "user",
	TokenExpiration:  24 * time.Hour,
	RefreshThreshold: 2 * time.Hour,
	Issuer:           "velocity-framework",
}

type JWTManager struct {
	Config Config
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64URLDecode(str string) ([]byte, error) {
	if m := len(str) % 4; m != 0 {
		str += strings.Repeat("=", 4-m)
	}
	return base64.URLEncoding.DecodeString(str)
}

func getHashFunc(method SigninMethod) func() hash.Hash {
	switch method {
	case HS256:
		return sha256.New
	case HS384:
		return sha512.New384
	case HS512:
		return sha512.New
	default:
		return sha256.New
	}
}

func (j *JWTManager) sign(data string) (string, error) {
	hashFunc := getHashFunc(j.Config.SigninMethode)
	h := hmac.New(hashFunc, []byte(j.Config.SecretKey))
	h.Write([]byte(data))
	signature := h.Sum(nil)

	return base64URLEncode(signature), nil
}

func (j *JWTManager) verify(data, signature string) error {
	expectedSignature, err := j.sign(data)
	if err != nil {
		return err
	}

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("signature verification failed")
	}
	return nil
}

func (j *JWTManager) GenerateToken(claims Claims) (string, error) {
	header := Header{
		Algorithm: string(j.Config.SigninMethode),
		Type:      "JWT",
	}

	now := time.Now()

	if claims.ExpirationTime == 0 {
		claims.ExpirationTime = now.Add()
	}
}
