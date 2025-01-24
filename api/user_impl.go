package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/SlotifyApp/slotify-backend/database"
	"go.uber.org/zap"
)

// (GET /users) Get a user by query params.
func (s Server) GetUsers(w http.ResponseWriter, _ *http.Request, params GetUsersParams) {
	users, err := s.UserRepository.GetUsersByQueryParams(params)
	if err != nil {
		s.Logger.Error("user api: failed to get users", zap.Error(err))
		sendError(w, http.StatusInternalServerError, "user api: failed to get users")
		return
	}

	SetHeaderAndWriteResponse(w, http.StatusOK, users)
}

// (POST /users) Create a new user.
func (s Server) PostUsers(w http.ResponseWriter, r *http.Request) {
	var userBody PostUsersJSONRequestBody
	var err error
	defer func() {
		if err = r.Body.Close(); err != nil {
			s.Logger.Warn("could not close request body", zap.Error(err))
		}
	}()
	if err = json.NewDecoder(r.Body).Decode(&userBody); err != nil {
		errMsg := "failed to unmarshal request body correctly"
		s.Logger.Error(errMsg, zap.Object("body", userBody), zap.Error(err))
		sendError(w, http.StatusBadRequest, errMsg)
		return
	}

	var user User
	if user, err = s.UserRepository.CreateUser(userBody); err != nil {
		if database.IsDuplicateEntrySQLError(err) {
			s.Logger.Error("user api: user already exists", zap.Object("req_body", userBody), zap.Error(err))
			sendError(w, http.StatusBadRequest, fmt.Sprintf("user with email %s already exists", userBody.Email))
			return
		}
		s.Logger.Error("user api failed to create user", zap.Object("req_body", userBody), zap.Error(err))
		sendError(w, http.StatusInternalServerError, "user api failed to create user")
		return
	}

	SetHeaderAndWriteResponse(w, http.StatusCreated, user)
}

// (DELETE /users/{userID}) Delete a user by id.
func (s Server) DeleteUsersUserID(w http.ResponseWriter, _ *http.Request, userID int) {
	if err := s.UserRepository.DeleteUserByID(userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errMsg := fmt.Sprintf("user api: user with id(%d) doesn't exist", userID)
			s.Logger.Error(errMsg, zap.Int("userID", userID), zap.Error(err))
			sendError(w, http.StatusBadRequest, errMsg)
			return
		}
		s.Logger.Error("user api failed to delete user", zap.Int("userID", userID), zap.Error(err))
		sendError(w, http.StatusInternalServerError, "user api failed to delete user")
		return
	}
	SetHeaderAndWriteResponse(w, http.StatusOK, "user deleted successfully")
}

// (GET /users/{userID}) Get a user by id.
func (s Server) GetUsersUserID(w http.ResponseWriter, _ *http.Request, userID int) {
	var user User
	var err error
	if user, err = s.UserRepository.GetUserByID(userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			errMsg := fmt.Sprintf("user api: user with id(%d) doesn't exist", userID)
			s.Logger.Error(errMsg, zap.Int("userID", userID), zap.Error(err))
			sendError(w, http.StatusNotFound, errMsg)
		} else {
			errMsg := fmt.Sprintf("user api: failed to get user with id(%d)", userID)
			s.Logger.Error(errMsg, zap.Int("userID", userID), zap.Error(err))
			sendError(w, http.StatusBadRequest, errMsg)
		}
		return
	}

	SetHeaderAndWriteResponse(w, http.StatusOK, user)
}
