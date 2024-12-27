package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type CreateFormRequest struct {
	FormTitle       string `json:"form_title"`
	FormDescription string `json:"form_description"`
}

type CreateFormFieldRequest struct {
	FieldTitle string `json:"field_title"`
	Required   bool   `json:"required"`
	FormId     int    `json:"form_id"`
}

type UpdateFormFieldRequest struct {
	FieldTitle string `json:"field_title"`
	Required   bool   `json:"required"`
}

func (s *APIServer) getAllForms(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "unauthorized: unable to retrieve user from context", http.StatusUnauthorized)
		return
	}

	forms, err := s.storage.Forms.GetAllForms()
	if err != nil {
		s.writeJSONError(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	if err = s.writeJSON(w, forms, http.StatusOK); err != nil {
		s.writeJSONError(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)
	}
}

func (s *APIServer) createForm(w http.ResponseWriter, r *http.Request) {
	// Retrieve the authenticated user's userId from the request context
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "unauthorized: unable to retrieve user from context", http.StatusUnauthorized)
		return
	}

	// Decode the JSON body into the CreateFormRequest struct
	var req CreateFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, "bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate the request payload
	if strings.TrimSpace(req.FormTitle) == "" {
		s.writeJSONError(w, "bad request: form title is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.FormDescription) == "" {
		s.writeJSONError(w, "bad request: form description is required", http.StatusBadRequest)
		return
	}

	// Create the form using the storage layer
	form, err := s.storage.Forms.CreateForm(req.FormTitle, req.FormDescription, userId)
	if err != nil {
		s.writeJSONError(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with the created form
	if err := s.writeJSON(w, form, http.StatusCreated); err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	}
}

func (s *APIServer) myForms(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "unauthorized: unable to retrieve user from context", http.StatusUnauthorized)
		return
	}

	// Retrieve the forms for the authenticated user from the storage layer
	forms, err := s.storage.Forms.GetFormsByUserId(userId)
	if err != nil {
		s.writeJSONError(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with the list of forms
	if err := s.writeJSON(w, forms, http.StatusOK); err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	}
}

func (s *APIServer) createFormField(w http.ResponseWriter, r *http.Request) {
	// an authenticated user is making the request
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "invalid user id", http.StatusUnauthorized)
		return
	}
	var payload CreateFormFieldRequest
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		s.writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	fieldTitle := strings.TrimSpace(payload.FieldTitle)
	formId := payload.FormId
	isFieldRequired := payload.Required

	if fieldTitle == "" {
		s.writeJSONError(w, "field title cannot be empty", http.StatusBadRequest)
		return
	}

	// get form and check if form.user_id = userId , if not then logged in user cannot create field on this for
	form, err := s.storage.Forms.GetFormById(formId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, fmt.Sprintf("Form with id %d not found", formId), http.StatusNotFound)
			return
		} else {
			log.Println(err.Error())
			s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if userId != form.UserId {
		s.writeJSONError(w, "user unauthorized to make field on form", http.StatusUnauthorized)
		return
	}

	// ok so the form belongs to user making the request , and form exists
	// can create field for form now
	field, err := s.storage.FormFields.CreateFormField(fieldTitle, isFieldRequired, form.Id)
	if err != nil {
		log.Println(err.Error())
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	// once a field on a form is created or deleted , the is_ready fiels is updated if the count(*) from form_fields is > 0 where form_id=form.Id
	if err = s.storage.FormFields.UpdateFormIsReady(form.Id); err != nil {
		log.Println(err.Error())
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if err = s.writeJSON(w, field, http.StatusCreated); err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
	}
}

func (s *APIServer) deleteFormField(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "invalid user id", http.StatusUnauthorized)
		return
	}
	fieldId, err := strconv.ParseInt(r.PathValue("fieldId"), 10, 64)
	if err != nil {
		s.writeJSONError(w, "invalid request parameter", http.StatusBadRequest)
		return
	}

	formField, err := s.storage.FormFields.GetFormFieldById(int(fieldId))
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, fmt.Sprintf("Field with id %d not found", fieldId), http.StatusNotFound)
			return
		} else {
			s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
			return
		}
	}

	// form which we're trying to delete a field from
	form, err := s.storage.Forms.GetFormById(formField.FormId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, fmt.Sprintf("Form with id %d not found", formField.FormId), http.StatusNotFound)
			return
		} else {
			s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if userId != form.UserId {
		s.writeJSONError(w, "user not authorized to delete field on this form", http.StatusUnauthorized)
		return
	}

	err = s.storage.FormFields.DeleteFormFieldById(formField.Id)
	if err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	// update is_ready field
	err = s.storage.FormFields.UpdateFormIsReady(form.Id)
	if err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type Envelope struct {
		Message string `json:"message"`
	}
	if err = s.writeJSON(w, Envelope{Message: fmt.Sprintf("field with id %d deleted", fieldId)}, http.StatusOK); err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
	}
}

