package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dhruv15803/internal/storage"
)

type CreateFormResponseRequest struct {
	FormId         int             `json:"form_id"`
	ResponseFields []ResponseField `json:"response_fields"`
}

type ResponseField struct {
	FieldValue  string `json:"field_value"`
	FormFieldId int    `json:"form_field_id"`
}

func (s *APIServer) createFormResponse(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "invalid user id", http.StatusUnauthorized)
		return
	}
	var req CreateFormResponseRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	form, err := s.storage.Forms.GetFormById(req.FormId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, "form not found", http.StatusNotFound)
			return
		}
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if !form.IsReady {
		s.writeJSONError(w, "form is not ready to accept responses", http.StatusBadRequest)
		return
	}

	formFields, err := s.storage.FormFields.GetFormFieldsByFormId(form.Id)
	if err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	validFieldIDs := make(map[int]bool)
	for _, field := range formFields {
		validFieldIDs[field.Id] = true
	}

	for _, respField := range req.ResponseFields {
		if _, exists := validFieldIDs[respField.FormFieldId]; !exists {
			s.writeJSONError(w, "invalid field id", http.StatusBadRequest)
			return
		}
	}

	responseFields := []struct {
		FieldValue  string
		FormFieldId int
	}{}

	for _, respField := range req.ResponseFields {
		responseFields = append(responseFields, struct {
			FieldValue  string
			FormFieldId int
		}{
			FieldValue:  respField.FieldValue,
			FormFieldId: respField.FormFieldId,
		})
	}

	formResponse, err := s.storage.FormResponse.CreateFormResponse(form.Id, userId)
	if err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	createdFields, err := s.storage.FormResponse.CreateResponseFields(formResponse.Id, responseFields)
	if err != nil {
		s.writeJSONError(w, "something went wrong while saving response fields", http.StatusInternalServerError)
		return
	}

	resp := struct {
		FormResponseId int                     `json:"form_response_id"`
		ResponseFields []storage.ResponseField `json:"response_fields"`
	}{
		FormResponseId: formResponse.Id,
		ResponseFields: createdFields,
	}

	if err = s.writeJSON(w, resp, http.StatusCreated); err != nil {
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

}

func (s *APIServer) getFormResponses(w http.ResponseWriter, r *http.Request) {
	// can only read responses if u created the form
	// /{formId}
	// parse form id from r.PathValue
	// get form
	// check if  userId==form.UserId
	// query for responses
	formId, err := strconv.ParseInt(r.PathValue("formId"), 10, 64)
	if err != nil {
		s.writeJSONError(w, "invalid form ID", http.StatusBadRequest)
		return
	}

	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "invalid user ID", http.StatusUnauthorized)
		return
	}

	form, err := s.storage.Forms.GetFormById(int(formId))
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, "form not found", http.StatusNotFound)
			return
		}
		s.writeJSONError(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if form.UserId != userId {
		s.writeJSONError(w, "unauthrorized access to form responses", http.StatusUnauthorized)
		return
	}

	formResponses, err := s.storage.FormResponse.GetFormResponsesByFormId(form.Id)
	if err != nil {
		s.writeJSONError(w, "something went wrong while retrieving form responses", http.StatusInternalServerError)
		return
	}

	if err := s.writeJSON(w, formResponses, http.StatusOK); err != nil {
		s.writeJSONError(w, "something went wrong while writing response", http.StatusInternalServerError)
	}
}

func (s *APIServer) getResponseFields(w http.ResponseWriter, r *http.Request) {

	formResponseId, err := strconv.ParseInt(r.PathValue("formResponseId"), 10, 64)

	if err != nil {
		s.writeJSONError(w, "invalid form response id", http.StatusBadRequest)
		return
	}

	userId, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		s.writeJSONError(w, "invalid user ID", http.StatusUnauthorized)
		return
	}

	// can only read  form responses if the form belongs to u
	// so can only read form response fields for the same
	formResponse, err := s.storage.FormResponse.GetFormResponseById(int(formResponseId))
	if err != nil {
		s.writeJSONError(w, fmt.Sprintf("response with id %d not found", formResponseId), http.StatusNotFound)
		return
	}
	formId := formResponse.FormId
	form, err := s.storage.Forms.GetFormById(formId)
	if err != nil {
		s.writeJSONError(w, "form that you're responding to not found", http.StatusNotFound)
		return
	}

	if form.UserId != userId {
		s.writeJSONError(w, "user not authorized to read form responses", http.StatusUnauthorized)
		return
	}

	// form is user's so responses to the form can be read

	responseFields, err := s.storage.FormResponse.GetResponseFieldsByFormResponseId(formResponse.Id)
	if err != nil {
		s.writeJSONError(w, "failed to fetch response fields", http.StatusInternalServerError)
		return
	}

	if err := s.writeJSON(w, responseFields, http.StatusOK); err != nil {
		s.writeJSONError(w, "failed to write response", http.StatusInternalServerError)
	}
}
