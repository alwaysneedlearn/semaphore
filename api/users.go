package api

import (
	"bytes"
	"image/png"
	"net/http"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pro_interfaces"
	log "github.com/sirupsen/logrus"

	"github.com/semaphoreui/semaphore/util"
)

type UsersController struct {
	subscriptionService pro_interfaces.SubscriptionService
}

func NewUsersController(subscriptionService pro_interfaces.SubscriptionService) *UsersController {
	return &UsersController{
		subscriptionService: subscriptionService,
	}
}

type minimalUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

func (c *UsersController) GetUsers(w http.ResponseWriter, r *http.Request) {
	currentUser := helpers.GetFromContext(r, "user").(*db.User)
	users, err := helpers.Store(r).GetUsers(db.RetrieveQueryParams{
		Filter: r.URL.Query().Get("s"),
	})

	if err != nil {
		panic(err)
	}

	if currentUser.Admin {
		helpers.WriteJSON(w, http.StatusOK, users)
	} else {
		var result = make([]minimalUser, 0)

		for _, user := range users {
			result = append(result, minimalUser{
				ID:       user.ID,
				Name:     user.Name,
				Username: user.Username,
			})
		}

		helpers.WriteJSON(w, http.StatusOK, result)
	}
}

func (c *UsersController) AddUser(w http.ResponseWriter, r *http.Request) {
	var user db.UserWithPwd
	if !helpers.Bind(w, r, &user) {
		return
	}

	editor := helpers.GetFromContext(r, "user").(*db.User)
	if !editor.Admin {
		log.Warn(editor.Username + " is not permitted to create users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if user.Pro {
		ok, err := c.subscriptionService.CanAddProUser()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			helpers.WriteErrorStatus(
				w,
				"You have reached the limit of Pro users for your subscription.",
				http.StatusForbidden,
			)
			return
		}
	}

	var err error
	var newUser db.User

	if user.External {
		newUser, err = helpers.Store(r).CreateUserWithoutPassword(user.User)
	} else {
		newUser, err = helpers.Store(r).CreateUser(user)
	}

	if err != nil {
		log.Warn(editor.Username + " is not created: " + err.Error())
		if msg := userIdentityConflictMessage(err); msg != "" {
			helpers.WriteErrorStatus(w, msg, http.StatusBadRequest)
			return
		}
		helpers.WriteError(w, err)
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, newUser)
}
func readonlyUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := helpers.GetIntParam("user_id", w, r)

		if err != nil {
			return
		}

		user, err := helpers.Store(r).GetUser(userID)

		if err != nil {
			helpers.WriteError(w, err)
			return
		}

		editor := helpers.GetFromContext(r, "user").(*db.User)

		if !editor.Admin && editor.ID != user.ID {
			user = db.User{
				ID:       user.ID,
				Username: user.Username,
				Name:     user.Name,
			}
		}

		r = helpers.SetContextValue(r, "_user", user)
		next.ServeHTTP(w, r)
	})
}

func getUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := helpers.GetIntParam("user_id", w, r)

		if err != nil {
			return
		}

		user, err := helpers.Store(r).GetUser(userID)

		if err != nil {
			helpers.WriteError(w, err)
			return
		}

		editor := helpers.GetFromContext(r, "user").(*db.User)

		if !editor.Admin && editor.ID != user.ID {
			log.Warn(editor.Username + " is not permitted to edit users")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		r = helpers.SetContextValue(r, "_user", user)
		next.ServeHTTP(w, r)
	})
}

