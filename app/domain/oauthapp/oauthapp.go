// Package oauthapp signs a person in with Google and returns them to CloudChamber.
package oauthapp

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/name"
	"github.com/MKAbdElrahman/cloudchamber/business/types/password"
	"github.com/MKAbdElrahman/cloudchamber/business/types/role"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

const googleProvider = "google"

type app struct {
	log       *logger.Logger
	auth      *auth.Auth
	userBus   userbus.ExtBusiness
	tokenKey  string
	bubbleURL string
}

func newApp(cfg Config) *app {
	callbackBase := cfg.GoogleCallbackBase
	if callbackBase == "" {
		callbackBase = "http://127.0.0.1:6100"
	}
	bubbleURL := cfg.BubbleURL
	if bubbleURL == "" {
		bubbleURL = "http://127.0.0.1:5173"
	}

	if cfg.SessionSecret != "" {
		store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
		store.Options.HttpOnly = true
		store.Options.SameSite = http.SameSiteLaxMode
		store.Options.Secure = false
		gothic.Store = store
	}

	goth.UseProviders(
		google.New(cfg.GoogleKey, cfg.GoogleSecret, callbackBase+"/api/auth/google/callback"),
	)

	gothic.GetProviderName = func(r *http.Request) (string, error) {
		return web.Param(r, "provider"), nil
	}

	return &app{
		auth:      cfg.Auth,
		log:       cfg.Log,
		userBus:   cfg.UserBus,
		tokenKey:  cfg.TokenKey,
		bubbleURL: bubbleURL,
	}
}

func (a *app) authenticate(ctx context.Context, r *http.Request) web.Encoder {
	if web.Param(r, "provider") != googleProvider {
		return errs.New(errs.NotFound, errors.New("provider not supported"))
	}

	gothic.BeginAuthHandler(web.GetWriter(ctx), r)

	return web.NewNoResponse()
}

func (a *app) authCallback(ctx context.Context, r *http.Request) web.Encoder {
	w := web.GetWriter(ctx)
	if web.Param(r, "provider") != googleProvider {
		return errs.New(errs.NotFound, errors.New("provider not supported"))
	}

	gu, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		a.redirectError(w, r, "Google sign-in failed")
		return web.NewNoResponse()
	}

	usr, err := a.account(ctx, gu)
	if err != nil {
		a.log.Info(ctx, "google-signin", "status", "rejected", "err", err)
		a.redirectError(w, r, "Google sign-in failed")
		return web.NewNoResponse()
	}

	claims := auth.Claims{
		Subject:   usr.ID.String(),
		Issuer:    a.auth.Issuer(),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(8760 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		Roles:     role.ParseToString(usr.Roles),
	}

	token, err := a.auth.GenerateToken(a.tokenKey, claims)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	target, err := url.Parse(a.bubbleURL)
	if err != nil {
		return errs.New(errs.Internal, err)
	}
	query := target.Query()
	query.Set("token", token)
	query.Set("name", usr.Name.String())
	target.RawQuery = query.Encode()

	http.Redirect(w, r, target.String(), http.StatusFound)

	return web.NewNoResponse()
}

func (a *app) logout(ctx context.Context, r *http.Request) web.Encoder {
	w := web.GetWriter(ctx)

	if err := gothic.Logout(w, r); err != nil {
		return errs.New(errs.Internal, err)
	}

	http.Redirect(w, r, a.bubbleURL, http.StatusTemporaryRedirect)

	return web.NewNoResponse()
}

func (a *app) account(ctx context.Context, gu goth.User) (userbus.User, error) {
	addr, err := mail.ParseAddress(gu.Email)
	if err != nil || addr.Address == "" {
		return userbus.User{}, errors.New("google account has no email")
	}

	usr, err := a.userBus.QueryByEmail(ctx, *addr)
	switch {
	case err == nil && !usr.Enabled:
		return userbus.User{}, errors.New("account is disabled")
	case err == nil:
		return usr, nil
	case !errors.Is(err, userbus.ErrNotFound):
		return userbus.User{}, err
	}

	pw, err := randomPassword()
	if err != nil {
		return userbus.User{}, err
	}

	created, err := a.userBus.Create(ctx, uuid.Nil, userbus.NewUser{
		Name:     accountName(gu.Name),
		Email:    *addr,
		Roles:    []role.Role{role.User},
		Password: pw,
	})
	if err != nil {
		return userbus.User{}, err
	}

	return created, nil
}

func (a *app) redirectError(w http.ResponseWriter, r *http.Request, message string) {
	target, err := url.Parse(a.bubbleURL)
	if err != nil {
		http.Error(w, message, http.StatusBadGateway)
		return
	}
	query := target.Query()
	query.Set("error", message)
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func accountName(raw string) name.Name {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ', r == '\'', r == '-':
			b.WriteRune(r)
		}
	}

	cleaned := strings.TrimSpace(b.String())
	if len(cleaned) > 20 {
		cleaned = strings.TrimSpace(cleaned[:20])
	}

	parsed, err := name.Parse(cleaned)
	if err != nil {
		return name.MustParse("Member")
	}

	return parsed
}

func randomPassword() (password.Password, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, 16)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return password.Password{}, fmt.Errorf("random password: %w", err)
		}
		buf[i] = alphabet[n.Int64()]
	}

	return password.Parse(string(buf))
}
