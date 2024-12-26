package storage

import (
	"database/sql"
	"fmt"
)

type FormField struct {
	Id         int    `json:"id"`
	FieldTitle string `json:"field_title"`
	Required   bool   `json:"required"`
	FormId     int    `json:"form_id"`
}

type FormFieldStore struct {
	db *sql.DB
}

func (s *FormFieldStore) CreateFormField(fieldTitle string, isRequired bool, formId int) (*FormField, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("transaction failed to start")
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var formField FormField
	query := `INSERT INTO form_fields(field_title,required,form_id)  
	VALUES($1,$2,$3) RETURNING id,field_title,required,form_id`
	row := tx.QueryRow(query, fieldTitle, isRequired, formId)
	if err := row.Scan(&formField.Id, &formField.FieldTitle, &formField.Required, &formField.FormId); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction")
	}
	// when a new form_field is added to a form
	// is_ready is updated to true when the form has more than 0 fields
	return &formField, nil
}

func (s *FormFieldStore) DeleteFormFieldById(fieldId int) error {
	query := `DELETE FROM form_fields WHERE id=$1`
	result, err := s.db.Exec(query, fieldId)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected < 1 {
		return fmt.Errorf("field with id %d not deleted", fieldId)
	}
	return nil
}

func (s *FormFieldStore) UpdateFormIsReady(formId int) error {
	query2 := `UPDATE forms 
	SET is_ready = (SELECT COUNT(*) > 0 FROM form_fields WHERE form_id=$1)
	WHERE id=$1`
	_, err := s.db.Exec(query2, formId)
	if err != nil {
		return err
	}
	return nil
}

func (s *FormFieldStore) GetFormFieldById(fieldId int) (*FormField, error) {
	var formField FormField
	query := `SELECT id,field_title,required,form_id FROM form_fields WHERE id=$1`
	row := s.db.QueryRow(query, fieldId)
	if err := row.Scan(&formField.Id, &formField.FieldTitle, &formField.Required, &formField.FormId); err != nil {
		return nil, err
	}
	return &formField, nil
}

func (s *FormFieldStore) UpdateFormField(fieldId int, fieldTitle string, isRequired bool) (*FormField, error) {
	query := `
        UPDATE form_fields
        SET field_title = $1, required = $2
        WHERE id = $3
        RETURNING id, field_title, required, form_id
    `
	row := s.db.QueryRow(query, fieldTitle, isRequired, fieldId)
	var field FormField
	if err := row.Scan(&field.Id, &field.FieldTitle, &field.Required, &field.FormId); err != nil {
		return nil, err
	}
	return &field, nil
}
