package session

import (
	"context"
	"net/http"

	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type UserClaims struct {
	PreferredUsername string   `json:"preferred_username" validate:"required"`
	Name              string   `json:"name" validate:"required"`
	Email             string   `json:"email" validate:"required,email"`
	AvatarURL         string   `json:"picture" validate:"omitempty,url"`
	Sub               string   `json:"sub" validate:"required"`
	Iss               string   `json:"iss" validate:"required"`
	Groups            []string `json:"groups"`
}

func (sub *SessionSubsystem) findProvider(providerName string) *oidcProviderConfig {
	for _, provider := range sub.oidcConfig {
		if provider.name == providerName {
			return &provider
		}
	}

	return nil

}

func (sub *SessionSubsystem) Login(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("name")
	providerConfig := sub.findProvider(providerName)
	if providerConfig == nil {
		log.Error().Str("provider_name", providerName).Msg("callback on unknown provider")
		http.Error(w, "unknown provider", http.StatusBadRequest)
		return
	}

	state := randString(64)
	setStateCookie(w, state, r.TLS != nil)

	codeVerifier := randString(64)
	setCodeVerifierCookie(w, codeVerifier, r.TLS != nil)

	log.Trace().Msg("started oidc auth code login flow")
	http.Redirect(w, r, providerConfig.config.AuthCodeURL(state, oauth2.S256ChallengeOption(codeVerifier)), http.StatusFound)
}

func (sub *SessionSubsystem) Callback(w http.ResponseWriter, r *http.Request) {

	providerName := r.PathValue("name")
	providerConfig := sub.findProvider(providerName)
	if providerConfig == nil {
		log.Error().Str("provider_name", providerName).Msg("callback on unknown provider")
		http.Error(w, "unknown provider", http.StatusBadRequest)
		return
	}

	state, err := r.Cookie("state")
	if err != nil {
		log.Err(err).Str("provider_name", providerConfig.name).
			Msg("parsing callback session state cookie")
		http.Error(w, "state not found", http.StatusBadRequest)
		return
	}

	codeVerifierCookie, err := r.Cookie("code_verifier")
	if err != nil {
		log.Err(err).Str("provider_name", providerConfig.name).
			Msg("parsing code_verifier cookie")
		http.Error(w, "code_verifier not found", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != state.Value {
		log.Warn().Str("provider_name", providerConfig.name).
			Msg("matching url and cookie session state")
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}

	oauth2Token, err := providerConfig.config.Exchange(context.Background(), r.URL.Query().Get("code"), oauth2.VerifierOption(codeVerifierCookie.Value))

	if err != nil {
		log.Err(err).Str("provider_name", providerConfig.name).
			Msg("exchanging oauth token")
		http.Error(w, "Failed exchanging token", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Error().Str("provider_name", providerConfig.name).
			Msg("parsing id_token in oauth2 token")
		http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}

	idToken, err := providerConfig.verifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		log.Err(err).Str("provider_name", providerConfig.name).
			Msg("verifying id token")
		http.Error(w, "failed to verify id token", http.StatusInternalServerError)
		return
	}

	userInfo, err := providerConfig.provider.UserInfo(context.Background(), oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		log.Err(err).Str("provider_name", providerConfig.name).Msg("getting userinfo from provider")
		http.Error(w, "failed to get userinfo", http.StatusInternalServerError)
		return
	}

	var userClaims UserClaims
	err = userInfo.Claims(&userClaims)
	if err != nil {
		log.Err(err).Str("provider_name", providerConfig.name).Msg("unmarshaling user claims from userinfo endpoint")
		http.Error(w, "failed to parse userinfo claims", http.StatusInternalServerError)
		return
	}

	userClaims.Iss = idToken.Issuer
	userClaims.Sub = idToken.Subject

	err = validation.Validate.Struct(userClaims)
	if err != nil {
		log.Err(err).Msg("validating user claims")
	}

	sub.newSession(w, r, userClaims, providerConfig.name)

	http.Redirect(w, r, "/", http.StatusFound)
}
