package storage

import (
	"database/sql"
	"fmt"
)

type User struct {
	Id        int     `json:"id"`
	Email     string  `json:"email"`
	Username  string  `json:"username"`
	Password  string  `json:"-"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

type UserStore struct {
	db *sql.DB
}

func (s *UserStore) GetUserById(userId int) (*User, error) {
	var user User
	query := `SELECT id,email,username,password,created_at,updated_at FROM users WHERE id=$1`
	row := s.db.QueryRow(query, userId)
	if err := row.Scan(&user.Id, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetUserByEmail(email string) (*User, error) {
	var user User
	query := `SELECT id,email,username,password,created_at,updated_at FROM users WHERE email=$1`
	row := s.db.QueryRow(query, email)
	if err := row.Scan(&user.Id, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetUserByUsername(username string) (*User, error) {
	var user User
	query := `SELECT id,email,username,password,created_at,updated_at FROM users WHERE username=$1`
	row := s.db.QueryRow(query, username)
	if err := row.Scan(&user.Id, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetUsersByUsernameOrEmail(username string, email string) ([]User, error) {
	var users []User
	query := `SELECT id,email,username,password,created_at,updated_at FROM users WHERE email=$1 OR username=$2`
	rows, err := s.db.Query(query, email, username)
	if err != nil {
		return []User{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var user User

		if err := rows.Scan(&user.Id, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return []User{}, err
		}

		users = append(users, user)
	}

	return users, nil

}

func (s *UserStore) CreateUser(username string, email string, hashedPassword string) (*User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("transaction failed to start :- %v", err.Error())
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var user User
	query := `INSERT INTO users(email,username,password) VALUES($1,$2,$3) RETURNING
	id,email,username,password,created_at,updated_at`

	row := tx.QueryRow(query, email, username, hashedPassword)
	if err := row.Scan(&user.Id, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("transaction failed to commit:- %v", err.Error())
	}

	return &user, nil
}