func (s *APIServer) updateFormField(w http.ResponseWriter, r *http.Request) {
	// Extract the authenticated user's ID
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	// Parse `fieldId` from the URL path
	fieldId, err := strconv.ParseInt(r.PathValue("fieldId"), 10, 64)
	if err != nil {
		s.writeJSONError(w, "invalid request parameter", http.StatusBadRequest)
		return
	}

	// Decode the JSON payload
	var payload UpdateFormFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	fieldTitle := strings.TrimSpace(payload.FieldTitle)
	isRequired := payload.Required

	// Validate the `fieldTitle` if it's part of the update
	if fieldTitle == "" {
		s.writeJSONError(w, "field title cannot be empty", http.StatusBadRequest)
		return
	}

	// Fetch the form field and its associated form
	formField, err := s.storage.FormFields.GetFormFieldById(int(fieldId))
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, fmt.Sprintf("field with id %d not found", fieldId), http.StatusNotFound)
			return
		}
		s.writeJSONError(w, "something went wrong while fetching form field", http.StatusInternalServerError)
		return
	}

	form, err := s.storage.Forms.GetFormById(formField.FormId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, fmt.Sprintf("form with id %d not found", formField.FormId), http.StatusNotFound)
			return
		}
		s.writeJSONError(w, "something went wrong while fetching form", http.StatusInternalServerError)
		return
	}

	// Verify the authenticated user owns the form
	if userId != form.UserId {
		s.writeJSONError(w, "user not authorized to update this field", http.StatusUnauthorized)
		return
	}

	updatedFormField, err := s.storage.FormFields.UpdateFormField(formField.Id, fieldTitle, isRequired)
	if err != nil {
		s.writeJSONError(w, "failed to update form field", http.StatusInternalServerError)
		return
	}

	if err = s.writeJSON(w, updatedFormField, http.StatusOK); err != nil {
		s.writeJSONError(w, "something went wrong while sending response", http.StatusInternalServerError)
	}
}

func (s *APIServer) getFormWithFields(w http.ResponseWriter, r *http.Request) {
	// parse formId from the path value
	// get form with the form fields using join between forms and form_fields
	formId, err := strconv.ParseInt(r.PathValue("formId"), 10, 64)
	if err != nil {
		s.writeJSONError(w, "invalid request parameter", http.StatusBadRequest)
		return
	}

	form, err := s.storage.Forms.GetFormById(int(formId))
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, fmt.Sprintf("form with id %d not found", formId), http.StatusNotFound)
			return
		} else {
			s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
			return
		}
	}

	// form that we're trying to get exists
	formWithFields, err := s.storage.Forms.GetFormByIdWithFieldsAndUser(form.Id)
	if err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if err = s.writeJSON(w, formWithFields, http.StatusOK); err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
	}
}

func (s *APIServer) deleteFormHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "user not authorized", http.StatusUnauthorized)
		return
	}

	formId, err := strconv.ParseInt(r.PathValue("formId"), 10, 64)
	if err != nil {
		s.writeJSONError(w, "invalid path parameter", http.StatusBadRequest)
		return
	}

	form, err := s.storage.Forms.GetFormById(int(formId))
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, fmt.Sprintf("form with id %d not found", formId), http.StatusNotFound)
			return
		} else {
			s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
			return
		}
	}

	// before deleting , check if form.UserId ==userId
	if form.UserId != userId {
		s.writeJSONError(w, "user not authorized to delete form", http.StatusUnauthorized)
		return
	}

	if err = s.storage.Forms.DeleteFormById(form.Id); err != nil {
		s.writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Envelope struct {
		Message string `json:"message"`
	}
	if err = s.writeJSON(w, Envelope{Message: fmt.Sprintf("form with id %d deleted", form.Id)}, http.StatusOK); err != nil {
		s.writeJSON(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
