package storage

import "database/sql"

type FormResponse struct {
	Id           int    `json:"id"`
	FormId       int    `json:"form_id"`
	RespondentId int    `json:"respondent_id"`
	SubmittedAt  string `json:"submitted_at"`
	Respondent   User   `json:"respondent"`
}

type ResponseField struct {
	Id             int       `json:"id"`
	FieldValue     string    `json:"field_value"`
	FormResponseId int       `json:"form_response_id"`
	FormFieldId    int       `json:"form_field_id"`
	FormField      FormField `json:"form_field"`
}

type FormResponseStore struct {
	db *sql.DB
}

func (s *FormResponseStore) GetFormResponsesByRespondentId(respondentId int) ([]FormResponse, error) {

	var formResponses []FormResponse

	query := `
	SELECT 
		fr.id,fr.form_id,fr.respondent_id,fr.submitted_at,
		u.id,u.email,u.username,u.password,u.created_at,u.updated_at
	FROM 
		form_responses AS fr INNER JOIN users AS u 
	ON 
		fr.respondent_id=u.id
	WHERE 
		fr.respondent_id=$1;`

	rows, err := s.db.Query(query, respondentId)
	if err != nil {
		return []FormResponse{}, err
	}

	for rows.Next() {
		var formResponse FormResponse
		var respondent User
		if err := rows.Scan(&formResponse.Id, &formResponse.FormId, &formResponse.RespondentId, &formResponse.SubmittedAt,
			&respondent.Id, &respondent.Email, &respondent.Username, &respondent.Password, &respondent.CreatedAt, &respondent.UpdatedAt); err != nil {
			return []FormResponse{}, err
		}
		formResponse.Respondent = respondent
		formResponses = append(formResponses, formResponse)
	}

	return formResponses, nil

}

func (s *FormResponseStore) CreateFormResponse(formId int, userId int) (*FormResponse, error) {

	var formResponse FormResponse
	query := `INSERT INTO form_responses(form_id,respondent_id) VALUES($1,$2) RETURNING id,form_id,respondent_id,submitted_at`
	row := s.db.QueryRow(query, formId, userId)

	if err := row.Scan(&formResponse.Id, &formResponse.FormId, &formResponse.RespondentId, &formResponse.SubmittedAt); err != nil {
		return nil, err
	}

	return &formResponse, nil
}

func (s *FormResponseStore) CreateResponseFields(formResponseId int, responseFields []struct {
	FieldValue  string
	FormFieldId int
}) ([]ResponseField, error) {

	tx, err := s.db.Begin()
	if err != nil {
		return []ResponseField{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`INSERT INTO response_fields(form_response_id,form_field_id,field_value) VALUES($1,$2,$3) RETURNING id,field_value,form_response_id,form_field_id`)
	if err != nil {
		return []ResponseField{}, err
	}

	defer stmt.Close()

	var result []ResponseField

	for _, respField := range responseFields {

		var responseField ResponseField
		row := stmt.QueryRow(formResponseId, respField.FormFieldId, respField.FieldValue)
		if err := row.Scan(&responseField.Id, &responseField.FieldValue, &responseField.FormResponseId, &responseField.FormFieldId); err != nil {
			return []ResponseField{}, err
		}
		result = append(result, responseField)

	}

	if err = tx.Commit(); err != nil {
		return []ResponseField{}, err
	}
	return result, nil
}

func (s *FormResponseStore) GetFormResponsesByFormId(formId int) ([]FormResponse, error) {

	var formResponses []FormResponse

	query :=
		`SELECT 
		fr.id,fr.form_id,fr.respondent_id,fr.submitted_at,
		u.id,u.email,u.username,u.password,u.created_at,u.updated_at  
	FROM 
		form_responses AS fr INNER JOIN users AS u ON fr.respondent_id=u.id
	WHERE fr.form_id=$1`

	rows, err := s.db.Query(query, formId)

	if err != nil {
		return []FormResponse{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var formResponse FormResponse
		var respondent User

		if err = rows.Scan(&formResponse.Id, &formResponse.FormId, &formResponse.RespondentId, &formResponse.SubmittedAt,
			&respondent.Id, &respondent.Email, &respondent.Username, &respondent.Password, &respondent.CreatedAt, &respondent.UpdatedAt); err != nil {
			return []FormResponse{}, err
		}

		formResponse.Respondent = respondent
		formResponses = append(formResponses, formResponse)
	}

	return formResponses, nil
}

func (s *FormResponseStore) GetFormResponseById(formResponseId int) (*FormResponse, error) {
	var formResponse FormResponse
	query := `SELECT id,form_id,respondent_id,submitted_at FROM form_responses WHERE id=$1`

	row := s.db.QueryRow(query, formResponseId)
	if err := row.Scan(&formResponse.Id, &formResponse.FormId, &formResponse.RespondentId, &formResponse.SubmittedAt); err != nil {
		return nil, err
	}

	return &formResponse, nil
}

func (s *FormResponseStore) GetResponseFieldsByFormResponseId(formResponseId int) ([]ResponseField, error) {

	var responseFields []ResponseField

	query := `
	SELECT 
		rf.id,rf.field_value,rf.form_response_id,
		rf.form_field_id,ff.id,ff.field_title,ff.required,ff.form_id 
	FROM 
		response_fields  AS rf INNER JOIN form_fields AS ff 
	ON rf.form_field_id=ff.id
	WHERE rf.form_response_id=$1`

	rows, err := s.db.Query(query, formResponseId)
	if err != nil {
		return []ResponseField{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var responseField ResponseField
		var formField FormField
		if err = rows.Scan(&responseField.Id, &responseField.FieldValue, &responseField.FormResponseId,
			&responseField.FormFieldId, &formField.Id, &formField.FieldTitle, &formField.Required, &formField.FormId); err != nil {
			return []ResponseField{}, err
		}
		responseField.FormField = formField
		responseFields = append(responseFields, responseField)
	}

	return responseFields, nil
}
