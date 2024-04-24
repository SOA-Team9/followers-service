package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"followers-service.xws.com/model"
	"followers-service.xws.com/repo"
	"github.com/gorilla/mux"
)

type FollowsHandler struct {
	logger *log.Logger
	repo   *repo.FollowRepo
}

type KeyProduct struct{}

func NewFollowsHandler(l *log.Logger, r *repo.FollowRepo) *FollowsHandler {
	return &FollowsHandler{l, r}
}

func (f *FollowsHandler) FollowUser(rw http.ResponseWriter, r *http.Request) {
	follows := r.Context().Value(KeyProduct{}).(*model.Follow)
	f.logger.Println("Follows: ", follows)

	newFollow, err := f.repo.FollowUser(follows.FollowerID, follows.FollowedID)
	if err != nil {
		f.logger.Println("Error creating follow:", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Serialize the newFollow object into JSON
	followJSON, err := json.Marshal(newFollow)
	if err != nil {
		f.logger.Println("Error marshaling follow:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set the response content type to JSON
	rw.Header().Set("Content-Type", "application/json")

	// Write the serialized follow object to the response body
	rw.WriteHeader(http.StatusCreated)
	_, err = rw.Write(followJSON)
	if err != nil {
		f.logger.Println("Error writing follow response:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (f *FollowsHandler) UnfollowUser(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	followedId, err := strconv.Atoi(vars["followedId"])
	followingId, err := strconv.Atoi(vars["followingId"])

	follows := &model.Follow{FollowedID: followedId, FollowerID: followingId}

	err = f.repo.UnfollowUser(*follows)
	if err != nil {
		f.logger.Println("Error unfollowing user:", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (f *FollowsHandler) CheckFollow(rw http.ResponseWriter, r *http.Request) {
	follows := r.Context().Value(KeyProduct{}).(*model.Follow)
	f.logger.Println("Follows: ", follows)

	isFollowed, err := f.repo.CheckFollow(follows.FollowerID, follows.FollowedID)
	if err != nil {
		f.logger.Println("Error checking follow:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	if isFollowed {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("User is following"))
	} else {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("User is not following"))
	}
}

func (u *FollowsHandler) AddUser(rw http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(KeyProduct{}).(*model.User)
	u.logger.Println("User: ", user)

	err := u.repo.AddUser(user)
	if err != nil {
		u.logger.Println("Error creating user:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusCreated)
}

func (u *FollowsHandler) GetUserFollowing(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currentUserID, err := strconv.Atoi(vars["user_id"])
	if err != nil {
		u.logger.Printf("Expected integer, got: %d", currentUserID)
		http.Error(rw, "Unable to convert limit to integer", http.StatusBadRequest)
		return
	}

	u.logger.Println("Current user ID:", currentUserID)
	followingIDs, err := u.repo.GetUserFollowing(currentUserID)
	if err != nil {
		u.logger.Println("Error fetching user following:", err)
		return
	}

	if followingIDs == nil {
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(followingIDs); err != nil {
		u.logger.Println("Error encoding JSON response:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (u *FollowsHandler) GetUserFollowingIds(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currentUserID, err := strconv.Atoi(vars["user_id"])
	if err != nil {
		u.logger.Printf("Expected integer, got: %d", currentUserID)
		http.Error(rw, "Unable to convert limit to integer", http.StatusBadRequest)
		return
	}

	u.logger.Println("Current user ID:", currentUserID)
	followingIDs, err := u.repo.GetUserFollowingIds(currentUserID)
	if err != nil {
		u.logger.Println("Error fetching user following:", err)
		return
	}

	if followingIDs == nil {
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(followingIDs); err != nil {
		u.logger.Println("Error encoding JSON response:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (u *FollowsHandler) GetUserFollowers(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currentUserID, err := strconv.Atoi(vars["user_id"])
	if err != nil {
		u.logger.Printf("Expected integer, got: %d", currentUserID)
		http.Error(rw, "Unable to convert limit to integer", http.StatusBadRequest)
		return
	}

	u.logger.Println("Current user ID:", currentUserID)
	followingIDs, err := u.repo.GetUserFollowers(currentUserID)
	if err != nil {
		u.logger.Println("Error fetching user followers:", err)
		return
	}

	if followingIDs == nil {
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(followingIDs); err != nil {
		u.logger.Println("Error encoding JSON response:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (u *FollowsHandler) GetFollowingRecommendation(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	personID := vars["user_id"]
	personIDInt, err := strconv.Atoi(personID)
	if err != nil {
		http.Error(rw, "Invalid user ID", http.StatusBadRequest)
		return
	}
	reccommendationIds, err := u.repo.GetFollowRecommendations(personIDInt)
	if err != nil {
		u.logger.Println("Error fetching recommendations:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Convert recommendation IDs to JSON
	jsonRecommendations, err := json.Marshal(reccommendationIds)
	if err != nil {
		u.logger.Println("Error marshalling recommendation IDs:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.Write(jsonRecommendations)
}

func (m *FollowsHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		m.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func (f *FollowsHandler) MiddlewareFollowDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		person := &model.Follow{}
		err := person.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			f.logger.Fatal(err)
			return
		}
		ctx := context.WithValue(h.Context(), KeyProduct{}, person)
		h = h.WithContext(ctx)
		next.ServeHTTP(rw, h)
	})
}

func (u *FollowsHandler) MiddlewareUserDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		person := &model.User{}
		err := person.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			u.logger.Fatal(err)
			return
		}
		ctx := context.WithValue(h.Context(), KeyProduct{}, person)
		h = h.WithContext(ctx)
		next.ServeHTTP(rw, h)
	})
}
