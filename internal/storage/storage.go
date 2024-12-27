package storage

import "database/sql"

type Storage struct {
	Users interface {
		GetUserById(userId int) (*User, error)
		GetUserByEmail(email string) (*User, error)
		GetUserByUsername(username string) (*User, error)
		GetUsersByUsernameOrEmail(username string, email string) ([]User, error)
		CreateUser(username string, email string, hashedPassword string) (*User, error)
	}
	Forms interface {
		CreateForm(formTitle string, formDescription string, userId int) (*Form, error)
		GetFormsByUserId(userId int) ([]Form, error)
		GetFormById(formId int) (*Form, error)
		GetFormByIdWithFieldsAndUser(formId int) (*Form, error)
		DeleteFormById(formId int) error
	}
	FormFields interface {
		CreateFormField(fieldTitle string, isRequired bool, formId int) (*FormField, error)
		DeleteFormFieldById(fieldId int) error
		UpdateFormIsReady(formId int) error
		GetFormFieldById(fieldId int) (*FormField, error)
		UpdateFormField(fieldId int, fieldTitle string, isRequired bool) (*FormField, error)
		GetFormFieldsByFormId(formId int) ([]FormField, error)
	}
	FormResponse interface {
		CreateFormResponse(formId int, userId int) (*FormResponse, error)
		CreateResponseFields(formResponseId int, responseFields []struct {
			FieldValue  string
			FormFieldId int
		}) ([]ResponseField, error)
		GetFormResponsesByFormId(formId int) ([]FormResponse, error)
		GetFormResponseById(FormResponseId int) (*FormResponse, error)
		GetResponseFieldsByFormResponseId(formResponseId int) ([]ResponseField, error)
		GetFormResponsesByRespondentId(respondentId int) ([]FormResponse, error)
	}
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		Users:        &UserStore{db: db},
		Forms:        &FormStore{db: db},
		FormFields:   &FormFieldStore{db: db},
		FormResponse: &FormResponseStore{db: db},
	}
}
