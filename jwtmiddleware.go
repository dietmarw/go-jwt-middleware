package jwtmiddleware

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"net/http"
	"strings"
)

type errorHandler func(w http.ResponseWriter, r *http.Request, err string)

// Options is a struct for specifying configuration options for the middleware.
type Options struct {
	// The function that will return the Key to validate the JWT.
	// It can be either a shared secret or a public key.
	// Default value: nil
	ValidationKeyGetter jwt.Keyfunc
	// The name of the property in the request where the user information
	// from the JWT will be stored.
	// Default value: "user"
	UserProperty string
	// The function that will be called when there's an error validating the token
	// Default value:
	ErrorHandler errorHandler
	// A boolean indicating if the credentials are required or not
	// Default value: false
	CredentialsOptional bool
}

type JWTMiddleware struct {
	Options Options
}

func OnError(w http.ResponseWriter, r *http.Request, err string) {
	http.Error(w, err, http.StatusUnauthorized)
}

// New constructs a new Secure instance with supplied options.
func New(options ...Options) *JWTMiddleware {

	var opts Options
	if len(options) == 0 {
		opts = Options{}
	} else {
		opts = options[0]
	}

	if opts.UserProperty == "" {
		opts.UserProperty = "user"
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = OnError
	}

	return &JWTMiddleware{
		Options: opts,
	}
}

// Special implementation for Negroni, but could be used elsewhere.
func (m *JWTMiddleware) HandlerWithNext(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	err := m.CheckJWT(w, r)

	// If there was an error, do not call next.
	if err == nil && next != nil {
		next(w, r)
	}
}

func (m *JWTMiddleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let secure process the request. If it returns an error,
		// that indicates the request should not continue.
		err := m.CheckJWT(w, r)

		// If there was an error, do not continue.
		if err != nil {
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (m *JWTMiddleware) CheckJWT(w http.ResponseWriter, r *http.Request) error {

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		if m.Options.CredentialsOptional {
			return nil
		}
		errorMsg := "Authorization header isn't sent"
		m.Options.ErrorHandler(w, r, errorMsg)
		return errors.New(errorMsg)
	}

	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		errorMsg := "Authorization header format must be Bearer {token}"
		m.Options.ErrorHandler(w, r, errorMsg)
		return errors.New(errorMsg)
	}

	token := authHeaderParts[1]

	parsedToken, err := jwt.Parse(token, m.Options.ValidationKeyGetter)

	if err != nil {
		m.Options.ErrorHandler(w, r, err.Error())
		return err
	}

	if !parsedToken.Valid {
		m.Options.ErrorHandler(w, r, "The token isn't valid")
	}

	context.Set(r, m.Options.UserProperty, parsedToken)

	return nil
}
