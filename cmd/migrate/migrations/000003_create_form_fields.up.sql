CREATE TABLE IF NOT EXISTS form_fields (
    id BIGSERIAL PRIMARY KEY,
    field_title VARCHAR(455) NOT NULL,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    form_id BIGINT NOT NULL,
    FOREIGN KEY(form_id) REFERENCES forms(id) ON DELETE CASCADE
);
