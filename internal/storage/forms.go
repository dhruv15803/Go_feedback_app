package storage

import (
	"database/sql"
	"fmt"
)

type Form struct {
	Id              int         `json:"id"`
	FormTitle       string      `json:"form_title"`
	FormDescription string      `json:"form_description"`
	IsReady         bool        `json:"is_ready"`
	UserId          int         `json:"user_id"`
	CreatedAt       string      `json:"created_at"`
	User            *User       `json:"user"`
	FormFields      []FormField `json:"form_fields"`
}

type FormStore struct {
	db *sql.DB
}

func (fs *FormStore) CreateForm(formTitle string, formDescription string, userId int) (*Form, error) {
	// Start a transaction
	tx, err := fs.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}

	// Ensure rollback in case of an error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Query to insert a new form into the database
	query := `INSERT INTO forms (form_title, form_description,user_id) 
	          VALUES ($1, $2, $3) RETURNING id, form_title, form_description, is_ready, user_id, created_at`

	// Create a Form instance to store the result
	var form Form

	// Execute the query
	row := tx.QueryRow(query, formTitle, formDescription, userId)
	if err := row.Scan(&form.Id, &form.FormTitle, &form.FormDescription, &form.IsReady, &form.UserId, &form.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to insert form: %v", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return &form, nil
}

func (fs *FormStore) GetFormsByUserId(userId int) ([]Form, error) {
	query := `
		SELECT 
			f.id, f.form_title, f.form_description, f.is_ready, f.user_id, f.created_at,
			u.id, u.email, u.username, u.password, u.created_at, u.updated_at
		FROM forms AS f 
		INNER JOIN users AS u ON f.user_id = u.id 
		WHERE f.user_id = $1`

	rows, err := fs.db.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []Form

	for rows.Next() {
		var form Form
		var user User

		// Scan both form and user details into their respective structs
		if err := rows.Scan(
			&form.Id, &form.FormTitle, &form.FormDescription, &form.IsReady, &form.UserId, &form.CreatedAt,
			&user.Id, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, err
		}

		form.User = &user // Link the user to the form
		forms = append(forms, form)
	}

	return forms, nil
}

func (fs *FormStore) GetFormById(formId int) (*Form, error) {
	var form Form

	query := `SELECT id,form_title,form_description,
	is_ready,user_id,created_at FROM forms WHERE id=$1`

	row := fs.db.QueryRow(query, formId)
	if err := row.Scan(&form.Id, &form.FormTitle, &form.FormDescription, &form.IsReady, &form.UserId, &form.CreatedAt); err != nil {
		return nil, err
	}

	return &form, nil
}

func (fs *FormStore) GetFormByIdWithFieldsAndUser(formId int) (*Form, error) {

	form, err := fs.GetFormById(formId)
	if err != nil {
		return nil, err
	}

	var user User
	query1 := `SELECT id,email,username,password,created_at,updated_at FROM users WHERE id=$1`
	row := fs.db.QueryRow(query1, form.UserId)
	if err = row.Scan(&user.Id, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}

	form.User = &user

	// query fields form form_fields.form_id=formId
	query2 := `SELECT id,field_title,required,form_id FROM form_fields
	WHERE form_id=$1`
	rows, err := fs.db.Query(query2, form.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fields []FormField
	for rows.Next() {
		var field FormField
		if err = rows.Scan(&field.Id, &field.FieldTitle, &field.Required, &field.FormId); err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	form.FormFields = fields
	return form, nil
}

func (fs *FormStore) DeleteFormById(formId int) error {
	query := `DELETE FROM forms WHERE id=$1`
	result, err := fs.db.Exec(query, formId)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected < 1 {
		return fmt.Errorf("Form with id %d not deleted", formId)
	}
	return nil
}
