package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"

	"github.com/99designs/gqlgen/handler"
	"github.com/photoview/photoview/api/dataloader"
	"github.com/photoview/photoview/api/graphql/models"
	"gorm.io/gorm"
)

var ErrUnauthorized = errors.New("unauthorized")

var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// Extracts user from token and adds it to the context
func addUserFromTokenToContext(r *http.Request, db *gorm.DB) (*http.Request, error) {
	tokenCookie, err := r.Cookie("auth-token")
	if err != nil {
		log.Println("Did not find auth-token cookie")
		return r, nil
	}

	user, err := dataloader.For(r.Context()).UserFromAccessToken.Load(tokenCookie.Value)
	if err != nil {
		log.Printf("Invalid token: %s\n", err)
		return nil, ErrUnauthorized
	}

	ctx := context.WithValue(r.Context(), userCtxKey, user)
	return r.WithContext(ctx), nil
}

// Middleware decodes the share session cookie and packs the session into context
func Middleware(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r, err := addUserFromTokenToContext(r, db)
			if err != nil {
				http.Error(w, "invalid authorization token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func TokenFromBearer(bearer *string) (*string, error) {
	regex, _ := regexp.Compile("^(?i)Bearer ([a-zA-Z0-9]{24})$")
	matches := regex.FindStringSubmatch(*bearer)
	if len(matches) != 2 {
		return nil, errors.New("invalid bearer format")
	}

	token := matches[1]
	return &token, nil
}

// UserFromContext finds the user from the context. REQUIRES Middleware to have run.
func UserFromContext(ctx context.Context) *models.User {
	raw, _ := ctx.Value(userCtxKey).(*models.User)
	return raw
}

func AuthWebsocketInit(db *gorm.DB) func(context.Context, handler.InitPayload) (context.Context, error) {
	return func(ctx context.Context, initPayload handler.InitPayload) (context.Context, error) {

		bearer, exists := initPayload["Authorization"].(string)
		if !exists {
			return ctx, nil
		}

		token, err := TokenFromBearer(&bearer)
		if err != nil {
			log.Printf("Invalid bearer format (websocket): %s\n", bearer)
			return nil, err
		}

		user, err := dataloader.For(ctx).UserFromAccessToken.Load(*token)
		if err != nil {
			log.Printf("Invalid token in websocket: %s\n", err)
			return nil, errors.New("invalid authorization token")
		}

		userCtx := context.WithValue(ctx, userCtxKey, user)

		return userCtx, nil
	}
}