func (c *UsersController) UpdateUser(w http.ResponseWriter, r *http.Request) {
	targetUser := helpers.GetFromContext(r, "_user").(db.User)
	editor := helpers.GetFromContext(r, "user").(*db.User)

	var user db.UserWithPwd
	if !helpers.Bind(w, r, &user) {
		return
	}

	user.Name = strings.TrimSpace(user.Name)
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)

	if err := db.ValidateUser(user.User); err != nil {
		helpers.WriteError(w, err)
		return
	}

	if !editor.Admin && (user.Pro && !targetUser.Pro) {
		log.Warn(editor.Username + " is not permitted to mark users as Pro")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if user.Pro {
		ok, err := c.subscriptionService.CanAddProUser()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			helpers.WriteErrorStatus(
				w,
				"You have reached the limit of Pro users for your subscription.",
				http.StatusForbidden,
			)
			return
		}
	}

	if !editor.Admin && editor.ID != targetUser.ID {
		log.Warn(editor.Username + " is not permitted to edit users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if editor.ID == targetUser.ID && targetUser.Admin != user.Admin {
		log.Warn("User can't edit his own role")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if targetUser.External {
		if targetUser.Username != user.Username {
			helpers.WriteErrorStatus(w, "Username is not editable for external users", http.StatusBadRequest)
			return
		}
		if !strings.EqualFold(targetUser.Email, user.Email) {
			helpers.WriteErrorStatus(w, "Email is not editable for external users", http.StatusBadRequest)
			return
		}
		// Keep the stored email casing for external accounts.
		user.Email = targetUser.Email
	}

	if err := helpers.Store(r).UserIdentityConflict(user.Username, user.Email, targetUser.ID); err != nil {
		helpers.WriteError(w, err)
		return
	}

	user.ID = targetUser.ID
	if err := helpers.Store(r).UpdateUser(user); err != nil {
		if msg := userIdentityConflictMessage(err); msg != "" {
			helpers.WriteErrorStatus(w, msg, http.StatusBadRequest)
			return
		}
		helpers.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateUserPassword(w http.ResponseWriter, r *http.Request) {
	user := helpers.GetFromContext(r, "_user").(db.User)
	editor := helpers.GetFromContext(r, "user").(*db.User)

	var pwd struct {
		Pwd string `json:"password"`
	}

	if !editor.Admin && editor.ID != user.ID {
		log.Warn(editor.Username + " is not permitted to edit users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if user.External {
		log.Warn("Password is not editable for external users")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !helpers.Bind(w, r, &pwd) {
		return
	}

	if err := helpers.Store(r).SetUserPassword(user.ID, pwd.Pwd); err != nil {
		util.LogWarning(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	user := helpers.GetFromContext(r, "_user").(db.User)
	editor := helpers.GetFromContext(r, "user").(*db.User)

	if !editor.Admin && editor.ID != user.ID {
		log.Warn(editor.Username + " is not permitted to delete users")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := helpers.Store(r).DeleteUser(user.ID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusNoContent)
}

func totpQr(w http.ResponseWriter, r *http.Request) {
	user := helpers.GetFromContext(r, "_user").(db.User)

	if user.Totp == nil {
		helpers.WriteErrorStatus(w, "TOTP not enabled", http.StatusNotFound)
		return
	}

	key, err := otp.NewKeyFromURL(user.Totp.URL)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	image, err := key.Image(256, 256)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, image)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	pngBytes := buf.Bytes()

	w.Header().Add("Content-Type", "image/png")
	_, err = w.Write(pngBytes)
}

func enableTotp(w http.ResponseWriter, r *http.Request) {
	user := helpers.GetFromContext(r, "_user").(db.User)

	if !util.Config.Auth.Totp.Enabled {
		helpers.WriteErrorStatus(w, "TOTP not enabled", http.StatusBadRequest)
		return
	}

	if user.Totp != nil {
		helpers.WriteErrorStatus(w, "TOTP already enabled", http.StatusBadRequest)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Semaphore",
		AccountName: user.Email,
	})

	if err != nil {
		http.Error(w, "Error generating key", http.StatusInternalServerError)
		return
	}

	var code, hash string

	if util.Config.Auth.Totp.AllowRecovery {
		code, hash, err = util.GenerateRecoveryCode()
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
	}

	newTotp, err := helpers.Store(r).AddTotpVerification(user.ID, key.URL(), hash)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	newTotp.RecoveryCode = code

	helpers.WriteJSON(w, http.StatusOK, newTotp)
}

func disableTotp(w http.ResponseWriter, r *http.Request) {
	user := helpers.GetFromContext(r, "_user").(db.User)
	if user.Totp == nil {
		helpers.WriteErrorStatus(w, "TOTP not enabled", http.StatusBadRequest)
		return
	}

	totpID, err := helpers.GetIntParam("totp_id", w, r)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	err = helpers.Store(r).DeleteTotpVerification(user.ID, totpID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func userIdentityConflictMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unique") && strings.Contains(msg, "username"):
		return "Username already exists"
	case strings.Contains(msg, "unique") && strings.Contains(msg, "email"):
		return "Email already exists"
	case strings.Contains(msg, "duplicate") && strings.Contains(msg, "username"):
		return "Username already exists"
	case strings.Contains(msg, "duplicate") && strings.Contains(msg, "email"):
		return "Email already exists"
	case strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate"):
		return "Username or email already exists"
	default:
		return ""
	}
}
