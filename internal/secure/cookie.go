package secure

import (
	"fmt"
	"github.com/mylxsw/secure-proxy/config"
	"net/http"

	"github.com/gorilla/securecookie"
)

type CookieManager struct {
	HashKey      []byte
	BlockKey     []byte
	CookieName   string
	Domain       string
	MaxAge       int
	secureCookie *securecookie.SecureCookie
}

func NewCookieManager(cookieName, domain string, hashKey, blockKey []byte, maxAge int) *CookieManager {
	return &CookieManager{
		CookieName:   cookieName,
		Domain:       domain,
		HashKey:      hashKey,
		BlockKey:     blockKey,
		MaxAge:       maxAge,
		secureCookie: securecookie.New(hashKey, blockKey),
	}
}
func (sc *CookieManager) SetCookie(rw http.ResponseWriter, userAuthInfo config.UserAuthInfo) error {
	encoded, err := sc.secureCookie.Encode(sc.CookieName, userAuthInfo)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     sc.CookieName,
		Value:    encoded,
		Path:     "/",
		Domain:   sc.Domain,
		HttpOnly: true,
		MaxAge:   sc.MaxAge,
	}

	http.SetCookie(rw, cookie)
	return nil
}

func (sc *CookieManager) GetCookie(req *http.Request) (*config.UserAuthInfo, error) {
	cookie, err := req.Cookie(sc.CookieName)
	if err != nil {
		return nil, err
	}

	var userAuthInfo config.UserAuthInfo
	if err := sc.secureCookie.Decode(sc.CookieName, cookie.Value, &userAuthInfo); err != nil {
		return nil, err
	}

	return &userAuthInfo, nil
}

func (sc *CookieManager) BuildAuthHandler() func(req *http.Request) (*config.UserAuthInfo, error) {
	return func(req *http.Request) (*config.UserAuthInfo, error) {
		userAuthInfo, err := sc.GetCookie(req)
		if err != nil {
			return nil, err
		}

		if userAuthInfo == nil || userAuthInfo.Account == "" {
			return nil, fmt.Errorf("user not login")
		}

		return userAuthInfo, nil
	}
}
