package api

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/SlotifyApp/slotify-backend/database"
	"go.uber.org/zap"
)

type UserRepositoryInterface interface {
	CreateUser(UserCreate) (User, error)
	// CheckUserExistsByID returns a boolean if
	// a user with the given id exists
	CheckUserExistsByID(int) (bool, error)
	DeleteUserByID(int) error
	GetUserByID(int) (User, error)
	// Get users by query parameters, if none match
	// return an empty array.
	GetUsersByQueryParams(GetUsersParams) (Users, error)
}

type UserRepository struct {
	logger *zap.SugaredLogger
	db     *sql.DB
}

func NewUserRepository(logger *zap.SugaredLogger, db *sql.DB) UserRepository {
	return UserRepository{
		logger: logger,
		db:     db,
	}
}

// check UserRepository conforms to the interface.
var _ UserRepositoryInterface = (*UserRepository)(nil)

func (ur UserRepository) CheckUserExistsByID(userID int) (bool, error) {
	var exists bool
	if err := ur.db.QueryRow("SELECT EXISTS(SELECT 1 FROM User WHERE id=?)", userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("user repository: error checking user existence: %w", err)
	}
	return exists, nil
}

func (ur UserRepository) CreateUser(uc UserCreate) (User, error) {
	var stmt *sql.Stmt
	var err error
	if stmt, err = ur.db.Prepare("INSERT INTO User (email, first_name, last_name) VALUES (?, ?, ?)"); err != nil {
		return User{}, fmt.Errorf("user repository: %s: %w", PrepareStmtFail, err)
	}
	defer database.CloseStmt(stmt, ur.logger)

	var res sql.Result
	if res, err = stmt.Exec(uc.Email, uc.FirstName, uc.LastName); err != nil {
		// database.IsSpecificMySQLError(err, mysqlerr.)
		return User{}, fmt.Errorf("user repository: %s: %w", QueryDBFail, err)
	}
	var rows int64
	if rows, err = res.RowsAffected(); err != nil {
		return User{}, fmt.Errorf("user repository fails to get rows affected: %w", err)
	}
	if rows != 1 {
		return User{}, fmt.Errorf("user repository affected rows after creating user is %d, should be 1: %w", rows, err)
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return User{}, fmt.Errorf("user repository failed to get last insert id: %w", err)
	}
	user := User{
		Id:        int(id),
		Email:     uc.Email,
		FirstName: uc.FirstName,
		LastName:  uc.LastName,
	}
	return user, nil
}

func (ur UserRepository) DeleteUserByID(userID int) error {
	var err error
	// Check existence of user first
	if _, err = ur.GetUserByID(userID); err != nil {
		return err
	}

	var stmt *sql.Stmt
	if stmt, err = ur.db.Prepare("DELETE FROM User WHERE id=?"); err != nil {
		return fmt.Errorf("user repository: %s: %w", PrepareStmtFail, err)
	}
	defer database.CloseStmt(stmt, ur.logger)

	var res sql.Result
	if res, err = stmt.Exec(userID); err != nil {
		return fmt.Errorf("user repository: %s: %w", QueryDBFail, err)
	}

	var rows int64
	if rows, err = res.RowsAffected(); err != nil {
		return fmt.Errorf("user repository fails to get rows affected: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("user repository affected rows after deleting user is %d, should be 1: %w", rows, err)
	}

	return nil
}

func (ur UserRepository) GetUserByID(userID int) (User, error) {
	var stmt *sql.Stmt
	var err error
	if stmt, err = ur.db.Prepare("SELECT * FROM User WHERE id=?"); err != nil {
		return User{}, fmt.Errorf("user repository: %s: %w", PrepareStmtFail, err)
	}
	defer database.CloseStmt(stmt, ur.logger)

	var user User
	if err = stmt.QueryRow(userID).Scan(&user.Id, &user.Email, &user.FirstName, &user.LastName); err != nil {
		return User{}, fmt.Errorf("user repository failed to scan: %w", err)
	}

	return user, nil
}

func (ur UserRepository) GetUsersByQueryParams(params GetUsersParams) (Users, error) {
	var conditions []string
	var args []any

	if params.Email != nil {
		conditions = append(conditions, "email=?")
		args = append(args, string(*params.Email))
	}

	if params.FirstName != nil {
		conditions = append(conditions, "first_name=?")
		args = append(args, *params.FirstName)
	}

	if params.LastName != nil {
		conditions = append(conditions, "last_name=?")
		args = append(args, *params.LastName)
	}

	query := "SELECT * FROM User"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	stmt, err := ur.db.Prepare(query)
	if err != nil {
		return Users{}, fmt.Errorf("user repository: %s: %w", PrepareStmtFail, err)
	}

	defer database.CloseStmt(stmt, ur.logger)

	var rows *sql.Rows
	if rows, err = stmt.Query(args...); err != nil {
		return Users{}, fmt.Errorf("user repository: %s: %w", QueryDBFail, err)
	}

	defer database.CloseRows(rows, ur.logger)

	users := Users{}
	for rows.Next() {
		var user User
		if err = rows.Scan(&user.Id, &user.Email, &user.FirstName, &user.LastName); err != nil {
			return Users{}, fmt.Errorf("user repository: failed to scan row: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return Users{}, fmt.Errorf("user repository: sql rows error: %w", err)
	}
	return users, nil
}
