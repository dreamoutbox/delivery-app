package session

import (
	"log"

	"github.com/antonlindstrom/pgstore"
	"github.com/dreamoutbox/delivery-app/internal/config"
	"github.com/dreamoutbox/delivery-app/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const SessionName = "delivery_app_session"

var Store sessions.Store

func InitStore(cfg *config.Config) {
	pgStore, err := pgstore.NewPGStore(cfg.DSN(), []byte(cfg.SessionSecret))
	if err != nil {
		log.Printf("Warning: failed to connect pgstore: %v. Falling back to CookieStore.", err)
		Store = sessions.NewCookieStore([]byte(cfg.SessionSecret))
		return
	}
	Store = pgStore
}

type SessionUser struct {
	ID    uint
	Name  string
	Email string
	Role  string
}

func SetUser(c *gin.Context, user *models.User) error {
	sess, err := Store.Get(c.Request, SessionName)
	if err != nil {
		sess, _ = Store.New(c.Request, SessionName)
	}

	sess.Values["user_id"] = user.ID
	sess.Values["user_name"] = user.Name
	sess.Values["user_email"] = user.Email
	sess.Values["user_role"] = user.Role

	return Store.Save(c.Request, c.Writer, sess)
}

func GetUser(c *gin.Context) *SessionUser {
	sess, err := Store.Get(c.Request, SessionName)
	if err != nil {
		return nil
	}

	userIDVal, ok := sess.Values["user_id"]
	if !ok || userIDVal == nil {
		return nil
	}

	var userID uint
	switch v := userIDVal.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case int64:
		userID = uint(v)
	default:
		return nil
	}

	role, _ := sess.Values["user_role"].(string)
	name, _ := sess.Values["user_name"].(string)
	email, _ := sess.Values["user_email"].(string)

	if userID == 0 || role == "" {
		return nil
	}

	return &SessionUser{
		ID:    userID,
		Name:  name,
		Email: email,
		Role:  role,
	}
}

func Clear(c *gin.Context) error {
	sess, err := Store.Get(c.Request, SessionName)
	if err != nil {
		return nil
	}

	sess.Options.MaxAge = -1
	return Store.Save(c.Request, c.Writer, sess)
}
