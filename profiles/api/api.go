package api

// Some useful imports :^).
import (
	"database/sql"
	"encoding/json"
	_ "encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Like before, think about what methods would be appropriate for these routes.
func RegisterRoutes(router *mux.Router, db *sql.DB) {
	router.HandleFunc("/api/profile/{uuid}", getProfile(db)).Methods( /* YOUR CODE HERE */ )
	router.HandleFunc("/api/profile/{uuid}", updateProfile(db)).Methods( /* YOUR CODE HERE */ )
}

// Retrieves a Profile from the users database and returns it in the response as a JSON.
func getProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var prof Profile
		// Obtain the uuid from the url path and store it in a `uuid` variable
		// (Hint: mux.Vars())
		id := mux.Vars(r)["uuid"]
		// Query the database and store a matching Profile into a variable. What errors might go wrong here?

		row := db.QueryRow("SELECT * FROM users WHERE uuid = ?", id)
		if err := row.Scan(&prof.Firstname, &prof.Lastname, &prof.Email, &prof.UUID); err != nil {
			http.Error(w, "error fetching profile", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		// Encode fetched data as JSON and serve to client
		json.NewEncoder(w).Encode(prof)
	}
}

func updateProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Obtain the requested uuid from the url path and store it in a `uuid` variable
		id := mux.Vars(r)["uuid"]

		otherID, err := getUUID(w, r)
		if err != nil {
			log.Print(err.Error())
			return
		}
		// Obtain the UUID from the cookie. See jwt.go. What errors should you check for?
		// (Hint: What if the UUID from the cookie doesn't match the UUID in the request?)
		if id != otherID {
			http.Error(w, "error verifyinh user ids", http.StatusUnauthorized)
			return
		}
		// Decode	 the Request Body's JSON data into a profile variable. Make sure to check for errors!
		var prof Profile
		err = json.NewDecoder(r.Body).Decode(&prof)

		if err != nil {
			http.Error(w, "error reading credentials", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		// Insert the profile data into the users table.
		// (Hint: Make sure to use REPLACE INTO)

		result, err := db.Exec("REPLACE INTO users (firstName,lastName, email,uuid) VALUES (?,?,?,?)", prof.Firstname, prof.Lastname, prof.Email, id)
		// Check for errors in executing the previous query
		if err != nil {
			http.Error(w, "error updating profile", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		eff, err := result.RowsAffected()
		if eff == 0 {
			http.Error(w, "nothing was updated", http.StatusBadRequest)
			return
		}
	}
}
