package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	paramRepository = "repo"   // repo where the file is located
	paramId         = "id"     // a sha in Git, nodeid in Mercurial, etc.
	paramCommit     = "commit" // a commit
	paramPath       = "path"   // the file name
)

// getFile gets the contents of a file.
//
// If the request method is GET, this sends a response containing the file
// blob, if it exists. Otherwise, it will send a 404 Not Found. If the request
// method is HEAD, this sends a 200 OK response if the file exists. Otherwise,
// it will send a 404 Not Found.
//
// Files can be retrieved either by providing a id, or a commit and file path
// pair in the query parameters.
//
// ID URL: /file?repo=<repo>&id=<id>
// Commit and file path URL: /file?repo=<repo>&commit=<commit>&path=<path>
func getFile(w http.ResponseWriter, r *http.Request) {
	repoName := r.URL.Query().Get(paramRepository)
	id := r.URL.Query().Get(paramId)
	commit := r.URL.Query().Get(paramCommit)
	path := r.URL.Query().Get(paramPath)

	if len(repoName) != 0 &&
		(len(id) != 0 || (len(commit) != 0 && len(path) != 0)) {
		repo := GetRepository(repoName)
		if repo == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case "GET":
			blob, err := getFileBlob(id, commit, path, repo)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(blob)
			}
		case "HEAD":
			exists, err := getFileExists(id, commit, path, repo)
			if exists {
				w.WriteHeader(http.StatusOK)
			} else {
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository or ID not specified", http.StatusBadRequest)
	}
}

func getFileBlob(id, commit, path string, repo Repository) ([]byte, error) {
	if len(id) != 0 {
		return repo.GetFile(id)
	}

	return repo.GetFileByCommit(commit, path)
}

func getFileExists(id, commit, path string, repo Repository) (bool, error) {
	if len(id) != 0 {
		return repo.FileExists(id)
	}

	return repo.FileExistsByCommit(commit, path)
}

// getPath retrieves the path of a valid repository.
//
// URL: /path?repo=<repo>
func getPath(w http.ResponseWriter, r *http.Request) {
	repoName := r.URL.Query().Get(paramRepository)

	if len(repoName) != 0 {
		repo := GetRepository(strings.Split(repoName, "/")[0])
		if repo == nil {
			http.Error(w, "Repository not found", http.StatusBadRequest)
		}

		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(repo.GetPath() + "/info/refs"))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository not specified", http.StatusBadRequest)
	}
}

// getSession uses Basic Authentication to return a private session token based
// on the authentication information provided in the request header.
//
// This responds with a JSON of the Session, in the following format:
// { "private-token":<token> }
//
// URL: /path/session
func getSession(w http.ResponseWriter, r *http.Request) {
	session, err := CreateSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	json, err := json.Marshal(session)
	if err != nil {
		http.Error(w, "Session marshalling error",
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

// Route handles all the URL routing.
func Route() {
	http.HandleFunc("/file", BasicAuth(getFile))
	http.HandleFunc("/path", BasicAuth(getPath))
	http.HandleFunc("/session", getSession)
}