package api

// As usual, remove the underscore if you'd like to use the package.
// You may use any packages you'd like.
import (
	"database/sql"
	"encoding/json"
	_ "encoding/json"
	"log"
	"net/http"
	"strconv"
	_ "strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, db *sql.DB) {
	// Spicy regex on the path names to help with integers :^).
	router.HandleFunc("/api/posts/{startIndex:[0-9]+}", getFeed(db)).Methods(http.MethodGet /*YOUR CODE HERE*/)
	router.HandleFunc("/api/posts/{uuid}/{startIndex:[0-9]+}", getPosts(db)).Methods(http.MethodGet /*YOUR CODE HERE*/)
	router.HandleFunc("/api/posts/create", createPost(db)).Methods(http.MethodPost /*YOUR CODE HERE*/)
	router.HandleFunc("/api/posts/delete/{postID}", deletePost(db)).Methods(http.MethodDelete, http.MethodPost /*YOUR CODE HERE*/)
}

// Returns the earliest 25 posts made by the user with ID uuid starting from startIndex.
func getPosts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// YOUR CODE HERE
		var posts []Post
		ind, err := strconv.Atoi(mux.Vars(r)["startIndex"])
		id, err := getUUID(w, r)

		if err != nil {
			log.Print(err.Error())
			return
		}

		rows, err := db.Query("SELECT * FROM posts WHERE authorID = ? ORDER BY postTime ASC", id)
		i := 0
		if err != nil {
			http.Error(w, "error querying database", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		for rows.Next() {
			if i < ind {
				i += 1
				continue
			}
			if i > ind+25 {
				break
			}
			var p Post
			if err := rows.Scan(&p.PostBody, &p.PostID, &p.AuthorID, &p.PostTime); err != nil {
				http.Error(w, "error reading from database", http.StatusInternalServerError)
				log.Print(err.Error())
				return
			}
			posts = append(posts, p)
			i += 1
		}

		err = rows.Close()
		e := rows.Err()
		if err != nil || e != nil {
			http.Error(w, "error reading from database", http.StatusInternalServerError)
			log.Print(e.Error())
			return
		}

		json.NewEncoder(w).Encode(posts)
	}
}

// Given a JSON containing a field called `postBody` that contains a message (make sure to error check!),
// adds the post to the database with the UUID of the author (which can be found using getUUID),
// a unique ID, and the timestamp of the post.
func createPost(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// YOUR CODE HERE
		var cont string
		err := json.NewDecoder(r.Body).Decode(&cont)

		if err != nil {
			http.Error(w, "error reading postBody", http.StatusBadRequest)
			log.Print(err.Error())
			return
		}

		id, err := getUUID(w, r)

		if err != nil {
			log.Print(err.Error())
			return
		}

		result, err := db.Exec("INSERT INTO posts VALUES (?,?,?,?)", cont, uuid.NewString(), id, time.Now())

		if err != nil {
			http.Error(w, "error inserting post into database", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		rows, _ := result.RowsAffected()

		if rows == 0 {
			http.Error(w, "error inserting post into database", http.StatusInternalServerError)
			return
		}
	}
}

// Given the ID of a post, removes the post from the database if the person requesting
// is the author of the post.
func deletePost(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// YOUR CODE HERE
		id, err := getUUID(w, r)
		if err != nil {
			log.Print(err.Error())
			return
		}

		postId := mux.Vars(r)["postID"]

		result, err := db.Exec("DELETE FROM posts WHERE postID = ? AND authorID = ?", postId, id)

		if err != nil {
			http.Error(w, "error deleting post from database", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		rows, _ := result.RowsAffected()

		if rows == 0 {
			http.Error(w, "no post was deleted", http.StatusBadRequest)
			return
		}

	}
}

// Similar to getPosts except it gets the posts of everyone else *except* the author.
func getFeed(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// YOUR CODE HERE
		var posts []Post
		ind, err := strconv.Atoi(mux.Vars(r)["startIndex"])
		id, err := getUUID(w, r)

		if err != nil {
			log.Print(err.Error())
			return
		}

		rows, err := db.Query("SELECT * FROM posts WHERE authorID <> ? ORDER BY postTime ASC", id)
		i := 0
		if err != nil {
			http.Error(w, "error querying database", http.StatusInternalServerError)
			log.Print(err.Error())
			return
		}

		for rows.Next() {
			if i < ind {
				i += 1
				continue
			}
			if i > ind+25 {
				break
			}
			var p Post
			if err := rows.Scan(&p.PostBody, &p.PostID, &p.AuthorID, &p.PostTime); err != nil {
				http.Error(w, "error reading from database", http.StatusInternalServerError)
				log.Print(err.Error())
				return
			}
			posts = append(posts, p)
			i += 1
		}

		err = rows.Close()
		e := rows.Err()
		if err != nil || e != nil {
			http.Error(w, "error reading from database", http.StatusInternalServerError)
			log.Print(e.Error())
			return
		}

		json.NewEncoder(w).Encode(posts)
	}
}
