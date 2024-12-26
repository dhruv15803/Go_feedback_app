package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dhruv15803/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *APIServer) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(payload.Email))
	username := strings.TrimSpace(payload.Username)
	password := strings.TrimSpace(payload.Password)

	if email == "" || username == "" || password == "" {
		s.writeJSONError(w, "email,username and password are compulsory fields", http.StatusBadRequest)
		return
	}
	// extracted email,username and password and checked that they are not empty
	// validate email , //
	if ok := s.validateEmail(email); !ok {
		s.writeJSONError(w, "Invalid email", http.StatusBadRequest)
		return
	}

	if ok := s.validatePassword(password); !ok {
		s.writeJSONError(w, "password should have atleast 6 characters,password should have atleast 1 special character,password should have atleast 1 uppercase character", http.StatusBadRequest)
		return
	}

	// email and password are valid
	// check if any users with above email or username already exists
	users, err := s.storage.Users.GetUsersByUsernameOrEmail(username, email)
	if err != nil {
		log.Println(err.Error())
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	if len(users) > 0 {
		s.writeJSONError(w, "user already exists", http.StatusBadRequest)
		return
	}

	// generate password hash
	hashedByte, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err.Error())
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	hashedPassword := string(hashedByte)
	user, err := s.storage.Users.CreateUser(username, email, hashedPassword)
	if err != nil {
		log.Println(err.Error())
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	// generate jwt token and use payload user.Id , set token in cookie for persisting

	tokenString, err := s.GenerateJWT(user.Id)
	if err != nil {
		log.Println(err.Error())
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 48),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	type Envelope struct {
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}
	if err = s.writeJSON(w, Envelope{Message: "user registered succesfully", User: *user}, http.StatusCreated); err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
	}
}

func (s *APIServer) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Extract and trim input values
	email := strings.ToLower(strings.TrimSpace(payload.Email))
	password := strings.TrimSpace(payload.Password)

	// Validate input fields
	if email == "" || password == "" {
		s.writeJSONError(w, "email and password are required fields", http.StatusBadRequest)
		return
	}

	// Fetch the user by email
	user, err := s.storage.Users.GetUserByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, "invalid email or password", http.StatusUnauthorized)
		} else {
			s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		}
		return
	}

	// Compare the provided password with the stored hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.writeJSONError(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// Generate a JWT token
	tokenString, err := s.GenerateJWT(user.Id)
	if err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	// Set the token in a cookie
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 48),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &cookie)

	// Respond with a success message and user information
	type Response struct {
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}
	s.writeJSON(w, Response{
		Message: "user logged in successfully",
		User:    *user,
	}, http.StatusOK)
}

func (s *APIServer) getAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		log.Println("asdk;laskd")
		s.writeJSONError(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	user, err := s.storage.Users.GetUserById(userId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		}
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if err = s.writeJSON(w, user, http.StatusOK); err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
	}
}

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware is the authentication middleware
func (s *APIServer) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the JWT token from the cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Error(w, "unauthorized: missing or invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value
		if strings.TrimSpace(tokenString) == "" {
			http.Error(w, "unauthorized: missing or invalid token", http.StatusUnauthorized)
			return
		}

		// Parse and validate the JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Ensure the signing method is HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			// Return the secret key for validation
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// Extract the userId from the token's payload
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		userIdFloat, ok := claims["userId"].(float64)
		if !ok {
			http.Error(w, "unauthorized: invalid token payload", http.StatusUnauthorized)
			return
		}

		userId := int(userIdFloat)

		user, err := s.storage.Users.GetUserById(userId)
		if err != nil {
			http.Error(w, "unauthorized: invalid token payload", http.StatusUnauthorized)
			return
		}

		// Attach the userId to the context
		ctx := context.WithValue(r.Context(), userIDKey, user.Id)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *APIServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "user not authorized", http.StatusUnauthorized)
		return
	}
	// authenticated cookie name -> auth_token
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	type Envelope struct {
		Message string `json:"message"`
	}
	if err := s.writeJSON(w, Envelope{Message: "logged out successfully"}, http.StatusOK); err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
	}
}

func (s *APIServer) validateEmail(email string) bool {
	if !strings.Contains(email, "@") {
		return false
	}
	slice := strings.Split(email, "@")
	if len(slice) < 2 {
		return false
	}

	if len(slice[0]) < 1 || len(slice[1]) < 1 {
		return false
	}
	return true
}

/*
min length 6
atleast 1 upper case char
atleast 1 special char
*/
func (s *APIServer) validatePassword(password string) bool {
	const specialChars string = "@#$%&!"
	const upperCaseChars string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	if len(password) < 6 {
		return false
	}

	isSpecialChar := false
	for _, specialChar := range specialChars {
		if strings.Contains(password, string(specialChar)) {
			isSpecialChar = true
			break
		}
	}
	if !isSpecialChar {
		return false
	}

	// check uppercase
	isUpperCase := false
	for _, upperCaseChar := range upperCaseChars {
		if strings.Contains(password, string(upperCaseChar)) {
			isUpperCase = true
			break
		}
	}

	if !isUpperCase {
		return false
	}

	return true
}
