package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

const (
	verifyTokenSize = 6
	resetTokenSize  = 6
)

// RegisterRoutes initializes the api endpoints and maps the requests to specific functions. The API will
// make use of the passed in Mailer and database connection. What HTTP methods would be most appropriate
// for each route?
func RegisterRoutes(router *mux.Router, m Mailer, db *sql.DB) {
	router.HandleFunc("/api/auth/signup", signup(m, db)).Methods(http.MethodPost /*YOUR CODE HERE*/)
	router.HandleFunc("/api/auth/signin", signin(db)).Methods(http.MethodPost /*YOUR CODE HERE*/)
	router.HandleFunc("/api/auth/logout", logout).Methods(http.MethodPost, http.MethodGet /*YOUR CODE HERE*/)
	router.HandleFunc("/api/auth/verify", verify(db)).Methods(http.MethodPost, http.MethodGet /*YOUR CODE HERE*/)
	router.HandleFunc("/api/auth/sendreset", sendReset(m, db)).Methods(http.MethodPost /*YOUR CODE HERE*/)
	router.HandleFunc("/api/auth/resetpw", resetPassword(db)).Methods(http.MethodPost /*YOUR CODE HERE*/)
}

// A function that handles signing a user up for Bearchat.
func signup(m Mailer, DB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Obtain the credentials from the request body
		var c Credentials
		err := json.NewDecoder(r.Body).Decode(&c)
		// Check if the username already exists
		row, err := DB.Query("SELECT * FROM users WHERE username = ?", c.Username)
		// Check for any errors
		if err != nil {
			http.Error(w, "error querying database for username", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		// Check boolean returned from query
		if row.Next() {
			http.Error(w, "username already exists", http.StatusBadRequest)
			return
		}
		// Check if the email already exists
		row, err = DB.Query("SELECT * FROM users WHERE email = ?", c.Email)
		// Check for any errors
		if err != nil {
			http.Error(w, "error querying database for email", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		// Check boolean returned from query
		if row.Next() {
			http.Error(w, "email already exists", http.StatusBadRequest)
			return
		}
		// Hash the password using bcrypt and store the hashed password in a variable
		pass, err := bcrypt.GenerateFromPassword([]byte(c.Password /*YOUR CODE HERE*/), bcrypt.DefaultCost)

		// Check for errors during hashing process
		if err != nil {
			http.Error(w, "error preparing password for storage", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		// Create a new user UUID, convert it to string, and store it within a variable
		id, err := uuid.FromBytes([]byte(c.Password))
		userID := id.String()
		// Create new verification token with the default token size (look at GetRandomBase62 and our constants)
		vertoken := GetRandomBase62(verifyTokenSize)
		// Store credentials in database
		_, err = DB.Exec("INSERT INTO users VALUES (?,?,?,?,?,?,?)", c.Username, c.Email, pass, 0, "", vertoken, userID)
		// Check for errors in storing the credentials
		if err != nil {
			http.Error(w, "error inserting user into database", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		// Generate an access token, expiry dates are in Unix time
		accessExpiresAt := time.Now().Add(DefaultAccessJWTExpiry)
		var accessToken string
		accessToken, err = setClaims(AuthClaims{
			UserID: userID, /*YOUR CODE HERE*/
			StandardClaims: jwt.StandardClaims{
				Subject:   "access",
				ExpiresAt: accessExpiresAt.Unix(),
				Issuer:    defaultJWTIssuer,
				IssuedAt:  time.Now().Unix(),
			},
		})

		// Check for error in generating an access token
		if err != nil {
			http.Error(w, "error generating access token", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		// Set the cookie, name it "access_token"
		http.SetCookie(w, &http.Cookie{
			Name:    "access_token",
			Value:   accessToken,
			Expires: accessExpiresAt,
			// Since our website does not use HTTPS, we have this commented out.
			// However, in an actual service you would definitely want this so no
			// cookies get stolen!
			//Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Path:     "/",
		})

		// Generate refresh token
		var refreshExpiresAt = time.Now().Add(DefaultRefreshJWTExpiry)
		var refreshToken string
		refreshToken, err = setClaims(AuthClaims{
			UserID: userID,
			StandardClaims: jwt.StandardClaims{
				Subject:   "refresh",
				ExpiresAt: refreshExpiresAt.Unix(),
				Issuer:    defaultJWTIssuer,
				IssuedAt:  time.Now().Unix(),
			},
		})

		if err != nil {
			http.Error(w, "error creating refreshToken", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		// Set the refresh token ("refresh_token") as a cookie
		http.SetCookie(w, &http.Cookie{
			Name:    "refresh_token",
			Value:   refreshToken,
			Expires: refreshExpiresAt,
			Path:    "/",
		})

		// Send verification email. Fill in the blank with the email of the user.
		err = m.SendEmail(c.Email /*YOUR CODE HERE*/, "Email Verification", "user-signup.html", map[string]interface{}{"Token": vertoken})
		if err != nil {
			http.Error(w, "error sending verification email", http.StatusInternalServerError)
			log.Print(err.Error())
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func signin(DB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Store the credentials in a instance of Credentials
		var c Credentials
		err := json.NewDecoder(r.Body).Decode(&c)
		// Check for errors in storing credntials
		if err != nil {
			http.Error(w, "error reading credentials", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		// Get the hashedPassword and userId of the user
		row := DB.QueryRow("SELECT userId, hashedPassword FROM users WHERE username = ?", c.Username)
		var userID string
		var corrPass string
		err = row.Scan(&userID, &corrPass)

		if err != nil {
			http.Error(w, "error querying database for user", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		thisPass, err := bcrypt.GenerateFromPassword([]byte(c.Password /*YOUR CODE HERE*/), bcrypt.DefaultCost)

		// Check for errors during hashing process
		if err != nil {
			http.Error(w, "error preparing password for verification", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		// Process errors associated with emails

		// Check if hashed password matches the one corresponding to the email
		err = bcrypt.CompareHashAndPassword([]byte(thisPass /*YOUR CODE HERE*/), []byte(corrPass /*YOUR CODE HERE*/))

		// Check error in comparing hashed passwords
		if err != nil {
			http.Error(w, "incorrect password", http.StatusBadRequest)
			return
		}

		// Generate an access token and set it as a cookie
		accessExpiresAt := time.Now().Add(DefaultAccessJWTExpiry)
		var accessToken string
		accessToken, err = setClaims(AuthClaims{
			UserID: userID,
			StandardClaims: jwt.StandardClaims{
				Subject:   "access",
				ExpiresAt: accessExpiresAt.Unix(),
				Issuer:    defaultJWTIssuer,
				IssuedAt:  time.Now().Unix(),
			},
		})

		//Check for error in generating an access token
		if err != nil {
			http.Error(w, "error creating accessToken", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		//Set the cookie, name it "access_token"
		http.SetCookie(w, &http.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			Expires:  accessExpiresAt,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Path:     "/",
		})

		// Generate a refresh token and set it as a cookie
		var refreshExpiresAt = time.Now().Add(DefaultRefreshJWTExpiry)
		var refreshToken string
		refreshToken, err = setClaims(AuthClaims{
			UserID: userID,
			StandardClaims: jwt.StandardClaims{
				Subject:   "refresh",
				ExpiresAt: refreshExpiresAt.Unix(),
				Issuer:    defaultJWTIssuer,
				IssuedAt:  time.Now().Unix(),
			},
		})

		if err != nil {
			http.Error(w, "error creating refreshToken", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "refresh_token",
			Value:   refreshToken,
			Expires: refreshExpiresAt,
			Path:    "/",
		})
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	// Set the access_token and refresh_token to have an empty value and set their expiration date to anytime in the past
	var expiresAt = time.Now().AddDate(-10, 1, 1) /*YOUR CODE HERE*/
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "", Expires: expiresAt})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Expires: expiresAt})
}

func verify(DB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		// Check that valid token exists
		if len(token) == 0 {
			http.Error(w, "url param 'token' is missing", http.StatusInternalServerError)
			log.Print("url param 'token' is missing")
			return
		}

		// Obtain the user with the verifiedToken from the query parameter and set their verification status to the integer "1"
		result, err := DB.Exec("UPDATE users SET verified = ? where verifiedToken = ?", 1, token)
		// Check for errors in executing the previous query
		if err != nil {
			http.Error(w, "error updating verification status", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		// Make sure there were some rows affected
		// Check: https://golang.org/pkg/database/sql/#Result
		// This is to make sure that there was an email that was actually changed by our query.
		// If no rows were affected return an error of type "StatusBadRequest"
		eff, err := result.RowsAffected()
		if eff == 0 {
			http.Error(w, "noone was verified", http.StatusBadRequest)
			return
		}
	}
}

func sendReset(m Mailer, DB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the email from the body (decode into an instance of Credentials)
		var c Credentials
		err := json.NewDecoder(r.Body).Decode(&c)
		// Check for errors decoding the object
		if err != nil {
			http.Error(w, "error reading credentials", http.StatusBadRequest)
			log.Print(err.Error())
			return
		}
		// Check for other miscallenous errors that may occur
		// What is considered an invalid input for an email?

		// Generate reset token
		token := GetRandomBase62(resetTokenSize)
		// Obtain the user with the specified email and set their resetToken to the token we generated
		result, err := DB.Exec("UPDATE users SET resetToken = ? where email = ?", token, c.Email)
		// Check for errors executing the queries
		if err != nil {
			http.Error(w, "error sending reset", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		eff, err := result.RowsAffected()
		if eff == 0 {
			http.Error(w, "reset not sent", http.StatusBadRequest)
			return
		}
		// Send verification email
		err = m.SendEmail(c.Email /*YOUR CODE HERE*/, "BearChat Password Reset", "password-reset.html", map[string]interface{}{"Token": token})
		if err != nil {
			http.Error(w, "error sending verification email", http.StatusInternalServerError)
			log.Print(err.Error())
		}
	}
}

func resetPassword(DB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from query params
		token := r.URL.Query().Get("token")
		// Get the username, email, and password from the body
		var c Credentials
		err := json.NewDecoder(r.Body).Decode(&c)
		// Check for errors decoding the body
		if err != nil {
			http.Error(w, "error reading credentials", http.StatusBadRequest)
			log.Print(err.Error())
			return
		}
		// Check for invalid inputs, return an error if input is invalid

		// Check if the username and token pair exist
		row, err := DB.Query("SELECT * FROM users WHERE username = ? AND resetToken = ?", c.Username, token)
		// Check for errors executing the query
		if err != nil {
			http.Error(w, "error querying database for user", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		// Check exists boolean. Call an error if the username-token pair doesn't exist
		if !row.Next() {
			http.Error(w, "Username or token invalid", http.StatusInternalServerError)
			return
		}
		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password /*YOUR CODE HERE*/), bcrypt.DefaultCost)

		// Check for errors in hashing the new password
		if err != nil {
			http.Error(w, "password preparation failed", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		// Input new password and clear the reset token (set the token equal to empty string)
		result, err := DB.Exec("UPDATE users SET hashedPassword = ?, resetToken = ? WHERE username = ?", hashedPassword, "", c.Username)
		if err != nil {
			http.Error(w, "error updating password", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}
		eff, err := result.RowsAffected()
		if eff == 0 {
			http.Error(w, "no password was updated", http.StatusBadRequest)
			return
		}
	}
}
