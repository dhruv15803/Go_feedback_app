


CREATE TABLE IF NOT EXISTS response_fields (
    id BIGSERIAL PRIMARY KEY,
    field_value TEXT NOT NULL,
    form_response_id BIGINT NOT NULL,
    form_field_id BIGINT NOT NULL,
    FOREIGN KEY(form_response_id) REFERENCES form_responses(id) ON DELETE CASCADE,
    FOREIGN KEY(form_field_id) REFERENCES  form_fields(id) ON DELETE CASCADE
);

