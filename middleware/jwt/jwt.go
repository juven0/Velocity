package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

type JWTError string

const (
	SingVerifFaildError     JWTError = "signature verification failed"
	InvalidTokenFormatError JWTError = "invalid token format"
	SignVerificationError   JWTError = "signature verification failed"
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
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now.Unix()
	}

	if claims.ExpirationTime == 0 {
		claims.ExpirationTime = now.Add(j.Config.TokenExpiration).Unix()
	}

	if claims.Issuer == "" && j.Config.Issuer != "" {
		claims.Issuer = j.Config.Issuer
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}
	headerEncoded := base64URLEncode(headerJSON)

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}
	claimsEncoded := base64URLEncode(claimsJSON)

	message := headerEncoded + "." + claimsEncoded

	signature, err := j.sign(message)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	token := message + "." + signature
	return token, nil
}

func (j *JWTManager) ParsToken(tokenString string) (*Token, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New(string(InvalidTokenFormatError))
	}

	headerPart, claimsPart, signaturePart := parts[0], parts[1], parts[2]
	
	message := headerPart + "." + claimsPart
	if err := j.verify(message, signaturePart); err != nil {
		return nil, fmt.Errorf("%w: %w",SignVerificationError, err)
	}

	headerData, err := base64URLDecode(headerPart)
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}
	
	var header Header
	if err := json.Unmarshal(headerData, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal header: %w", err)
	}
	
	if header.Algorithm != string(j.Config.SigninMethode) {
		return nil, errors.New("algorithm mismatch")
	}
	
	claimsData, err := base64URLDecode(claimsPart)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}
	
	var claims Claims
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}
	
	now := time.Now().Unix()
	
	if claims.ExpirationTime > 0 && claims.ExpirationTime < now {
		return &Token{
			Header:    header,
			Claims:    claims,
			Signature: signaturePart,
			Raw:       tokenString,
			Valid:     false,
		}, errors.New("token is expired")
	}
	
	if claims.NotBefore > 0 && claims.NotBefore > now {
		return &Token{
			Header:    header,
			Claims:    claims,
			Signature: signaturePart,
			Raw:       tokenString,
			Valid:     false,
		}, errors.New("token used before valid")
	}
	
	return &Token{
		Header:    header,
		Claims:    claims,
		Signature: signaturePart,
		Raw:       tokenString,
		Valid:     true,
	}, nil
}
func (j *JWTManager) extractToken(ctx *types.Context)(string, error){
	parts := strings.Split(j.Config.TokenLookUp , ":")

	if len(parts) != 2 {
		return nil, errors.New("invalid token lookup format")
	}

	method, key := parts[0], parts[1]

	switch method {
	case "header":
		authHeader := string(ctx.Request.Header.Peek(key))

		if authHeader == "" {
			return "", errors.New("missing authorization header")
		 }
		if strings.HasPrefix(authHeader, "Bearer "){
			return strings.TrimPrefix(authHeader, "Bearer ") , nil
		}
		return authHeader, nil
	case "query"
	}
}
func (j *JWTManager) Middleware() types.HandlerFunc{
	return func(ctx *types.Context) error{
		tokenString, err := j.
	}
}
